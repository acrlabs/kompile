package kompiler

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	_ "embed"
)

//go:embed embeds/Dockerfile
var dockerfile string

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

func (self *goBuilder) build(outputDir, dockerRegistry string, names []string) error {
	for _, name := range names {
		workingDir := fmt.Sprintf("%s/%s", outputDir, name)
		buildCmd := exec.Command("go", "build", "-ldflags", "-s -w", "-trimpath", "-o", exeFile, mainGoFile)
		buildCmd.Dir = workingDir
		buildCmd.Env = self.goEnv
		buildCmd.Stderr = os.Stderr
		fmt.Printf("  Running %v\n", buildCmd)

		if err := buildCmd.Run(); err != nil {
			return fmt.Errorf("could not run go build for %s: %w", name, err)
		}

		f, err := os.Create(fmt.Sprintf("%s/Dockerfile", workingDir))
		if err != nil {
			return fmt.Errorf("could not create Dockerfile for %s: %w", name, err)
		}
		defer f.Close()

		fmt.Fprintf(f, dockerfile)

		dockerPath := strings.ToLower(fmt.Sprintf("%s/%s:latest", dockerRegistry, name))
		dockerBuildCmd := exec.Command("docker", "build", ".", "-t", dockerPath)
		dockerBuildCmd.Dir = workingDir
		dockerBuildCmd.Stderr = os.Stderr
		fmt.Printf("  Running %v\n", dockerBuildCmd)

		if err := dockerBuildCmd.Run(); err != nil {
			return fmt.Errorf("could not run docker build for %s: %w", name, err)
		}

		dockerPushCmd := exec.Command("docker", "push", dockerPath)
		dockerPushCmd.Dir = workingDir
		dockerPushCmd.Stderr = os.Stderr
		fmt.Printf("  Running %v\n", dockerPushCmd)

		if err := dockerPushCmd.Run(); err != nil {
			return fmt.Errorf("could not run docker push for %s: %w", name, err)
		}

	}

	return nil
}
