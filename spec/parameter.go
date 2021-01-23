package spec

import (
	"fmt"
)

// Constants
const (
	ArgParameter  = "arg"
	FlagParameter = "flag"

	StringType = "string"
)

// A Parameter represents a configurable aspect of a command, and is represented as an argument or flag.
type Parameter struct {
	Name     string `json:"name"`
	As       string `json:"as,omitempty"`       // 'arg' (default) or 'flag'
	Type     string `json:"type,omitempty"`     // type of value, e.g. 'string', 'bool', etc.
	Required *bool  `json:"required,omitempty"` // args are always required, flags can be required (default = false)
}

// Validate returns an error if the parameter spec is invalid.
func (p Parameter) Validate() (Parameter, error) {
	if p.Name == "" {
		return p, fmt.Errorf("invalid parameter spec: missing name")
	}

	// assume string type by default
	if p.Type == "" {
		p.Type = StringType
	} else if p.Type != StringType {
		return p, fmt.Errorf("invalid parameter spec: '%s' is a not a valid parameter type", p.Type)
	}

	// assume all parameters are required by default
	if p.Required == nil {
		p.Required = boolPtr(false)
	}

	// assume all parameters are args by default, unless required = false
	if p.As == "" {
		p.As = ArgParameter
		if !*p.Required {
			p.As = FlagParameter
		}
	}

	// can't set required = false for arg parameters
	if p.As == ArgParameter && !*p.Required {
		return p, fmt.Errorf("invalid parameter spec: arg '%s' cannot be optional", p.Name)
	}

	return p, nil
}

// A ParameterSet is a set of parameters.
type ParameterSet []Parameter

// Validate returns an error if the ParameterSet is invalid.
func (ps ParameterSet) Validate() (ParameterSet, error) {
	vs := ParameterSet{}
	for _, p := range ps {
		vp, err := p.Validate()
		if err != nil {
			return ps, err
		}

		vs = append(vs, vp)
	}

	return vs, nil
}

func boolPtr(v bool) *bool {
	return &v
}
