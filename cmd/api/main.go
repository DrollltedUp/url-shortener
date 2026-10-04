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

// var store = map[string]string{}
var stores = store.NewStore()

func main() {
	http.HandleFunc("POST /shorten", shorten)
	http.HandleFunc("GET /{code}", getCode)

	log.Fatal(http.ListenAndServe(addr, nil))
}

// Functions for Call

func shorten(w http.ResponseWriter, r *http.Request) {
	var request Request
	var shortener service.Shortener
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "bad request")
		return
	}

	code, err := shortener.Shorten(request.URL)
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

	// if !isValidURL(request.URL) {
	// 	writeError(w, http.StatusBadRequest, "bad URL")
	// 	return
	// }

	// for i := 0; i <= 5; i++ {
	// 	code := generateCode()
	// 	err := stores.Save(code, request.URL)
	// 	if err != nil {
	// 		if errors.Is(err, store.ErrCodeTaken) {
	// 			writeError(w, http.StatusConflict, "Already taken")
	// 		} else {
	// 			writeError(w, http.StatusInternalServerError, "Internal error")
	// 		}
	// 		return
	// 	} else {
	// 		writeJSON(w, http.StatusCreated, map[string]string{
	// 			"short_url": baseURL + "/" + code,
	// 		})
	// 		return
	// 	}
	// }

}

func getCode(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	target, ok := stores.Get(code)
	if !ok {
		writeError(w, http.StatusNotFound, "Не найдено")
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
