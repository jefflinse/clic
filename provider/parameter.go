package provider

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/jefflinse/clic/ioutil"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	parameterTemplate = "{{params.%s}}"
)

// A Parameter specifies a command parameter.
type Parameter struct {
	Name        string `json:"name"                  yaml:"name"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	Type        string `json:"type"                  yaml:"type"`
	Required    bool   `json:"required"              yaml:"required"`
	Default     any    `json:"default,omitempty"     yaml:"default,omitempty"`
	AsFlag      string `json:"as_flag,omitempty"     yaml:"as_flag,omitempty"`

	value any
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

// CLIFlagName returns the parameter name formatted as a CLI flag name. The
// original Name is preserved for use as the request key (header/query/body);
// only the flag spelling is normalized to lower-kebab-case.
func (param *Parameter) CLIFlagName() string {
	return strings.ToLower(toDashes(param.Name))
}

// registerFlag registers the parameter as a flag on the given flag set.
func (param *Parameter) registerFlag(flags *pflag.FlagSet) {
	name, usage := param.CLIFlagName(), param.Description
	switch param.Type {
	case BoolParamType:
		flags.Bool(name, false, usage)
	case IntParamType:
		flags.Int(name, 0, usage)
	case NumberParamType:
		flags.Float64(name, 0, usage)
	case StringParamType:
		flags.String(name, "", usage)
	}
}

// setFromFlag assigns the parameter's value from its corresponding flag.
func (param *Parameter) setFromFlag(flags *pflag.FlagSet) {
	switch param.Type {
	case BoolParamType:
		value, _ := flags.GetBool(param.CLIFlagName())
		param.SetValue(value)
	case IntParamType:
		value, _ := flags.GetInt(param.CLIFlagName())
		param.SetValue(value)
	case NumberParamType:
		value, _ := flags.GetFloat64(param.CLIFlagName())
		param.SetValue(value)
	case StringParamType:
		value, _ := flags.GetString(param.CLIFlagName())
		param.SetValue(value)
	}
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
func (param *Parameter) SetValue(value any) {
	param.value = value
}

// Value returns the parameter's assigned value.
func (param *Parameter) Value() any {
	if param.Type == BoolParamType {
		value, _ := param.value.(bool)
		if param.AsFlag != "" {
			if value {
				return param.AsFlag
			}

			return ""
		}
	}

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

// ArgsUsage returns a usage string describing the set's required positional arguments.
func (ps ParameterSet) ArgsUsage() string {
	names := []string{}
	for _, param := range ps.Required() {
		names = append(names, "<"+param.CLIFlagName()+">")
	}

	return strings.Join(names, " ")
}

// RegisterFlags registers the set's optional parameters as flags on the given flag set.
func (ps ParameterSet) RegisterFlags(flags *pflag.FlagSet) {
	for _, param := range ps.Optional() {
		param.registerFlag(flags)
	}
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

// Optional returns a subset of the ParameterSet containing only optional parameters.
func (ps ParameterSet) Optional() ParameterSet {
	optional := ParameterSet{}
	for _, param := range ps {
		if !param.Required {
			optional = append(optional, param)
		}
	}

	return optional
}

// Required returns a subset of the ParameterSet containing only required parameters.
func (ps ParameterSet) Required() ParameterSet {
	required := ParameterSet{}
	for _, param := range ps {
		if param.Required {
			required = append(required, param)
		}
	}

	return required
}

// ResolveValues assigns values to the parameters from the positional arguments,
// flags, and defaults provided via the cobra command.
func (ps ParameterSet) ResolveValues(cmd *cobra.Command, args []string) error {
	// assign required parameters from positional args, in order
	for _, p := range ps.Required() {
		if len(args) == 0 {
			return fmt.Errorf("missing required argument: %s", p.CLIFlagName())
		}

		p.SetValue(args[0])
		args = args[1:]
	}

	if len(args) > 0 {
		return fmt.Errorf("unexpected argument(s): %v", strings.Join(args, " "))
	}

	// assign optional parameters from flags, falling back to defaults
	flags := cmd.Flags()
	for _, p := range ps.Optional() {
		p.SetDefaultValue()

		if flags.Changed(p.CLIFlagName()) {
			p.setFromFlag(flags)
		}
	}

	return nil
}

// RegisterAsFlags registers every parameter in the set as a flag, marking
// required parameters as required flags on the command.
func (ps ParameterSet) RegisterAsFlags(cmd *cobra.Command) {
	for _, param := range ps {
		param.registerFlag(cmd.Flags())
		if param.Required {
			_ = cmd.MarkFlagRequired(param.CLIFlagName())
		}
	}
}

// ResolveFromFlags assigns every parameter's value from its flag, applying
// defaults for optional parameters that were not set.
func (ps ParameterSet) ResolveFromFlags(cmd *cobra.Command) {
	flags := cmd.Flags()
	for _, p := range ps {
		p.SetDefaultValue()

		if flags.Changed(p.CLIFlagName()) {
			p.setFromFlag(flags)
		}
	}
}

// InjectPathValues substitutes {name} placeholders in a URL path template with
// the URL-escaped values of the matching parameters.
func (ps ParameterSet) InjectPathValues(endpoint string) string {
	result := endpoint
	for _, param := range ps {
		placeholder := "{" + param.Name + "}"
		value := url.PathEscape(fmt.Sprintf("%v", param.Value()))
		result = strings.ReplaceAll(result, placeholder, value)
	}

	return result
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
