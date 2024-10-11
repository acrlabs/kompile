package main

import (
	"fmt"
	"go/parser"
	"go/token"
	"log"
	"os"

	"github.com/spf13/cobra"

	"github.com/acrlabs/kompile/pkg/kompiler"
)

const progname = "kompile"

type options struct {
	filename  string
	outputDir string
}

func rootCmd() *cobra.Command {
	opts := options{
		filename: "foobar",
	}

	root := &cobra.Command{
		Use:   progname,
		Short: "Demo compiler for Kubernetes",
		Run: func(_ *cobra.Command, _ []string) {
			start(&opts)
		},
	}

	root.PersistentFlags().StringVarP(&opts.filename, "filename", "f", "", "go program to parse")
	root.PersistentFlags().StringVarP(&opts.outputDir, "output", "o", "output", "directory to create generated files")
	if err := root.MarkPersistentFlagRequired("filename"); err != nil {
		panic(err)
	}

	return root
}

func start(opts *options) {
	fset := token.NewFileSet()

	// parse the source file into an AST
	node, err := parser.ParseFile(fset, opts.filename, nil, parser.AllErrors)
	if err != nil {
		log.Fatalf("Error parsing file: %s", err)
		fmt.Println(err)
		return
	}

	kompiler.FindFunctions(node)
	kompiler.FindGoroutines(node, fset, opts.outputDir)
}

func main() {
	if err := rootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}
