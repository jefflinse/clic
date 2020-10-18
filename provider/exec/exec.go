package exec

import (
	"fmt"
	"os"
	osexec "os/exec"

	"github.com/jefflinse/handyman/ioutil"
	"github.com/jefflinse/handyman/provider"
	"github.com/urfave/cli/v2"
)

// Spec describes the provider.
type Spec struct {
	Name       string                `json:"name"             yaml:"name"`
	Args       []string              `json:"args,omitempty"   yaml:"args,omitempty"`
	Parameters provider.ParameterSet `json:"params,omitempty" yaml:"params,omitempty"`
}

// New creates a new provider.
func New(v interface{}) (provider.Provider, error) {
	s := Spec{}
	return &s, ioutil.Intermarshal(v, &s)
}

// CLIActionFn creates a CLI action fuction.
func (s Spec) CLIActionFn() cli.ActionFunc {
	return func(ctx *cli.Context) error {
		name, args := s.parameterizedNameAndArgs(ctx)
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
		case provider.BoolParamType:
			flag = &cli.BoolFlag{
				Name:     param.CLIFlagName(),
				Usage:    param.Description,
				Required: param.Required,
			}
		case provider.IntParamType:
			flag = &cli.IntFlag{
				Name:     param.CLIFlagName(),
				Usage:    param.Description,
				Required: param.Required,
			}
		case provider.NumberParamType:
			flag = &cli.Float64Flag{
				Name:     param.CLIFlagName(),
				Usage:    param.Description,
				Required: param.Required,
			}
		case provider.StringParamType:
			flag = &cli.StringFlag{
				Name:     param.CLIFlagName(),
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

	for _, param := range s.Parameters {
		if err := param.Validate(); err != nil {
			return err
		}
	}

	return nil
}

func (s *Spec) parameterizedNameAndArgs(ctx *cli.Context) (string, []string) {
	name := s.Name
	args := make([]string, len(s.Args))
	copy(args, s.Args)

	s.Parameters.ResolveValues(ctx)
	name = s.Parameters.InjectValues(name)
	for i := range args {
		args[i] = s.Parameters.InjectValues(args[i])
	}

	return name, args
}
