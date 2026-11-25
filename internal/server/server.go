package server

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/drollllted/url-shortener/internal/service"
)

const maxBodyBytes = 4 << 10
const readyzTimeout = 2 * time.Second

type errorDTO struct {
	Message string    `json:"message"`
	Time    time.Time `json:"time"`
}

type Pinger interface {
	Ping(ctx context.Context) error
}

type Server struct {
	svc     *service.Shortener
	log     *slog.Logger
	baseURL string
	pinger  Pinger
}

type Request struct {
	URL string `json:"url"`
}

func NewServer(svc *service.Shortener, log *slog.Logger, baseURL string, pinger Pinger) *Server {
	return &Server{
		svc:     svc,
		log:     log,
		baseURL: baseURL,
		pinger:  pinger,
	}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /shorten", s.shorten)
	mux.HandleFunc("GET /{code}", s.getCode)
	mux.HandleFunc("GET /health", s.healthHandle)
	mux.HandleFunc("GET /readyz", s.readyzHandle)

	return s.logRequest(s.recovery(mux))
}

func (s *Server) shorten(w http.ResponseWriter, r *http.Request) {

	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	var request Request
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			writeError(w, http.StatusRequestEntityTooLarge, "request too large")
			return
		}
		writeError(w, http.StatusBadRequest, "bad request")
		return
	}

	code, err := s.svc.Shorten(r.Context(), request.URL)
	if err != nil {
		s.handlerError(w, r, err)
		return
	}
	s.log.InfoContext(r.Context(), "link created", "code", code)
	writeJSON(w, http.StatusCreated, map[string]string{"short_url": s.baseURL + "/" + code})

}

func (s *Server) getCode(w http.ResponseWriter, r *http.Request) {
	target, err := s.svc.Resolve(r.Context(), r.PathValue("code"))
	if err != nil {
		s.handlerError(w, r, err)
		return
	}
	http.Redirect(w, r, target, http.StatusFound)
}

func (s *Server) healthHandle(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) readyzHandle(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), readyzTimeout)
	defer cancel()

	if s.pinger == nil {
		writeError(w, http.StatusServiceUnavailable, "readiness not configured")
		return
	}

	if err := s.pinger.Ping(ctx); err != nil {
		s.log.WarnContext(r.Context(), "readiness check failed", "request_id", requestID(r.Context()), "err", err)
		writeError(w, http.StatusServiceUnavailable, "database unavailable")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handlerError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidURL):
		writeError(w, http.StatusBadRequest, "bad url")
	case errors.Is(err, service.ErrNotFound):
		writeError(w, http.StatusNotFound, "not found")
	default:
		s.log.ErrorContext(r.Context(), "request failed", "request_id", requestID(r.Context()), "method", r.Method, "path", r.URL.Path, "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
	}
}

// Functions Other

func writeJSON(w http.ResponseWriter, statusCode int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(v)
}

// Functions for Errors

func writeError(w http.ResponseWriter, statusCode int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(errorDTO{Message: msg, Time: time.Now()})
}
