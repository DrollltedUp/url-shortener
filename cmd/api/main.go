package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/drollllted/url-shortener/internal/service"
	"github.com/drollllted/url-shortener/internal/store"
)

// Const

const baseURL = "http://localhost:9091"
const addr = ":9091"

// Struct main
type Request struct {
	URL string `json:"url"`
}

// Error types

type errorDTO struct {
	Message string    `json:"message"`
	Time    time.Time `json:"time"`
}

var stores = store.NewStore()
var shorteners = service.NewShortener(stores, service.RandomGeneration{})

func main() {
	http.HandleFunc("POST /shorten", shorten)
	http.HandleFunc("GET /{code}", getCode)

	log.Fatal(http.ListenAndServe(addr, nil))
}

func shorten(w http.ResponseWriter, r *http.Request) {
	var request Request
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "bad request")
		return
	}

	code, err := shorteners.Shorten(request.URL)
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

func getCode(w http.ResponseWriter, r *http.Request) {
	target, err := shorteners.Resolve(r.PathValue("code"))
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
