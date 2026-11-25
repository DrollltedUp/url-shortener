package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/drollllted/url-shortener/internal/db"
	"github.com/drollllted/url-shortener/internal/server"
	"github.com/drollllted/url-shortener/internal/service"
)

type Config struct {
	addr     string
	BaseURL  string
	LogLevel slog.Level
	LogJSON  bool
}

func loadConfig() Config {
	return Config{
		addr:     getEnv("ADDR", ":9091"),
		BaseURL:  getEnv("BASEURL", "http://localhost:9091"),
		LogLevel: parseLevel(getEnv("LOG_LEVEL", "info")),
		LogJSON:  strings.EqualFold(getEnv("LOG_FORMAT", "text"), "json"),
	}
}

func parseLevel(s string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func newLogger(level slog.Level, json bool) *slog.Logger {
	opts := &slog.HandlerOptions{Level: level}
	var h slog.Handler
	if json {
		h = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		h = slog.NewTextHandler(os.Stdout, opts)
	}
	return slog.New(h)
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}

	return fallback
}

func main() {
	if err := run(); err != nil {
		slog.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg := loadConfig()
	log := newLogger(cfg.LogLevel, cfg.LogJSON)
	slog.SetDefault(log)

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Error("DATABASE_URL not set")
	}

	pool, err := db.NewPool(context.Background(), dbURL)
	if err != nil {
		log.Error("Cannot connect to database", "err", err)
	}

	defer pool.Close()

	st := db.NewURLMemory(pool)
	svc := service.NewShortener(st, service.RandomGeneration{})
	srv := server.NewServer(svc, log, cfg.BaseURL, pool)

	httpServer := &http.Server{
		Addr:              cfg.addr,
		Handler:           srv.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		log.Info("server started", "addr", cfg.addr, "base_url", cfg.BaseURL)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		log.Info("shutting Down")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return httpServer.Shutdown(shutdownCtx)
}
