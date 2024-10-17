package main

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/acrlabs/kompile/pkg/kompiler"
)

const progname = "kompile"

type options struct {
	filename       string
	outputDir      string
	dockerRegistry string
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
	root.PersistentFlags().StringVarP(
		&opts.dockerRegistry,
		"docker-registry",
		"r",
		"localhost:5000",
		"location of docker registry to push to",
	)
	if err := root.MarkPersistentFlagRequired("filename"); err != nil {
		panic(err)
	}

	return root
}

func start(opts *options) {
	k, err := kompiler.New(opts.filename)
	if err != nil {
		panic(err)
	}
	if err := k.Compile(opts.outputDir, opts.dockerRegistry); err != nil {
		panic(err)
	}
}

func main() {
	if err := rootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}
