package noop

import (
	"github.com/jefflinse/handyman/provider"
	"github.com/urfave/cli/v2"
)

// Spec describes the provider.
type Spec struct {
}

// New creates a new provider.
func New(v interface{}) (provider.Provider, error) {
	return &Spec{}, nil
}

// CLIActionFn creates a CLI action fuction.
func (s Spec) CLIActionFn() cli.ActionFunc {
	return func(ctx *cli.Context) error {
		return nil
	}
}

// CLIFlags creates a set of CLI flags.
func (s Spec) CLIFlags() []cli.Flag {
	return nil
}

// Type returns the type.
func (s Spec) Type() string {
	return "noop"
}

// Validate validates the provider.
func (s Spec) Validate() error {
	return nil
}
