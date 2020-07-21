package exec

import (
	"fmt"
	osexec "os/exec"
	"strings"

	"github.com/jefflinse/handyman/spec"
	"github.com/urfave/cli/v2"
)

// New creates a new command that executes a local binary.
func New(cmdSpec *spec.Command) *cli.Command {
	return &cli.Command{
		Name:   cmdSpec.Name,
		Usage:  cmdSpec.Description,
		Action: newActionFn(cmdSpec.Exec),
	}
}

// Creates a action function.
func newActionFn(cmd string) cli.ActionFunc {
	command := osexec.Command("/bin/bash", "-c", cmd)
	output := strings.Builder{}
	command.Stdout = &output
	command.Stderr = &output
	return func(ctx *cli.Context) error {
		err := command.Run()
		fmt.Print(output.String())
		return err
	}
}
