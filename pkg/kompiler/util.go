package kompiler

import (
	"fmt"
	"os/exec"
)

func initGoMod(name, outputDir string) error {
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
	return nil
}
