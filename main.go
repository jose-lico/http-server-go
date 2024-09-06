package main

import (
	"fmt"
	"io"
	"net/http"

	"github.com/jose-lico/http-server-go/server"
)

// curl requests to test server and enpoints

// Error states

// Send a large header to trigger a `431 Request Header Fields Too Large` status
// curl -v http://localhost:8000/ -H "X-Large-Header: $(head -c 1500 </dev/urandom | base64)" -d "This is a test body"

// Send a large body to trigger a `413 Payload Too Large` status
// curl -v --data-binary "$(head -c 3000 </dev/urandom | base64)" "http://localhost:8000"

// Requests with handlers

// curl -v "http://localhost:8000"
// curl -v "http://localhost:8000/?name=Joe&pets=Turtle&pets=Dog"
// curl -v "http://localhost:8000/echo/ligma"
// curl -v "http://localhost:8000/echo/Hello/reverse/World"
// curl -v "http://localhost:8000/123"
// curl -v "http://localhost:8000/321"
// curl -v -X POST "http://localhost:8000"
// curl -v -X POST "http://localhost:8000/create" -d "name=Joe&last_name=Mama"

func main() {
	s := server.NewServer()

	s.Get("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for key, values := range r.URL.Query() {
			fmt.Printf("Key: %s	| Value(s): %v\n", key, values)
		}

		fmt.Fprintln(w, "Hello world!")
	}))

	s.Get("/echo/{echo}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		echo := r.PathValue("echo")

		fmt.Fprintln(w, echo)
	}))

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

	s.Post("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Unable to read body", http.StatusBadRequest)
			return
		}

		defer r.Body.Close()

		fmt.Fprintf(w, "Hello world from a post! Body is %d bytes long\n", len(body))
	}))

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
