package noop

import (
	"context"

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

// Sections reports that a noop command takes no interactive input.
func (s *Spec) Sections() []provider.Section {
	return nil
}

// Execute does nothing and returns an empty text result.
func (s *Spec) Execute(_ context.Context, _ provider.Inputs) (*provider.Result, error) {
	return &provider.Result{Kind: provider.ResultText, Body: []byte("(noop)")}, nil
}

// Type returns the type.
func (s *Spec) Type() string {
	return "noop"
}

// Validate validates the provider.
func (s *Spec) Validate() error {
	return nil
}
