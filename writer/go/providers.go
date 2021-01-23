package gowriter

import (
	"os"
	"os/exec"
)

func doExec(path string, args ...string) {
	command := exec.Command(path, args...)
	command.Env = os.Environ()
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr

	if err := command.Run(); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			os.Exit(exitError.ExitCode())
		}

		os.Exit(1)
	}

	os.Exit(0)
}
