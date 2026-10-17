package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/drollllted/url-shortener/internal/server"
	"github.com/drollllted/url-shortener/internal/service"
	"github.com/drollllted/url-shortener/internal/store"
)

type Config struct {
	addr    string
	BaseURL string
}

func loadConfig() Config {
	return Config{
		addr:    getEnv("ADDR", ":9091"),
		BaseURL: getEnv("BASEURL", "http://localhost:9091"),
	}
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
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))

	st := store.NewStore()
	svc := service.NewShortener(st, service.RandomGeneration{})
	srv := server.NewServer(svc, cfg.BaseURL)

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
		if err := httpServer.ListenAndServe(); err != nil && errors.Is(err, http.ErrServerClosed) {
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
