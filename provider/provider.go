package provider

import (
	"github.com/spf13/cobra"
)

// A Provider defines what happens when a command is invoked on the command line.
type Provider interface {
	// Configure wires the provider's positional arguments, flags, and run
	// behavior onto the given cobra command.
	Configure(cmd *cobra.Command)
	Type() string
	Validate() error
}
