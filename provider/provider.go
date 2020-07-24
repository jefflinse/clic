package provider

import (
	"github.com/urfave/cli/v2"
)

// A Provider defines what happens when a command is invoked on the command line.
type Provider interface {
	CLIActionFn() cli.ActionFunc
	CLIFlags() []cli.Flag
	Type() string
	Validate() error
}
