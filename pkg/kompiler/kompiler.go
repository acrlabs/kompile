package kompiler

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"

	_ "embed"

	"golang.org/x/tools/go/ast/astutil"
)

const (
	clientDir  = "client"
	mainGoFile = "main.go"
	exeFile    = "main"
)

//go:embed embeds/client.yml
var clientYml string

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

func (self *Kompiler) Compile(outputDir, dockerRegistry string) error {
	fmt.Println("finding potential service calls")
	self.findFunctions()
	services := self.findGoroutines(outputDir, dockerRegistry)
	if err := generateClientFile(self.node, services, outputDir, self.fset); err != nil {
		return fmt.Errorf("could not generate client file: %w", err)
	}

	fmt.Println("building executables")
	goBuilder, err := newGoBuilder()
	if err != nil {
		return fmt.Errorf("could not create builder: %w", err)
	}

	toBuild := append(services, "client")
	if err := goBuilder.build(outputDir, dockerRegistry, toBuild); err != nil {
		return fmt.Errorf("could not build executables: %w", err)
	}

	f, err := os.Create(fmt.Sprintf("%s/client/deployment.yml", outputDir))
	if err != nil {
		return fmt.Errorf("could not create client k8s manifest: %w", err)
	}
	defer f.Close()
	fmt.Fprintf(f, clientYml)

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

func (self *Kompiler) findGoroutines(outputDir, dockerRegistry string) []string {
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

					stmt := generateServiceCall(function.Name.Name, dockerRegistry, goStmt.Call.Args[0])
					c.Replace(stmt)
				}
			}
		}
		return true
	})

	return services
}
