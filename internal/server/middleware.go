package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"
)

type ctxKey struct{}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func newRequestID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func requestID(ctx context.Context) string {
	id, _ := ctx.Value(ctxKey{}).(string)
	return id
}

func (rw *responseWriter) Unwrap() http.ResponseWriter {
	return rw.ResponseWriter
}

// Middleware

func (s *Server) logRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		id := newRequestID()
		w.Header().Set("X-Request-Id", id)
		r = r.WithContext(context.WithValue(r.Context(), ctxKey{}, id))

		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rw, r)

		level := slog.LevelInfo
		switch {
		case rw.status >= 500:
			level = slog.LevelError
		case rw.status >= 400:
			level = slog.LevelWarn
		case r.URL.Path == "/health":
			level = slog.LevelDebug
		case r.URL.Path == "/health" || (r.URL.Path == "/readyz" && rw.status == http.StatusOK):
			level = slog.LevelDebug

		}
		s.log.Log(r.Context(), level, "request", "request_id", id, "method", r.Method, "path", r.URL.Path, "status", rw.status,
			"duration_ms", time.Since(start).Milliseconds())
	})
}

func (s *Server) recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			rec := recover()
			if rec == nil {
				return
			}
			if rec == http.ErrAbortHandler {
				panic(rec)
			}

			s.log.ErrorContext(r.Context(), "panic recovered",
				"request_id", requestID(r.Context()),
				"method", r.Method,
				"path", r.URL.Path,
				"panic", rec,
				"stack", string(debug.Stack()),
			)
			writeError(w, http.StatusInternalServerError, "internal error")
		}()

		next.ServeHTTP(w, r)
	})
}
