package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/drollllted/url-shortener/internal/service"
)

const baseURL = "http://localhost:9091"

type errorDTO struct {
	Message string    `json:"message"`
	Time    time.Time `json:"time"`
}

type Server struct {
	svc     *service.Shortener
	baseURL string
}

type Request struct {
	URL string `json:"url"`
}

func NewServer(svc *service.Shortener, baseURL string) *Server {
	return &Server{
		svc:     svc,
		baseURL: baseURL,
	}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /shorten", s.shorten)
	mux.HandleFunc("GET /{code}", s.getCode)

	return mux
}

func (s *Server) shorten(w http.ResponseWriter, r *http.Request) {
	var request Request
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "bad request")
		return
	}

	code, err := s.svc.Shorten(request.URL)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidURL):
			writeError(w, http.StatusBadRequest, "bad URL")
		default:
			writeError(w, http.StatusInternalServerError, "internal error")
		}
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"short_url": baseURL + "/" + code})

}

func (s *Server) getCode(w http.ResponseWriter, r *http.Request) {
	target, err := s.svc.Resolve(r.PathValue("code"))
	//shorteners.Resolve(r.PathValue("code"))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrNotFound):
			writeError(w, http.StatusNotFound, "not found")
		default:
			writeError(w, http.StatusInternalServerError, "internal error")
		}
		return
	}
	http.Redirect(w, r, target, http.StatusFound)
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
