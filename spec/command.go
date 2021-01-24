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
	REST REST `json:"rest,omitempty"`
}

// Provider retuns the provider for the command, or nil if none was specified.
func (c Command) Provider() Provider {
	var provider Provider
	for _, p := range []Provider{c.Exec, c.REST} {
		if !p.IsEmpty() {
			provider = p
			break
		}
	}

	return provider
}

// Validate returns an error if the command is invalid.
func (c Command) Validate() (Command, error) {
	if c.Name == "" {
		return c, fmt.Errorf("invalid command spec: missing name")
	}

	// require exactly one provider, specifying >1 is undefined behavior
	provider := c.Provider()
	if provider == nil {
		return c, fmt.Errorf("invalid command spec: missing provider")
	}

	vp, err := provider.Validate()
	if err != nil {
		return c, err
	}

	vc := Command{
		Name:        c.Name,
		Description: c.Description,
	}

	if vp.Name() == "exec" {
		execProvider, _ := vp.(Exec)
		vc.Exec = execProvider
	} else if vp.Name() == "rest" {
		restProvider, _ := vp.(REST)
		vc.REST = restProvider
	}

	return vc, nil
}
