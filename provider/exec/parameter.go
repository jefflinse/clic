package exec

import (
	"fmt"

	"github.com/jefflinse/handyman/ioutil"
)

// A Parameter specifies a command parameter.
type Parameter struct {
	Name        string      `json:"name"                  yaml:"name"`
	Description string      `json:"description,omitempty" yaml:"description,omitempty"`
	Type        string      `json:"type"                  yaml:"type"`
	Required    bool        `json:"required"              yaml:"required"`
	Default     interface{} `json:"default,omitempty"     yaml:"default,omitempty"`

	value interface{}
}

const (
	// BoolParamType is a bool parameter.
	BoolParamType = "bool"

	// IntParamType is an int parameter.
	IntParamType = "int"

	// NumberParamType is a number parameter.
	NumberParamType = "number"

	// StringParamType is a string parameter.
	StringParamType = "string"
)

// NewParameter creates a new Parameter from the provided spec.
func NewParameter(content []byte) (*Parameter, error) {
	param := &Parameter{}
	if err := ioutil.Unmarshal(content, param); err != nil {
		return nil, NewInvalidParameterSpecError(err.Error())
	}

	return param, nil
}

// Validate validates a Parameter.
func (param Parameter) Validate() error {
	if param.Name == "" {
		return NewInvalidParameterSpecError("missing name")
	} else if param.Type == "" {
		return NewInvalidParameterSpecError("missing type")
	} else {
		switch param.Type {
		case BoolParamType:
		case IntParamType:
		case NumberParamType:
		case StringParamType:
		default:
			return NewInvalidParameterSpecError(fmt.Sprintf("unknown type: %s", param.Type))
		}
	}

	return nil
}

// NewInvalidParameterSpecError creates a new error indicating that a parameter spec is invalid.
func NewInvalidParameterSpecError(reason string) error {
	return fmt.Errorf("invalid parameter spec: %s", reason)
}
