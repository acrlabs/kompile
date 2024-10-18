package service

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	"log"
	"os"
	"text/template"

	"github.com/go-toolsmith/astcopy"
	"github.com/samber/lo"

	"github.com/acrlabs/kompile/pkg/util"

	_ "embed"
)

//go:embed embeds/server.tmpl.go
var serverTemplate string

// Struct to hold the function declaration and its name
type ServerConfig struct {
	FunctionDeclaration string
	FunctionName        string
}

func PrintFullFuncDecl(funcDecl *ast.FuncDecl, args []*ast.Field, fset *token.FileSet) string {
	var buf bytes.Buffer
	newFuncDecl := astcopy.FuncDecl(funcDecl)

	// Look for "channel" arguments; these are interpreted as returns
	newFuncDecl.Type.Params.List = args

	// "return" statements in the body should be discard, and channel sends should be converted to HTTP callbacks
	newBody := stripReturns(funcDecl.Body)
	convertChannelSendToHTTPPost(funcDecl.Name.Name, newBody)

	newFuncDecl.Body = newBody

	err := printer.Fprint(&buf, fset, newFuncDecl)
	if err != nil {
		log.Printf("Failed to print function declaration: %v", err)
		return ""
	}
	return buf.String()
}

// Function to generate the Go source file
func GenerateMain(funcName, functionDecl, outputDir string) error {
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
	outfile := fmt.Sprintf("%s/%s", serverOutputDir, util.MainGoFile)
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

	if err := util.GenerateImports(outfile); err != nil {
		return fmt.Errorf("could not generate imports: %w", err)
	}

	if err := util.InitGoMod(funcName, serverOutputDir); err != nil {
		return fmt.Errorf("could not set up go.mod: %w", err)
	}

	return nil
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

func convertChannelSendToHTTPPost(name string, block *ast.BlockStmt) {
	for i, stmt := range block.List {
		if send, ok := stmt.(*ast.SendStmt); ok {
			chName := send.Chan.(*ast.Ident).Name
			block.List[i] = &ast.ExprStmt{
				X: &ast.CallExpr{
					Fun: &ast.Ident{Name: "http.Post"},
					Args: []ast.Expr{
						&ast.BasicLit{
							Value: fmt.Sprintf("\"%s:8080/%s_%s\"", util.ControllerName, name, chName),
							Kind:  token.STRING,
						},
						&ast.BasicLit{
							Value: "\"application/text\"",
							Kind:  token.STRING,
						},
						sendStmtToIoReader(send),
					},
				},
			}
		}
	}
}

func sendStmtToIoReader(send *ast.SendStmt) ast.Expr {
	return &ast.CallExpr{
		Fun: &ast.Ident{Name: "strings.NewReader"},
		Args: []ast.Expr{
			&ast.CallExpr{
				Fun: &ast.Ident{Name: "fmt.Sprintf"},
				Args: []ast.Expr{
					&ast.BasicLit{
						Value: "\"%v\"",
						Kind:  token.STRING,
					},
					send.Value,
				},
			},
		},
	}
}
