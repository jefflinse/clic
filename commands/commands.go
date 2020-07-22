package commands

import (
	"github.com/urfave/cli/v2"
)

// An Executor defines what happens when a command is invoked on the command line.
type Executor interface {
	CLIActionFn() cli.ActionFunc
	CLIFlags() []cli.Flag
	Type() string
	Validate() error
}
