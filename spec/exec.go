package spec

import (
	"fmt"
	"strings"
)

// Exec is a provider for running any arbitrary local command.
type Exec struct {
	Path       string       `json:"path"`
	Args       []string     `json:"args,omitempty"`
	Parameters ParameterSet `json:"params,omitempty"`
}

// Name returns the name of the provider.
func (e Exec) Name() string {
	return "exec"
}

// TraceString prints the provider hierarchy.
func (e Exec) TraceString() string {
	return fmt.Sprintf("(exec): %s %s", e.Path, strings.Join(e.Args, " "))
}

// Validate returns an error if the provider is invalid.
func (e Exec) Validate() (Provider, error) {
	if e.Path == "" {
		return e, fmt.Errorf("invalid exec provider: missing name")
	}

	validatedParams, err := e.Parameters.Validate()
	if err != nil {
		return e, err
	}

	return Exec{
		Path:       e.Path,
		Args:       e.Args,
		Parameters: validatedParams,
	}, nil
}
