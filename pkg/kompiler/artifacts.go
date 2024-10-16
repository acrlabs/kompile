package kompiler

import (
	"fmt"
	"os"
	"os/exec"
)

const dockerfileTemplate = `
FROM alpine:latest

COPY main /main

CMD ["/main"]
`

type goBuilder struct {
	goEnv []string
}

func newGoBuilder() (*goBuilder, error) {
	home := os.Getenv("HOME")
	goEnv := []string{
		"CGO_ENABLED=0",
		fmt.Sprintf("GOPATH=%s/go", home),
		fmt.Sprintf("HOME=%s", home),
	}
	return &goBuilder{
		goEnv: goEnv,
	}, nil
}

func (self *goBuilder) build(outputDir string, names []string) error {
	for _, name := range names {
		buildCmd := exec.Command("go", "build", "-ldflags", "-s -w", "-trimpath", "-o", exeFile, mainGoFile)
		buildCmd.Dir = fmt.Sprintf("%s/%s", outputDir, name)
		buildCmd.Env = self.goEnv
		buildCmd.Stderr = os.Stderr
		fmt.Printf("  Running %v\n", buildCmd)

		if err := buildCmd.Run(); err != nil {
			return fmt.Errorf("could not run go build: %w", err)
		}
	}

	return nil
}
