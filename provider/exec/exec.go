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
	Name       string                `json:"name"             yaml:"name"`
	Args       []string              `json:"args,omitempty"   yaml:"args,omitempty"`
	Parameters provider.ParameterSet `json:"params,omitempty" yaml:"params,omitempty"`
	Echo       bool                  `json:"echo,omitempty"   yaml:"echo,omitempty"`
}

// New creates a new provider.
func New(v interface{}) (provider.Provider, error) {
	s := Spec{}
	return &s, ioutil.Intermarshal(v, &s)
}

// CLIActionFn creates a CLI action fuction.
func (s Spec) CLIActionFn() cli.ActionFunc {
	return func(ctx *cli.Context) error {
		name, args, err := s.parameterizedNameAndArgs(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n\n", err)
			cli.ShowCommandHelpAndExit(ctx, ctx.Command.Name, 1)
		}

		command := osexec.Command(name, args...)
		command.Env = os.Environ()
		command.Stdin = os.Stdin
		command.Stdout = os.Stdout
		command.Stderr = os.Stderr

		if s.Echo {
			fmt.Printf("%s %s\n", name, strings.Join(args, " "))
		}

		if err := command.Run(); err != nil {
			if exitErr, ok := err.(*osexec.ExitError); ok {
				os.Exit(exitErr.ProcessState.ExitCode())
			}
		}

		return nil
	}
}

// ArgsUsage returns usage text for the arguments.
func (s Spec) ArgsUsage() string {
	argNames := []string{}
	for _, param := range s.Parameters {
		if param.Required {
			argNames = append(argNames, param.CLIFlagName())
		}
	}

	return strings.Join(argNames, " ")
}

// CLIFlags creates a set of CLI flags.
func (s Spec) CLIFlags() []cli.Flag {
	return s.Parameters.CreateCLIFlags()
}

// Type returns the type.
func (s Spec) Type() string {
	return "exec"
}

// Validate validates the provider.
func (s Spec) Validate() error {
	if s.Name == "" {
		return fmt.Errorf("invalid %s command spec: missing name", s.Type())
	} else if err := s.Parameters.Validate(); err != nil {
		return err
	}

	return nil
}

func (s *Spec) parameterizedNameAndArgs(ctx *cli.Context) (string, []string, error) {
	name := s.Name
	args := make([]string, len(s.Args))
	copy(args, s.Args)

	if err := s.Parameters.ResolveValues(ctx); err != nil {
		return "", nil, err
	}

	name = s.Parameters.InjectValues(name)
	for i := range args {
		args[i] = s.Parameters.InjectValues(args[i])
	}

	return name, args, nil
}
