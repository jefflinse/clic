package provider

import (
	"fmt"
	"strings"

	"github.com/jefflinse/handyman/ioutil"
	"github.com/urfave/cli/v2"
)

const (
	parameterTemplate = "{{params.%s}}"
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

// CLIFlagName returns the parameter name formatted as a CLI flag name.
func (param *Parameter) CLIFlagName() string {
	return toDashes(param.Name)
}

// CreateCLIFlag creates a CLI flag for this parameter.
func (param *Parameter) CreateCLIFlag() cli.Flag {
	var flag cli.Flag
	switch param.Type {
	case "bool":
		flag = &cli.BoolFlag{
			Name:     param.CLIFlagName(),
			Usage:    param.Description,
			Required: param.Required,
		}
	case "int":
		flag = &cli.IntFlag{
			Name:     param.CLIFlagName(),
			Usage:    param.Description,
			Required: param.Required,
		}
	case "number":
		flag = &cli.Float64Flag{
			Name:     param.CLIFlagName(),
			Usage:    param.Description,
			Required: param.Required,
		}
	case "string":
		flag = &cli.StringFlag{
			Name:     param.CLIFlagName(),
			Usage:    param.Description,
			Required: param.Required,
		}
	}

	return flag
}

// SetDefaultValue assigns the default value to the parameter.
func (param *Parameter) SetDefaultValue() {
	if param.Required {
		// required parameters don't use default values
		return
	} else if param.Default == nil {
		// parameters with no default resolve to empty string
		param.SetValue("")
		return
	}

	// assign default value based on parameter type
	switch param.Type {
	case BoolParamType:
		param.SetValue(param.Default.(bool))
	case IntParamType:
		param.SetValue(int(param.Default.(float64)))
	case NumberParamType:
		param.SetValue(param.Default.(float64))
	case StringParamType:
		param.SetValue(param.Default.(string))
	}
}

// SetValue assigns a value to the parameter.
func (param *Parameter) SetValue(value interface{}) {
	param.value = value
}

// Value returns the parameter's assigned value.
func (param *Parameter) Value() interface{} {
	return param.value
}

// Validate validates a Parameter.
func (param *Parameter) Validate() error {
	if param.Name == "" {
		return NewInvalidParameterSpecError("param missing name")
	} else if param.Type == "" {
		return NewInvalidParameterSpecError(fmt.Sprintf("param '%s' missing type", param.Name))
	} else if param.Default != nil {
		if param.Required {
			return NewInvalidParameterSpecError(fmt.Sprintf("required param '%s' cannot have default value", param.Name))
		}

		switch param.Type {
		case BoolParamType:
			if _, ok := param.Default.(bool); !ok {
				return NewInvalidParameterSpecError(
					fmt.Sprintf("invalid default value '%v' for param '%s' (type %s)", param.Default, param.Name, param.Type),
				)
			}
		case IntParamType:
			if _, ok := param.Default.(int); !ok {
				return NewInvalidParameterSpecError(
					fmt.Sprintf("invalid default value '%v' for param '%s' (type %s)", param.Default, param.Name, param.Type),
				)
			}
		case NumberParamType:
			if _, ok := param.Default.(float64); !ok {
				return NewInvalidParameterSpecError(
					fmt.Sprintf("invalid default value '%v' for param '%s' (type %s)", param.Default, param.Name, param.Type),
				)
			}
		case StringParamType:
			if _, ok := param.Default.(string); !ok {
				return NewInvalidParameterSpecError(
					fmt.Sprintf("invalid default value '%v' for param '%s' (type %s)", param.Default, param.Name, param.Type),
				)
			}
		default:
			return NewInvalidParameterSpecError(fmt.Sprintf("unknown type '%s' for param '%s'", param.Type, param.Name))
		}
	} else {
		switch param.Type {
		case BoolParamType:
		case IntParamType:
		case NumberParamType:
		case StringParamType:
		default:
			return NewInvalidParameterSpecError(fmt.Sprintf("unknown type '%s' for param '%s'", param.Type, param.Name))
		}
	}

	return nil
}

// NewInvalidParameterSpecError creates a new error indicating that a parameter spec is invalid.
func NewInvalidParameterSpecError(reason string) error {
	return fmt.Errorf("invalid parameter spec: %s", reason)
}

// A ParameterSet is a slice of parameter pointers.
type ParameterSet []*Parameter

// CreateCLIFlags creates a set of CLI flags for this parameter set.
func (ps ParameterSet) CreateCLIFlags() []cli.Flag {
	flags := []cli.Flag{}
	for _, param := range ps {
		flags = append(flags, param.CreateCLIFlag())
	}

	return flags
}

// InjectValues replaces all param references with their corresponding values in the given string.
func (ps ParameterSet) InjectValues(str string) string {
	result := str
	for _, param := range ps {
		placeholderStr := fmt.Sprintf(parameterTemplate, param.Name)
		value := fmt.Sprintf("%v", param.Value())
		result = strings.ReplaceAll(result, placeholderStr, value)
	}

	return result
}

// ResolveValues assigns values to the parameters from defaults and the CLI context.
func (ps ParameterSet) ResolveValues(ctx *cli.Context) {
	for _, p := range ps {
		p.SetDefaultValue()
		if ctx == nil {
			continue
		}

		for _, flagName := range ctx.LocalFlagNames() {
			if flagName != p.CLIFlagName() {
				continue
			}

			switch p.Type {
			case BoolParamType:
				p.SetValue(ctx.Bool(flagName))
			case IntParamType:
				p.SetValue(ctx.Int(flagName))
			case NumberParamType:
				p.SetValue(ctx.Float64(flagName))
			case StringParamType:
				p.SetValue(ctx.String(flagName))
			}
		}
	}
}

// Validate validates the parameter set, returning the first error it encounters, if any.
func (ps ParameterSet) Validate() error {
	for _, param := range ps {
		if err := param.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// Underscores to dashes.
func toDashes(str string) string {
	return strings.ReplaceAll(str, "_", "-")
}

// Dashes to underscores.
func toUnderscores(str string) string {
	return strings.ReplaceAll(str, "-", "_")
}
