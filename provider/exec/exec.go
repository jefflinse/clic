package exec

import (
	"fmt"
	"os"
	osexec "os/exec"
	"strings"

	"github.com/jefflinse/handyman/ioutil"
	"github.com/jefflinse/handyman/provider"
	"github.com/urfave/cli/v2"
)

const (
	parameterTemplate = "{{params.%s}}"
)

// Spec describes the provider.
type Spec struct {
	Name       string       `json:"name"             yaml:"name"`
	Args       []string     `json:"args,omitempty"   yaml:"args,omitempty"`
	Parameters []*Parameter `json:"params,omitempty" yaml:"params,omitempty"`
}

// New creates a new provider.
func New(v interface{}) (provider.Provider, error) {
	s := Spec{}
	return &s, ioutil.Intermarshal(v, &s)
}

// CLIActionFn creates a CLI action fuction.
func (s Spec) CLIActionFn() cli.ActionFunc {
	s.assignDefaultParameterValues()

	return func(ctx *cli.Context) error {
		s.assignFlagParameterValues(ctx)

		name, args := s.parameterizedNameAndArgs()
		command := osexec.Command(name, args...)
		command.Env = os.Environ()
		command.Stdin = os.Stdin
		command.Stdout = os.Stdout
		command.Stderr = os.Stderr

		if err := command.Run(); err != nil {
			if exitErr, ok := err.(*osexec.ExitError); ok {
				os.Exit(exitErr.ProcessState.ExitCode())
			}
		}

		return nil
	}
}

// CLIFlags creates a set of CLI flags.
func (s Spec) CLIFlags() []cli.Flag {
	flags := []cli.Flag{}
	for _, param := range s.Parameters {
		var flag cli.Flag
		switch param.Type {
		case BoolParamType:
			flag = &cli.BoolFlag{
				Name:     toDashes(param.Name),
				Usage:    param.Description,
				Required: param.Required,
			}
		case IntParamType:
			flag = &cli.IntFlag{
				Name:     toDashes(param.Name),
				Usage:    param.Description,
				Required: param.Required,
			}
		case NumberParamType:
			flag = &cli.Float64Flag{
				Name:     toDashes(param.Name),
				Usage:    param.Description,
				Required: param.Required,
			}
		case StringParamType:
			flag = &cli.StringFlag{
				Name:     toDashes(param.Name),
				Usage:    param.Description,
				Required: param.Required,
			}
		}

		flags = append(flags, flag)
	}

	return flags
}

// Type returns the type.
func (s Spec) Type() string {
	return "exec"
}

// Validate validates the provider.
func (s Spec) Validate() error {
	if s.Name == "" {
		return fmt.Errorf("invalid %s command spec: missing name", s.Type())
	}

	return nil
}

// Copy parameter values from parameter defaults.
func (s *Spec) assignDefaultParameterValues() {
	for _, param := range s.Parameters {
		if param.Required {
			// required parameters don't use default values
			continue
		}

		// assign default values to optional parameters
		switch param.Type {
		case BoolParamType:
			if param.Default == nil {
				param.Default = false
			}
			param.value = param.Default.(bool)
		case IntParamType:
			if param.Default == nil {
				param.Default = 0.0
			}
			param.value = int(param.Default.(float64))
		case NumberParamType:
			if param.Default == nil {
				param.Default = 0.0
			}
			param.value = param.Default.(float64)
		case StringParamType:
			if param.Default == nil {
				param.Default = ""
			}
			param.value = param.Default.(string)
		}
	}
}

// Copy parameter values from flags that have been specified.
func (s *Spec) assignFlagParameterValues(ctx *cli.Context) {
	params := map[string]*Parameter{}
	for i := range s.Parameters {
		params[s.Parameters[i].Name] = s.Parameters[i]
	}

	for _, flagName := range ctx.LocalFlagNames() {
		param := params[toUnderscores(flagName)]

		switch param.Type {
		case BoolParamType:
			param.value = ctx.Bool(flagName)
		case IntParamType:
			param.value = ctx.Int(flagName)
		case NumberParamType:
			param.value = ctx.Float64(flagName)
		case StringParamType:
			param.value = ctx.String(flagName)
		}
	}
}

func (s *Spec) parameterizedNameAndArgs() (string, []string) {
	name := s.Name
	args := make([]string, len(s.Args))
	copy(args, s.Args)
	for _, param := range s.Parameters {
		placeholderStr := fmt.Sprintf(parameterTemplate, param.Name)
		value := fmt.Sprintf("%v", param.value)
		name = strings.ReplaceAll(s.Name, placeholderStr, value)
		for i, arg := range args {
			args[i] = strings.ReplaceAll(arg, placeholderStr, value)
		}
	}

	return name, args
}

// Underscores to dashes.
func toDashes(str string) string {
	return strings.ReplaceAll(str, "_", "-")
}

// Dashes to underscores.
func toUnderscores(str string) string {
	return strings.ReplaceAll(str, "-", "_")
}
