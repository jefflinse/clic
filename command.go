package handyman

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/jefflinse/handyman/spec"
	"github.com/urfave/cli/v2"
)

// Creates a CLI command from the provided command spec.
func newCommandFromSpec(commandSpec *spec.Command) *cli.Command {
	command := &cli.Command{
		Name:  commandSpec.Name,
		Usage: commandSpec.Description,
	}

	switch commandSpec.Type {
	case spec.ExecCommandType:
		command.Action = newExecInvocationFn(commandSpec.Exec)
	case spec.LambdaCommandType:
		command.Action = newLambdaInvocationFn(commandSpec.LambdaARN)
	case spec.NoopCommandType:
		command.Action = newNoopInvocationFn()
	case spec.SubcommandsCommandType:
		for _, subcommandSpec := range commandSpec.Subcommands {
			command.Subcommands = append(command.Subcommands, newCommandFromSpec(subcommandSpec))
		}
	}

	return command
}

// Creates an invocation function that executes a local command via the shell.
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

// Creates an invocation function that executes an AWS Lambda function and prints its results.
func newLambdaInvocationFn(lambdaARN string) func(ctx *cli.Context) error {
	request := map[string]interface{}{}
	return func(ctx *cli.Context) error {
		response, functionError, err := executeLambda(lambdaARN, request)
		if err != nil {
			return err
		} else if functionError != nil {
			fmt.Fprint(os.Stderr, *functionError)
		}

		fmt.Print(string(response))
		return nil
	}
}

// Creates an invocation function that does nothing (no-op).
func newNoopInvocationFn() func(ctx *cli.Context) error {
	return func(ctx *cli.Context) error {
		return nil
	}
}
