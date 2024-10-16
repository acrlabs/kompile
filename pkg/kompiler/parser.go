package kompiler

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"log"

	"github.com/go-toolsmith/astcopy"
	"github.com/samber/lo"
	"golang.org/x/tools/go/ast/astutil"
)

const (
	clientDir  = "client"
	mainGoFile = "main.go"
	exeFile    = "main"
)

type Kompiler struct {
	node      ast.Node
	fset      *token.FileSet
	functions map[string]*ast.FuncDecl
}

func New(filename string) (*Kompiler, error) {
	fset := token.NewFileSet()

	// parse the source file into an AST
	node, err := parser.ParseFile(fset, filename, nil, parser.AllErrors)
	if err != nil {
		return nil, fmt.Errorf("error parsing file: %w", err)
	}
	return &Kompiler{
		node:      node,
		fset:      fset,
		functions: map[string]*ast.FuncDecl{},
	}, nil
}

func (self *Kompiler) Compile(outputDir string) error {
	fmt.Println("finding potential service calls")
	self.findFunctions()
	services := self.findGoroutines(outputDir)
	if err := generateClientFile(self.node, self.fset, outputDir); err != nil {
		return fmt.Errorf("could not generate client file: %w", err)
	}

	fmt.Println("building executables")
	goBuilder, err := newGoBuilder()
	if err != nil {
		return fmt.Errorf("could not create builder: %w", err)
	}

	toBuild := append(services, "client")
	if err := goBuilder.build(outputDir, toBuild); err != nil {
		return fmt.Errorf("could not build executables: %w", err)
	}

	return nil
}

func (self *Kompiler) findFunctions() {
	ast.Inspect(self.node, func(n ast.Node) bool {
		//nolint:gocritic // can't use .(type) outside of switch
		switch x := n.(type) {
		case *ast.FuncDecl:
			self.functions[x.Name.Name] = x
		}
		return true
	})
}

func (self *Kompiler) findGoroutines(outputDir string) []string {
	services := []string{}
	astutil.Apply(self.node, nil, func(c *astutil.Cursor) bool {
		n := c.Node()
		if goStmt, ok := n.(*ast.GoStmt); ok {
			if callExpr, ok := goStmt.Call.Fun.(*ast.Ident); ok {
				if function, ok := self.functions[callExpr.Name]; ok {
					fmt.Printf("The goroutine is calling the function %s\n", function.Name.Name)
					services = append(services, function.Name.Name)

					// These are the argument parameters inside the function declaration...
					args, returns := convertChannelArgsToReturns(function)
					fstring := printFullFuncDecl(function, args, returns, self.fset)
					if err := generateServerFile(function.Name.Name, fstring, outputDir); err != nil {
						log.Fatalf("Error generating server file: %s", err)
					}

					c.Replace(generateServiceCall(goStmt.Call.Args[0]))
				}
			}
		}
		return true
	})

	return services
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

func generateServiceCall(callArg ast.Expr) ast.Stmt {
	// TODO replace goroutine call with a function call that:
	// registers an HTTP handler to listen for channel responses'
	// creates a pod, runs it
	// gets pod IP address
	// makes a POST request to the pod
	callArgs := []ast.Expr{
		&ast.BasicLit{Value: "\"localhost:8080\"", Kind: token.STRING},
		&ast.BasicLit{Value: "\"application/octet-stream\"", Kind: token.STRING},
		callArg,
	}

	return &ast.ExprStmt{
		X: &ast.CallExpr{
			Fun:  &ast.Ident{Name: "http.Post"},
			Args: callArgs,
		},
	}
}
