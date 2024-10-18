package kompiler

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"

	"github.com/samber/lo"
	"golang.org/x/tools/go/ast/astutil"

	"github.com/acrlabs/kompile/pkg/controller"
	"github.com/acrlabs/kompile/pkg/service"
	"github.com/acrlabs/kompile/pkg/util"
)

type Kompiler struct {
	node ast.Node
	fset *token.FileSet

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
		node: node,
		fset: fset,

		functions: make(map[string]*ast.FuncDecl),
	}, nil
}

func (self *Kompiler) Compile(outputDir, dockerRegistry string) error {
	fmt.Println("finding potential service calls")
	self.findImportantNodes()
	services, endpoints := self.replaceGoroutines(outputDir, dockerRegistry)
	if err := controller.GenerateMain(self.node, services, endpoints, outputDir, self.fset); err != nil {
		return fmt.Errorf("could not generate client file: %w", err)
	}

	fmt.Println("building executables")
	goBuilder, err := newGoBuilder()
	if err != nil {
		return fmt.Errorf("could not create builder: %w", err)
	}

	toBuild := append(services, util.ControllerDir)
	if err := goBuilder.build(outputDir, dockerRegistry, toBuild); err != nil {
		return fmt.Errorf("could not build executables: %w", err)
	}

	if err := controller.WriteYaml(outputDir); err != nil {
		return fmt.Errorf("could not write controller YAML: %w", err)
	}

	return nil
}

func (self *Kompiler) findImportantNodes() {
	ast.Inspect(self.node, func(n ast.Node) bool {
		if x, ok := n.(*ast.FuncDecl); ok {
			self.functions[x.Name.Name] = x
		}
		return true
	})
}

type nodeScanData struct {
	node             ast.Node
	chanReplacements map[string]string
}

func (self *Kompiler) replaceGoroutines(outputDir, dockerRegistry string) ([]string, []string) {
	services := []string{}
	endpoints := []string{}
	toScan := []nodeScanData{}

	astutil.Apply(self.node, nil, func(c *astutil.Cursor) bool {
		n := c.Node()
		if goStmt, ok := n.(*ast.GoStmt); ok {
			if callFun, ok := goStmt.Call.Fun.(*ast.Ident); ok {
				if function, ok := self.functions[callFun.Name]; ok {
					fmt.Printf("The goroutine is calling the function %s\n", function.Name.Name)

					// These are the argument parameters inside the function declaration...
					args, chanReplacements := selectNonChannelArgs(function, goStmt.Call.Args)

					services = append(services, function.Name.Name)
					endpoints = append(endpoints, lo.Values(chanReplacements)...)
					toScan = append(toScan, nodeScanData{
						node:             c.Parent(),
						chanReplacements: chanReplacements,
					})

					fstring := service.PrintFullFuncDecl(function, args, self.fset)
					if err := service.GenerateMain(function.Name.Name, fstring, outputDir); err != nil {
						log.Fatalf("Error generating server file: %s", err)
					}

					stmt := controller.GenerateServiceCall(function.Name.Name, dockerRegistry, goStmt.Call.Args[0])
					c.Replace(stmt)
				}
			}
		}
		return true
	})

	for _, nsd := range toScan {
		astutil.Apply(nsd.node, nil, func(c *astutil.Cursor) bool {
			n := c.Node()
			if assStmt, ok := n.(*ast.AssignStmt); ok {
				if lhsName, ok := assStmt.Lhs[0].(*ast.Ident); ok {
					if _, ok := nsd.chanReplacements[lhsName.Name]; ok {
						c.Delete()
						return true
					}
				}

				if rhsExpr, ok := assStmt.Rhs[0].(*ast.UnaryExpr); ok {
					if rhsName, ok := rhsExpr.X.(*ast.Ident); ok {
						if replacement, ok := nsd.chanReplacements[rhsName.Name]; ok {
							rhsExpr.X = &ast.Ident{Name: fmt.Sprintf("%s_ch", replacement)}
							return true
						}
					}
					c.Replace(assStmt)
				}
			}
			return true
		})
	}

	return services, endpoints
}

func selectNonChannelArgs(funcDecl *ast.FuncDecl, callArgs []ast.Expr) ([]*ast.Field, map[string]string) {
	chanReplacements := make(map[string]string)
	args := lo.Filter(funcDecl.Type.Params.List, func(arg *ast.Field, i int) bool {
		_, ok := arg.Type.(*ast.ChanType)
		if ok {
			callerChannelName := callArgs[i].(*ast.Ident).Name
			chanReplacements[callerChannelName] = fmt.Sprintf("%s_%s", funcDecl.Name.Name, arg.Names[0].Name)
		}
		return !ok
	})

	return args, chanReplacements
}
