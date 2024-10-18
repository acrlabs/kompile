package util

import (
	"fmt"
	"go/ast"
	"go/token"

	"github.com/samber/lo"
)

func FmtPrintExpr(fn, msg string, args ...string) *ast.CallExpr {
	argExprs := []ast.Expr{
		&ast.BasicLit{Value: "\"" + msg + "\"", Kind: token.STRING},
	}
	params := lo.Map(args, func(a string, _ int) ast.Expr { return &ast.Ident{Name: a} })
	argExprs = append(argExprs, params...)

	return &ast.CallExpr{
		Fun:  &ast.Ident{Name: fmt.Sprintf("fmt.%s", fn)},
		Args: argExprs,
	}
}

func HttpErrorExpr(msg string) *ast.CallExpr {
	return &ast.CallExpr{
		Fun: &ast.Ident{Name: "http.Error"},
		Args: []ast.Expr{
			&ast.Ident{Name: "w"},
			FmtPrintExpr("Sprintf", msg, "err"),
			&ast.Ident{Name: "http.StatusInternalServerError"},
		},
	}
}
