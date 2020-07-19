package handyman

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/jefflinse/handyman/spec"
	"github.com/urfave/cli/v2"
)

func newCommandFromSpec(commandSpec *spec.Command) *cli.Command {
	command := &cli.Command{
		Name:  commandSpec.Name,
		Usage: commandSpec.Description,
	}

	switch commandSpec.Type {
	case spec.NoopCommandType:
		command.Action = newNoopInvocationFn()
	case spec.ExecCommandType:
		command.Action = newExecInvocationFn(commandSpec.Exec)
	}

	if commandSpec.Subcommands != nil {
		for _, subcommandSpec := range commandSpec.Subcommands {
			command.Subcommands = append(command.Subcommands, newCommandFromSpec(subcommandSpec))
		}
	}

	return command
}

func newNoopInvocationFn() func(ctx *cli.Context) error {
	return func(ctx *cli.Context) error {
		return nil
	}
}

func newExecInvocationFn(cmd string) func(ctx *cli.Context) error {
	command := exec.Command("/bin/bash", "-c", cmd)
	output := strings.Builder{}
	command.Stdout = &output
	command.Stderr = &output
	return func(ctx *cli.Context) error {
		err := command.Run()
		fmt.Print(output.String())
		return err
	}
}
