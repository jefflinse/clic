package spec

import (
	"fmt"
)

// A Parameter represents a configurable aspect of a command, and is represented as an argument or flag.
type Parameter struct {
	Name     string `json:"name"`
	Type     string `json:"type,omitempty"`
	Required bool   `json:"required,omitempty"`
}

// Validate returns an error if the parameter spec is invalid.
func (p Parameter) Validate() error {
	if p.Name == "" {
		return fmt.Errorf("invalid parameter spec: missing name")
	}

	return nil
}

// A ParameterSet is a set of parameters.
type ParameterSet []Parameter
