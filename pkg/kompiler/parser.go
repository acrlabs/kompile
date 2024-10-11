package kompiler

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	"log"

	"github.com/go-toolsmith/astcopy"
	"github.com/samber/lo"
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
	newFuncDecl := astcopy.FuncDecl(funcDecl)

	// Look for "channel" arguments; these are interpreted as returns
	args, returns := convertChannelArgsToReturns(funcDecl)
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

func replaceWithServiceCall(node *ast.GoStmt) {
	// TODO replace goroutine call with a function call that:
	// registers an HTTP handler to listen for channel responses'
	// creates a pod, runs it
	// gets pod IP address
	// makes a POST request to the pod
}
