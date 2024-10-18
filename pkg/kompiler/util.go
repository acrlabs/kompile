package kompiler

import (
	"go/ast"
)

func isChannelAssignment(a *ast.AssignStmt) bool {
	if len(a.Rhs) != 1 {
		return false
	}

	callExpr, ok := a.Rhs[0].(*ast.CallExpr)
	if !ok {
		return false
	}

	fn, ok := callExpr.Fun.(*ast.Ident)
	if !ok {
		return false
	}

	if fn.Name != "make" {
		return false
	}

	if len(callExpr.Args) != 1 {
		return false
	}

	_, ok = callExpr.Args[0].(*ast.ChanType)
	return ok
}
