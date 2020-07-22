package noop

import (
	"github.com/jefflinse/handyman/command"
	"github.com/urfave/cli/v2"
)

type Spec struct {
}

func New(v interface{}) (command.Executor, error) {
	return &Spec{}, nil
}

func (s Spec) CLIActionFn() cli.ActionFunc {
	return func(ctx *cli.Context) error {
		return nil
	}
}

func (s Spec) CLIFlags() []cli.Flag {
	return nil
}

func (s Spec) Type() string {
	return "noop"
}

func (s Spec) Validate() error {
	return nil
}
