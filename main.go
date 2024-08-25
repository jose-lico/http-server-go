package main

import (
	"fmt"
	"net/http"

	"github.com/jose-lico/http-server-go/server"
)

func main() {
	s := server.NewServer()

	// Query params

	// curl -v "http://localhost:8000"
	// curl -v "http://localhost:8000/?name=Joe&pets=Turtle&pets=Dog"
	s.Get("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for key, values := range r.URL.Query() {
			fmt.Printf("Key: %s	| Value(s): %v\n", key, values)
		}

		fmt.Fprintln(w, "Hello world!")
	}))

	// Support for wildcards

	// curl -v "http://localhost:8000/echo/ligma"
	s.Get("/echo/{echo}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		echo := r.PathValue("echo")

		fmt.Fprintln(w, echo)
	}))

	// curl -v "http://localhost:8000/echo/Hello/reverse/World"
	s.Get("/echo/{echo}/reverse/{echo2}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reverseString := func(s string) string {
			runes := []rune(s)
			for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
				runes[i], runes[j] = runes[j], runes[i]
			}
			return string(runes)
		}

		echo := reverseString(r.PathValue("echo"))
		echo2 := reverseString(r.PathValue("echo2"))

		fmt.Fprintln(w, echo, echo2)
	}))

	// TODO: Support wildcard as initial part of URL

	// curl -v "http://localhost:8000/123"
	// curl -v "http://localhost:8000/321"
	// s.Get("/{id}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	// 	id := r.PathValue("id")

	// 	fmt.Println(id)

	// 	if id == "123" {
	// 		fmt.Fprintln(w, "User Found!")
	// 	} else {
	// 		w.WriteHeader(http.StatusNotFound)
	// 		fmt.Fprintln(w, "User Not Found!")
	// 	}
	// }))

	// Same route, different method

	// curl -v -X POST "http://localhost:8000"
	s.Post("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Hello world from a post!")
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
