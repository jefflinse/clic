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

// Spec describes the provider.
type Spec struct {
	Name       string      `json:"name"             yaml:"name"`
	Args       []string    `json:"args,omitempty"   yaml:"args,omitempty"`
	Parameters []Parameter `json:"params,omitempty" yaml:"params,omitempty"`
}

// New creates a new provider.
func New(v interface{}) (provider.Provider, error) {
	s := Spec{}
	return &s, ioutil.Intermarshal(v, &s)
}

// CLIActionFn creates a CLI action fuction.
func (s Spec) CLIActionFn() cli.ActionFunc {
	paramTypes := map[string]string{}
	for _, param := range s.Parameters {
		paramTypes[param.Name] = param.Type
	}

	return func(ctx *cli.Context) error {
		for _, flagName := range ctx.LocalFlagNames() {
			paramName := toUnderscores(flagName)
			var paramValue interface{}
			switch paramTypes[paramName] {
			case BoolParamType:
				paramValue = ctx.Bool(flagName)
			case IntParamType:
				paramValue = ctx.Int(flagName)
			case NumberParamType:
				paramValue = ctx.Float64(flagName)
			case StringParamType:
				paramValue = ctx.String(flagName)
			}

			// inject flag values into the command
			placeholderStr := fmt.Sprintf("{{params.%s}}", paramName)
			s.Name = strings.ReplaceAll(s.Name, placeholderStr, paramValue.(string))

			// inject flag values into the args
			for i, arg := range s.Args {
				s.Args[i] = strings.ReplaceAll(arg, placeholderStr, paramValue.(string))
			}
		}

		command := osexec.Command(s.Name, s.Args...)
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

// Underscores to dashes.
func toDashes(str string) string {
	return strings.ReplaceAll(str, "_", "-")
}

// Dashes to underscores.
func toUnderscores(str string) string {
	return strings.ReplaceAll(str, "-", "_")
}
