package main

import (
	"log"
	"net/http"
)

// Const

const addr = ":9091"

// Struct main
type Request struct {
	URL string `json:"url"`
}

func main() {
	log.Fatal(http.ListenAndServe(addr, nil))
}
