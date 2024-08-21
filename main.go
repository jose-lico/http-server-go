package main

import (
	"fmt"
	"net/http"

	"github.com/jose-lico/http-server-go/server"
)

func main() {
	s := server.NewServer()

	// curl -v "http://localhost:8000"
	// curl -v "http://localhost:8000/?name=Joe&pets=Turtle&pets=Dog"
	s.Get("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for key, values := range r.URL.Query() {
			fmt.Printf("Key: %s	| Value(s): %v\n", key, values)
		}

		fmt.Fprintln(w, "Hello world!")
	}))

	// curl -v -X POST "http://localhost:8000/create" -d "name=Joe&last_name=Mama"
	s.Post("/create", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			http.Error(w, "Unable to parse form", http.StatusBadRequest)
			return
		}

		name := r.FormValue("name")
		lastName := r.FormValue("last_name")

		w.WriteHeader(http.StatusCreated)
		fmt.Fprintf(w, "Created user %s %s\n", name, lastName)
	}))

	fmt.Println("Starting server on port 8000")
	s.ListenAndServe("localhost:8000")
}
