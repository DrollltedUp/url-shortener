package http

import (
	"net/http"

	"github.com/gorilla/mux"
)

type HTTPServer struct {
}

func NewHTTPServer(httpHandlers *HTTPHandlers) *HTTPServer {
	return &HTTPServer{}

}

func StartServer() error {
	router := mux.NewRouter()

	// router.Path("/short").Methods("POST").HandlerFunc()
	// router.Path("/code/{code}").Methods("GET").HandlerFunc()

	return http.ListenAndServe(":9091", router)
}
