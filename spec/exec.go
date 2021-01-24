package spec

import (
	"fmt"
)

// Exec is a provider for running any arbitrary local command.
type Exec struct {
	Path       string       `json:"path"`
	Args       []string     `json:"args,omitempty"`
	Parameters ParameterSet `json:"params,omitempty"`
}

// GetParameters returns the set of parameters for the provider.
func (e Exec) GetParameters() ParameterSet {
	return e.Parameters
}

// IsEmpty returns true if all of the fields on the provider are empty.
func (e Exec) IsEmpty() bool {
	return e.Path == "" && len(e.Args) == 0 && len(e.Parameters) == 0
}

// Name returns the name of the provider.
func (e Exec) Name() string {
	return "exec"
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
