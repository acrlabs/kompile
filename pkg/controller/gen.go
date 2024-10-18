package controller

import (
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	"html/template"
	"os"
	"strings"

	_ "embed"

	"github.com/samber/lo"
	"golang.org/x/tools/go/ast/astutil"

	"github.com/acrlabs/kompile/pkg/util"
)

//go:embed embeds/controller.yml
var controllerYamlTemplate string

type ControllerConfig struct {
	ControllerName  string
	ControllerImage string
}

func httpError(msg string) *ast.CallExpr {
	return &ast.CallExpr{
		Fun: &ast.Ident{Name: "http.Error"},
		Args: []ast.Expr{
			&ast.Ident{Name: "w"},
			&ast.CallExpr{
				Fun: &ast.Ident{Name: "fmt.Sprintf"},
				Args: []ast.Expr{
					&ast.BasicLit{Value: fmt.Sprintf("\"%s: %%v\"", msg), Kind: token.STRING},
					&ast.Ident{Name: "err"},
				},
			},
			&ast.Ident{Name: "http.StatusInternalServerError"},
		},
	}
}

func GenerateServiceCall(funcName, dockerRegistry string, callArg ast.Expr) ast.Stmt {
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
					&ast.ExprStmt{X: httpError("could not create pod")},
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

func GenerateMain(rootNode ast.Node, services, endpoints []string, outputDir string, fset *token.FileSet) error {
	stripServiceFunctions(rootNode, services)
	addCallbackEndpoints(rootNode, endpoints)
	addChannelGlobals(rootNode, endpoints)
	addHandlerFuncs(rootNode, endpoints)

	// Create the output file
	controllerOutputDir := fmt.Sprintf("%s/%s", outputDir, util.ControllerDir)
	os.RemoveAll(controllerOutputDir)
	if err := os.MkdirAll(controllerOutputDir, os.ModePerm); err != nil {
		return fmt.Errorf("could not create output directory: %w", err)
	}
	outfile := fmt.Sprintf("%s/%s", controllerOutputDir, util.MainGoFile)
	file, err := os.Create(outfile)
	if err != nil {
		return fmt.Errorf("could not create file: %w", err)
	}
	defer file.Close()

	if err := printer.Fprint(file, fset, rootNode); err != nil {
		return fmt.Errorf("could not write to file: %w", err)
	}

	if err := util.GenerateImports(controllerOutputDir); err != nil {
		return fmt.Errorf("could not generate imports: %w", err)
	}

	return util.InitGoMod("client", controllerOutputDir)
}

func WriteYaml(outputDir string) error {
	config := ControllerConfig{
		ControllerName:  util.ControllerName,
		ControllerImage: fmt.Sprintf("localhost:5000/%s:latest", util.ControllerDir),
	}
	f, err := os.Create(fmt.Sprintf("%s/%s/deployment.yml", outputDir, util.ControllerDir))
	if err != nil {
		return fmt.Errorf("could not create client k8s manifest: %w", err)
	}
	defer f.Close()

	tmpl, err := template.New("controllerYml").Parse(controllerYamlTemplate)
	if err != nil {
		return fmt.Errorf("could not parse template: %w", err)
	}

	if err := tmpl.Execute(f, config); err != nil {
		return fmt.Errorf("could not execute template: %w", err)
	}
	return nil
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

func addCallbackEndpoints(rootNode ast.Node, endpoints []string) {
	astutil.Apply(rootNode, nil, func(c *astutil.Cursor) bool {
		n := c.Node()
		if f, ok := n.(*ast.FuncDecl); ok {
			if f.Name.Name == "main" {
				stmts := lo.Map(endpoints, func(endpoint string, _ int) ast.Stmt {
					return &ast.ExprStmt{
						X: &ast.CallExpr{
							Fun: &ast.Ident{Name: "http.HandleFunc"},
							Args: []ast.Expr{
								&ast.BasicLit{
									Value: fmt.Sprintf("\"/%s\"", endpoint),
									Kind:  token.STRING,
								},
								&ast.Ident{Name: endpoint},
							},
						},
					}
				})
				f.Body.List = append(stmts, f.Body.List...)
			}
		}
		return true
	})
}

func addChannelGlobals(rootNode ast.Node, endpoints []string) {
	file, ok := rootNode.(*ast.File)
	if !ok {
		panic("root node is not a file")
	}
	channelDecls := lo.Map(endpoints, func(endpoint string, _ int) ast.Decl {
		return &ast.GenDecl{
			Tok: token.VAR,
			Specs: []ast.Spec{
				&ast.ValueSpec{
					Names: []*ast.Ident{{Name: fmt.Sprintf("%s_ch", endpoint)}},
					Values: []ast.Expr{
						&ast.CallExpr{
							Fun: &ast.Ident{Name: "make"},
							Args: []ast.Expr{
								&ast.ChanType{
									Dir:   3,
									Value: &ast.Ident{Name: "string"},
								},
							},
						},
					},
				},
			},
		}
	})
	file.Decls = append(file.Decls, channelDecls...)
}

func addHandlerFuncs(rootNode ast.Node, endpoints []string) {
	file, ok := rootNode.(*ast.File)
	if !ok {
		panic("root node is not a file")
	}
	handlerFuncDecls := lo.Map(endpoints, func(endpoint string, _ int) ast.Decl {
		return &ast.FuncDecl{
			Name: &ast.Ident{Name: endpoint},
			Type: &ast.FuncType{
				Params: &ast.FieldList{
					List: []*ast.Field{
						{
							Names: []*ast.Ident{{Name: "w"}},
							Type:  &ast.Ident{Name: "http.ResponseWriter"},
						},
						{
							Names: []*ast.Ident{{Name: "r"}},
							Type:  &ast.Ident{Name: "*http.Request"},
						},
					},
				},
			},
			Body: &ast.BlockStmt{
				List: []ast.Stmt{
					&ast.AssignStmt{
						Lhs: []ast.Expr{
							&ast.Ident{Name: "b"},
							&ast.Ident{Name: "err"},
						},
						Tok: token.DEFINE,
						Rhs: []ast.Expr{
							&ast.CallExpr{
								Fun: &ast.Ident{Name: "io.ReadAll"},
								Args: []ast.Expr{
									&ast.Ident{Name: "r.Body"},
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
								&ast.ExprStmt{X: httpError("could not read response")},
								&ast.ReturnStmt{},
							},
						},
					},
					&ast.SendStmt{
						Chan: &ast.Ident{Name: fmt.Sprintf("%s_ch", endpoint)},
						Value: &ast.CallExpr{
							Fun: &ast.Ident{Name: "string"},
							Args: []ast.Expr{
								&ast.Ident{Name: "b"},
							},
						},
					},
					&ast.ExprStmt{
						X: &ast.CallExpr{
							Fun: &ast.Ident{Name: "r.Body.Close"},
						},
					},
					&ast.ExprStmt{
						X: &ast.CallExpr{
							Fun: &ast.Ident{Name: "w.WriteHeader"},
							Args: []ast.Expr{
								&ast.Ident{Name: "http.StatusOK"},
							},
						},
					},
				},
			},
		}
	})
	file.Decls = append(file.Decls, handlerFuncDecls...)
}
