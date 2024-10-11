package kompiler

import (
	"fmt"
	"os"
	"os/exec"
	"text/template"
)

// Template for the HTTP server
const serverTemplate = `
package main

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Handler function to be invoked
{{ .FunctionDeclaration }}

// Main function to set up the HTTP server
func main() {
	r := chi.NewRouter()

	// Define the POST endpoint
	r.Post("/", {{ .FunctionName }})

	// Start the HTTP server
	fmt.Println("Starting server on :8080")
	if err := http.ListenAndServe(":8080", r); err != nil {
		fmt.Printf("Error starting server: %v\n", err)
	}
}
`

// TODO wrap the extracted goroutine in a function that
// takes a http.ResponseWriter and *http.Request as arguments
// OR perhaps we only support extracting goroutines that already have this signature for now
const handlerTemplate = `
func {{ .FunctionName }}Handler(w http.ResponseWriter, r *http.Request) {
	{{ .FunctionName }}(w, r)
}
`

// Struct to hold the function declaration and its name
type ServerConfig struct {
	FunctionDeclaration string
	FunctionName        string
}

func generateImports(filePath string) error {
	err := exec.Command("goimports", "-w", filePath).Run()
	if err != nil {
		return fmt.Errorf("could not run goimports: %w", err)
	}
	return nil
}

// Function to generate the Go source file
func generateServerFile(funcName, functionDecl, outputDir string) error {
	// Create the server configuration
	config := ServerConfig{
		FunctionDeclaration: functionDecl,
		FunctionName:        funcName,
	}

	// Create the output file
	outfile := fmt.Sprintf("%s/%s_server.go", outputDir, funcName)
	file, err := os.Create(outfile)
	if err != nil {
		return fmt.Errorf("could not create file: %w", err)
	}
	defer file.Close()

	// Parse and execute the template
	tmpl, err := template.New("server").Parse(serverTemplate)
	if err != nil {
		return fmt.Errorf("could not parse template: %w", err)
	}

	if err := tmpl.Execute(file, config); err != nil {
		return fmt.Errorf("could not execute template: %w", err)
	}

	if err := generateImports(outfile); err != nil {
		return fmt.Errorf("could not generate imports: %w", err)
	}

	return nil
}
