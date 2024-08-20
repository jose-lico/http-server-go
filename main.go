package main

import (
	"fmt"
	"net/http"

	"github.com/jose-lico/http-server-go/server"
)

func main() {
	s := server.NewServer()
	// curl -v http://localhost:8000
	s.Get("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Hello world!")
	}))

	// curl -v -X POST http://localhost:8000/create -d "name=Joe&last_name=Mama"
	s.Post("/create", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, "Created an object!")
	}))

	fmt.Println("Starting server on port 8000")
	s.ListenAndServe("localhost:8000")

	http.NewServeMux()
}
