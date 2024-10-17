package kompiler

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	"log"
	"os"
	"text/template"

	_ "embed"

	"github.com/go-toolsmith/astcopy"
	"github.com/samber/lo"
)

//go:embed embeds/server.tmpl.go
var serverTemplate string

// Struct to hold the function declaration and its name
type ServerConfig struct {
	FunctionDeclaration string
	FunctionName        string
}

func printFullFuncDecl(funcDecl *ast.FuncDecl, args, returns []*ast.Field, fset *token.FileSet) string {
	var buf bytes.Buffer
	newFuncDecl := astcopy.FuncDecl(funcDecl)

	// Look for "channel" arguments; these are interpreted as returns
	newFuncDecl.Type.Params.List = args
	newFuncDecl.Type.Results = &ast.FieldList{List: returns}

	// "return" statements in the body should be discard, and channel sends should be converted to return statements
	newBody := stripReturns(funcDecl.Body)
	newBody = convertChannelSendToReturn(newBody)

	newFuncDecl.Body = newBody

	err := printer.Fprint(&buf, fset, newFuncDecl)
	if err != nil {
		log.Printf("Failed to print function declaration: %v", err)
		return ""
	}
	return buf.String()
}

func convertChannelArgsToReturns(funcDecl *ast.FuncDecl) ([]*ast.Field, []*ast.Field) {
	args, channels := lo.FilterReject(funcDecl.Type.Params.List, func(f *ast.Field, _ int) bool {
		_, ok := f.Type.(*ast.ChanType)
		return !ok
	})
	returns := lo.Map(channels, func(f *ast.Field, _ int) *ast.Field {
		ct, ok := f.Type.(*ast.ChanType)
		if !ok {
			panic("already checked, should not fail")
		}
		return &ast.Field{Type: ct.Value}
	})
	return args, returns
}

func stripReturns(block *ast.BlockStmt) *ast.BlockStmt {
	stmts := lo.FilterMap(block.List, func(stmt ast.Stmt, _ int) (ast.Stmt, bool) {
		switch s := stmt.(type) {
		case *ast.ReturnStmt:
			return nil, false
		case *ast.IfStmt:
			newIf := astcopy.IfStmt(s)
			newIf.Body = stripReturns(s.Body)
			return newIf, true
		default:
			return stmt, true
		}
	})

	return &ast.BlockStmt{List: stmts}
}

func convertChannelSendToReturn(block *ast.BlockStmt) *ast.BlockStmt {
	stmts, sends := lo.FilterReject(block.List, func(stmt ast.Stmt, _ int) bool {
		_, ok := stmt.(*ast.SendStmt)
		return !ok
	})

	if len(sends) > 1 {
		panic("don't know how to handle too many sends")
	}

	if len(sends) == 1 {
		send, ok := sends[0].(*ast.SendStmt)
		if !ok {
			panic("already checked, should not fail")
		}
		stmts = append(stmts, &ast.ReturnStmt{Results: []ast.Expr{send.Value}})
	}

	return &ast.BlockStmt{List: stmts}
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
