package noop

import (
	"github.com/jefflinse/handyman/spec"
	"github.com/urfave/cli/v2"
)

// New creates a new command that does nothing.
func New(cmdSpec *spec.Command) *cli.Command {
	return &cli.Command{
		Name:   cmdSpec.Name,
		Usage:  cmdSpec.Description,
		Action: newActionFn(),
	}
}

// Creates the action function.
func newActionFn() cli.ActionFunc {
	return func(ctx *cli.Context) error {
		return nil
	}
}
