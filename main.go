package main

import (
	"fmt"
	"net/http"

	"github.com/jose-lico/http-server-go/server"
)

func main() {
	s := server.NewServer()
	s.Get("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "Hello, world!")
	}))

	s.Post("/create", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, "Created an object!")
	}))

	s.ListenAndServe("localhost:8000")
}
