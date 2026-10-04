package main

import (
	"encoding/json"
	"log"
	"math/rand/v2"
	"net/http"
	"net/url"
	"time"
)

// Const

const baseURL = "http://localhost:9091"
const addr = ":9091"
const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// Struct main
type Request struct {
	URL string `json:"url"`
}

// Error types

type errorDTO struct {
	Message string    `json:"message"`
	Time    time.Time `json:"time"`
}

func (e errorDTO) ToString() string {
	b, err := json.MarshalIndent(e, "", "    ")
	if err != nil {
		panic(err)
	}

	return string(b)
}

var store = map[string]string{}

func main() {
	http.HandleFunc("POST /shorten", shorten)
	http.HandleFunc("GET /{code}", getCode)

	log.Fatal(http.ListenAndServe(addr, nil))
}

// Functions for Call

func shorten(w http.ResponseWriter, r *http.Request) {
	var request Request
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "bad request")
		return
	}

	if !isValidURL(request.URL) {
		writeError(w, http.StatusBadRequest, "bad URL")
		return
	}

	code := generateCode()
	store[code] = request.URL

	writeJSON(w, http.StatusCreated, map[string]string{
		"short_url": baseURL + "/" + code,
	})

}

func getCode(w http.ResponseWriter, r *http.Request) {

}

// Functions Other

func isValidURL(raw string) bool {
	u, err := url.ParseRequestURI(raw)
	if err != nil {
		return false
	}

	if (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return false
	}

	return true
}

func generateCode() string {
	b := make([]byte, 6)
	for i := range b {
		b[i] = alphabet[rand.IntN(len(alphabet))]
	}

	return string(b)
}

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
