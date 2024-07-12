package main

import (
	"fmt"
	"net/http"

	"github.com/jose-lico/http-server-go/server"
)

func main() {
	s := server.NewServer()
	s.Get("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Hello, world!")
	}))

	s.ListenAndServe("localhost:8000")

	// mux := http.NewServeMux()
	// mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	// 	fmt.Fprintln(w, "Hello, world!")
	// }))

	// http.ListenAndServe("localhost:8000", mux)
}
