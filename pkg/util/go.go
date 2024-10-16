package util

import (
	"fmt"
	"os/exec"
)

func GenerateImports(filePath string) error {
	err := exec.Command("goimports", "-w", filePath).Run()
	if err != nil {
		return fmt.Errorf("could not run goimports: %w", err)
	}
	return nil
}

func InitGoMod(name, outputDir string) error {
	initCmd := exec.Command("go", "mod", "init", name)
	initCmd.Dir = outputDir
	if err := initCmd.Run(); err != nil {
		return fmt.Errorf("could not run go mod init: %w", err)
	}
	tidyCmd := exec.Command("go", "mod", "tidy")
	tidyCmd.Dir = outputDir
	if err := tidyCmd.Run(); err != nil {
		return fmt.Errorf("could not run go mod tidy: %w", err)
	}
	replaceCmd := exec.Command("go", "mod", "edit", "-replace=github.com/acrlabs/kompile=../../")
	replaceCmd.Dir = outputDir
	if err := replaceCmd.Run(); err != nil {
		return fmt.Errorf("could not run go mod edit: %w", err)
	}
	return nil
}
