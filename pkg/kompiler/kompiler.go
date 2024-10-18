package kompiler

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"

	_ "embed"

	"github.com/samber/lo"
	"golang.org/x/tools/go/ast/astutil"

	"github.com/acrlabs/kompile/pkg/controller"
	"github.com/acrlabs/kompile/pkg/service"
)

const (
	exeFile = "main"
)

//go:embed embeds/controller.yml
var clientYml string

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
	callbackEndpointMap := self.replaceGoroutines(outputDir, dockerRegistry)
	if err := controller.GenerateMain(self.node, callbackEndpointMap, outputDir, self.fset); err != nil {
		return fmt.Errorf("could not generate client file: %w", err)
	}

	// fmt.Println("building executables")
	// goBuilder, err := newGoBuilder()
	// if err != nil {
	// 	return fmt.Errorf("could not create builder: %w", err)
	// }

	// toBuild := append(services, "client")
	// if err := goBuilder.build(outputDir, dockerRegistry, toBuild); err != nil {
	// 	return fmt.Errorf("could not build executables: %w", err)
	// }

	// f, err := os.Create(fmt.Sprintf("%s/client/deployment.yml", outputDir))
	// if err != nil {
	// 	return fmt.Errorf("could not create client k8s manifest: %w", err)
	// }
	// defer f.Close()
	// fmt.Fprintf(f, clientYml)

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

func (self *Kompiler) replaceGoroutines(outputDir, dockerRegistry string) map[string]map[int]string {
	callbackEndpointMap := map[string]map[int]string{}

	astutil.Apply(self.node, nil, func(c *astutil.Cursor) bool {
		n := c.Node()
		if goStmt, ok := n.(*ast.GoStmt); ok {
			if callExpr, ok := goStmt.Call.Fun.(*ast.Ident); ok {
				if function, ok := self.functions[callExpr.Name]; ok {
					fmt.Printf("The goroutine is calling the function %s\n", function.Name.Name)

					// These are the argument parameters inside the function declaration...
					args, channelArgPositions := selectNonChannelArgs(function)
					callbacks := computeCallbackEndpointMap(function, channelArgPositions)

					callbackEndpointMap[function.Name.Name] = callbacks

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

	return callbackEndpointMap
}

func selectNonChannelArgs(funcDecl *ast.FuncDecl) ([]*ast.Field, []int) {
	channelArgPositions := []int{}
	args := lo.Filter(funcDecl.Type.Params.List, func(arg *ast.Field, i int) bool {
		_, ok := arg.Type.(*ast.ChanType)
		if ok {
			channelArgPositions = append(channelArgPositions, i)
		}
		return !ok
	})

	return args, channelArgPositions
}

func computeCallbackEndpointMap(function *ast.FuncDecl, channelArgPositions []int) map[int]string {
	entries := lo.Map(channelArgPositions, func(pos, _ int) lo.Entry[int, string] {
		return lo.Entry[int, string]{
			Key:   pos,
			Value: fmt.Sprintf("%s_%s", function.Name.Name, function.Type.Params.List[pos].Names[0]),
		}
	})
	return lo.FromEntries(entries)
}
