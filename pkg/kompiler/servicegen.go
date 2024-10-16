package kompiler

import (
	"fmt"
	"os"
	"os/exec"
	"text/template"

	_ "embed"
)

//go:embed embeds/server.tmpl.go
var serverTemplate string

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
	outfile := fmt.Sprintf("%s/%s", serverOutputDir, mainGoFile)
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
