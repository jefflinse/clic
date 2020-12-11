package spec

import (
	"fmt"
)

// A Command is a command that can be executed on the command line with args and/or flags.
type Command struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`

	// providers
	Exec *Exec `json:"exec,omitempty"`
}

func (c Command) provider() Provider {
	return c.Exec
}

// TraceString prints the command hierarchy.
func (c Command) TraceString() string {
	return fmt.Sprintf("%s %s", c.Name, c.provider().TraceString())
}

// Validate returns an error if the command is invalid.
func (c Command) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("invalid command spec: missing name")
	}

	// require exactly one provider
	if c.Exec == nil {
		return fmt.Errorf("invalid command spec: missing provider")
	}

	return nil
}
