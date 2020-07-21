package handyman

import (
	"github.com/jefflinse/handyman/commands/exec"
	"github.com/jefflinse/handyman/commands/lambda"
	"github.com/jefflinse/handyman/commands/noop"
	"github.com/jefflinse/handyman/spec"
	"github.com/urfave/cli/v2"
)

// Creates a CLI command from the provided command spec.
func newCommandFromSpec(cmdSpec *spec.Command) *cli.Command {
	switch cmdSpec.Type {
	case spec.ExecCommandType:
		return exec.New(cmdSpec)
	case spec.LambdaCommandType:
		return lambda.New(cmdSpec)
	case spec.NoopCommandType:
		return noop.New(cmdSpec)
	case spec.SubcommandsCommandType:
		// subcommands is a special case, at least for now,
		// to avoid a command-specific dependency on newCommandFromSpec().
		command := &cli.Command{
			Name:  cmdSpec.Name,
			Usage: cmdSpec.Description,
		}

		for _, subCmdSpec := range cmdSpec.Subcommands {
			command.Subcommands = append(command.Subcommands, newCommandFromSpec(subCmdSpec))
		}

		return command
	default:
		// shouldn't happen, spec validation should catch this
		panic("unrecognized command type '" + cmdSpec.Type + "'")
	}
}
