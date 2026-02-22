package util

import (
	"os/exec"
	"strings"

	"github.com/pkg/errors"
)

func Run(workDir string, args ...string) (string, error) {
	executablePath, err := exec.LookPath(args[0])

	if err != nil {
		return "", err
	}

	command := exec.Command(executablePath, args[1:]...)
	command.Dir = workDir
	// command.Stderr = os.Stderr
	// command.Stdout = os.Stdout

	output, err := command.Output()
	if exitErr, ok := err.(*exec.ExitError); ok {
		return "", errors.Wrapf(exitErr, "failed to execute command %s: %s", executablePath, exitErr.Stderr)
	}

	return strings.TrimSuffix(string(output), "\n"), err
}
