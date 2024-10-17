package kompiler

import (
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	"os"
	"strings"

	"github.com/samber/lo"
	"golang.org/x/tools/go/ast/astutil"
)

var createPodErr = ast.CallExpr{
	Fun: &ast.Ident{Name: "http.Error"},
	Args: []ast.Expr{
		&ast.Ident{Name: "w"},
		&ast.CallExpr{
			Fun: &ast.Ident{Name: "fmt.Sprintf"},
			Args: []ast.Expr{
				&ast.BasicLit{Value: "\"could not create pod: %v\"", Kind: token.STRING},
				&ast.Ident{Name: "err"},
			},
		},
		&ast.Ident{Name: "http.StatusInternalServerError"},
	},
}

func stripServiceFunctions(rootNode ast.Node, services []string) {
	astutil.Apply(rootNode, nil, func(c *astutil.Cursor) bool {
		n := c.Node()
		if f, ok := n.(*ast.FuncDecl); ok {
			if lo.Contains(services, f.Name.Name) {
				c.Delete()
			}
		}
		return true
	})
}

func generateServiceCall(funcName, dockerRegistry string, callArg ast.Expr) ast.Stmt {
	// TODO replace goroutine call with a function call that:
	// registers an HTTP handler to listen for channel responses'
	lowerName := strings.ToLower(funcName)
	dockerImageStr := fmt.Sprintf("\"%s/%s:latest\"", dockerRegistry, lowerName)
	stmts := []ast.Stmt{
		&ast.AssignStmt{
			Lhs: []ast.Expr{
				&ast.Ident{Name: "podUrl"},
				&ast.Ident{Name: "err"},
			},
			Tok: token.DEFINE,
			Rhs: []ast.Expr{
				&ast.CallExpr{
					Fun: &ast.Ident{Name: "komputil.CreateAndWaitForPod"},
					Args: []ast.Expr{
						&ast.BasicLit{Value: fmt.Sprintf("\"%s\"", lowerName), Kind: token.STRING},
						&ast.BasicLit{Value: dockerImageStr, Kind: token.STRING},
					},
				},
			},
		},
		&ast.IfStmt{
			Cond: &ast.BinaryExpr{
				X:  &ast.Ident{Name: "err"},
				Op: token.NEQ,
				Y:  &ast.Ident{Name: "nil"},
			},
			Body: &ast.BlockStmt{
				List: []ast.Stmt{
					&ast.ExprStmt{X: &createPodErr},
					&ast.ReturnStmt{},
				},
			},
		},
		&ast.ExprStmt{
			X: &ast.CallExpr{
				Fun: &ast.Ident{Name: "http.Post"},
				Args: []ast.Expr{
					&ast.Ident{Name: "podUrl"},
					&ast.BasicLit{Value: "\"application/octet-stream\"", Kind: token.STRING},
					callArg,
				},
			},
		},
	}

	return &ast.BlockStmt{List: stmts}
}

func generateClientFile(rootNode ast.Node, services []string, outputDir string, fset *token.FileSet) error {
	stripServiceFunctions(rootNode, services)

	// Create the output file
	clientOutputDir := fmt.Sprintf("%s/%s", outputDir, clientDir)
	os.RemoveAll(clientOutputDir)
	if err := os.MkdirAll(clientOutputDir, os.ModePerm); err != nil {
		return fmt.Errorf("could not create output directory: %w", err)
	}
	outfile := fmt.Sprintf("%s/%s", clientOutputDir, mainGoFile)
	file, err := os.Create(outfile)
	if err != nil {
		return fmt.Errorf("could not create file: %w", err)
	}
	defer file.Close()

	if err := printer.Fprint(file, fset, rootNode); err != nil {
		return fmt.Errorf("could not write to file: %w", err)
	}

	if err := generateImports(clientOutputDir); err != nil {
		return fmt.Errorf("could not generate imports: %w", err)
	}

	return initGoMod("client", clientOutputDir)
}
