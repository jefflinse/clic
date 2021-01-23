package spec

import (
	"fmt"
)

// A Command is a command that can be executed on the command line with args and/or flags.
type Command struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`

	// providers
	Exec Exec `json:"exec,omitempty"`
}

// Validate returns an error if the command is invalid.
func (c Command) Validate() (Command, error) {
	if c.Name == "" {
		return c, fmt.Errorf("invalid command spec: missing name")
	}

	// require exactly one provider, specifying >1 is undefined behavior
	var provider Provider
	for _, p := range []Provider{c.Exec} {
		provider = p
		if provider == nil {
			continue
		}
	}

	if provider == nil {
		return c, fmt.Errorf("invalid command spec: missing provider")
	}

	vp, err := provider.Validate()
	if err != nil {
		return c, err
	}

	return Command{
		Name:        c.Name,
		Description: c.Description,
		Exec:        vp.(Exec),
	}, nil
}
