package noop

import (
	"github.com/jefflinse/clic/provider"
	"github.com/spf13/cobra"
)

// Spec describes the provider.
type Spec struct {
}

// New creates a new provider.
func New(v any) (provider.Provider, error) {
	// all properties on a noop command are ignored
	return &Spec{}, nil
}

// Configure wires up the command's run behavior.
func (s *Spec) Configure(cmd *cobra.Command) {
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return nil
	}
}

// Type returns the type.
func (s *Spec) Type() string {
	return "noop"
}

// Validate validates the provider.
func (s *Spec) Validate() error {
	return nil
}
