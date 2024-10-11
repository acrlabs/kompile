package kompiler

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	"log"
)

//nolint:gochecknoglobals // this is demo code
var functions = make(map[string]*ast.FuncDecl)

func FindFunctions(node ast.Node) {
	ast.Inspect(node, func(n ast.Node) bool {
		//nolint:gocritic // can't use .(type) outside of switch
		switch x := n.(type) {
		case *ast.FuncDecl:
			functions[x.Name.Name] = x
		}
		return true
	})
}

func FindGoroutines(node ast.Node, fset *token.FileSet, outputDir string) {
	ast.Inspect(node, func(n ast.Node) bool {
		if goStmt, ok := n.(*ast.GoStmt); ok {
			if callExpr, ok := goStmt.Call.Fun.(*ast.Ident); ok {
				if function, ok := functions[callExpr.Name]; ok {
					fmt.Printf("The goroutine is calling the function %s\n", function.Name.Name)
					fstring := printFullFuncDecl(function, fset)
					if err := generateServerFile(function.Name.Name, fstring, outputDir); err != nil {
						log.Fatalf("Error generating server file: %s", err)
					}
					replaceWithServiceCall(goStmt)
				}
			}
		}
		return true
	})
}

func printFullFuncDecl(funcDecl *ast.FuncDecl, fset *token.FileSet) string {
	var buf bytes.Buffer
	// Use the printer package to format the function declaration
	err := printer.Fprint(&buf, fset, funcDecl)
	if err != nil {
		log.Printf("Failed to print function declaration: %v", err)
		return ""
	}
	return buf.String()
}

func replaceWithServiceCall(node *ast.GoStmt) {
	// TODO replace goroutine call with a function call that:
	// registers an HTTP handler to listen for channel responses'
	// creates a pod, runs it
	// gets pod IP address
	// makes a POST request to the pod
}
