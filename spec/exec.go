package spec

import (
	"fmt"
	"strings"
)

// Exec is a provider for running any arbitrary local command.
type Exec struct {
	Path   string       `json:"path"`
	Args   []string     `json:"args,omitempty"`
	Params ParameterSet `json:"parameters,omitempty"`
}

// TraceString prints the provider hierarchy.
func (e Exec) TraceString() string {
	return fmt.Sprintf("(exec): %s %s", e.Path, strings.Join(e.Args, " "))
}

// Validate returns an error if the provider is invalid.
func (e Exec) Validate() error {
	if e.Path == "" {
		return fmt.Errorf("invalid exec provider: missing name")
	}

	for _, p := range e.Params {
		if err := p.Validate(); err != nil {
			return err
		}
	}

	return nil
}
