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

	serverOutputDir := fmt.Sprintf("%s/%s", outputDir, funcName)
	os.RemoveAll(serverOutputDir)
	if err := os.MkdirAll(serverOutputDir, os.ModePerm); err != nil {
		return fmt.Errorf("could not create directory: %w", err)
	}

	// Create the output file
	outfile := fmt.Sprintf("%s/main.go", serverOutputDir)
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

	return initGoMod(funcName, serverOutputDir)
}
