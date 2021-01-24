package gowriter

import (
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

func applyParameters(target string, cmd *cobra.Command, argValues []string) string {
	parameterized := target

	cmd.Flags().GetBool("fff")

	return parameterized
}

func doexec(path string, args []string) {
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
