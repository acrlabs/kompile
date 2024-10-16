package main

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Handler function to be invoked
{{ .FunctionDeclaration }}

// Wrapped handler function
func {{ .FunctionName }}Handler(w http.ResponseWriter, r *http.Request) {
	ret := {{ .FunctionName}}(r.Body)
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte(ret)); err != nil {
		fmt.Printf("Error writing to client: %v\n", err)
	}
}

// Main function to set up the HTTP server
func main() {
	r := chi.NewRouter()

	// Define the POST endpoint
	r.Post("/", {{ .FunctionName }}Handler)

	// Start the HTTP server
	fmt.Println("Starting server on :8080")
	if err := http.ListenAndServe(":8080", r); err != nil {
		fmt.Printf("Error starting server: %v\n", err)
	}
}
