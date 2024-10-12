package kompiler

import (
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	"os"
)

func generateClientFile(rootNode ast.Node, fset *token.FileSet, outputDir string) error {
	// Create the output file
	clientOutputDir := fmt.Sprintf("%s/client", outputDir)
	os.RemoveAll(clientOutputDir)
	if err := os.MkdirAll(clientOutputDir, os.ModePerm); err != nil {
		return fmt.Errorf("could not create output directory: %w", err)
	}
	outfile := fmt.Sprintf("%s/main.go", clientOutputDir)
	file, err := os.Create(outfile)
	if err != nil {
		return fmt.Errorf("could not create file: %w", err)
	}
	defer file.Close()

	if err := printer.Fprint(file, fset, rootNode); err != nil {
		return fmt.Errorf("could not write to file: %w", err)
	}

	return initGoMod("client", clientOutputDir)
}
