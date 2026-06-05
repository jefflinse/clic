package exec

import (
	"fmt"
	"os"
	osexec "os/exec"
	"strings"

	"github.com/jefflinse/clic/ioutil"
	"github.com/jefflinse/clic/provider"
	"github.com/spf13/cobra"
)

// Spec describes the provider.
type Spec struct {
	Name       string                `json:"name"             yaml:"name"`
	Args       []string              `json:"args,omitempty"   yaml:"args,omitempty"`
	Parameters provider.ParameterSet `json:"params,omitempty" yaml:"params,omitempty"`
	Echo       bool                  `json:"echo,omitempty"   yaml:"echo,omitempty"`
}

// New creates a new provider.
func New(v any) (provider.Provider, error) {
	s := Spec{}
	return &s, ioutil.Intermarshal(v, &s)
}

// Configure wires up the command's positional arguments, flags, and run behavior.
func (s *Spec) Configure(cmd *cobra.Command) {
	if usage := s.Parameters.ArgsUsage(); usage != "" {
		cmd.Use += " " + usage
	}

	s.Parameters.RegisterFlags(cmd.Flags())

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		name, cmdArgs, err := s.parameterizedNameAndArgs(cmd, args)
		if err != nil {
			return err
		}

		command := osexec.Command(name, cmdArgs...)
		command.Env = os.Environ()
		command.Stdin = os.Stdin
		command.Stdout = os.Stdout
		command.Stderr = os.Stderr

		if s.Echo {
			fmt.Printf("%s %s\n", name, strings.Join(cmdArgs, " "))
		}

		if err := command.Run(); err != nil {
			if exitErr, ok := err.(*osexec.ExitError); ok {
				os.Exit(exitErr.ProcessState.ExitCode())
			}
		}

		return nil
	}
}

// Type returns the type.
func (s *Spec) Type() string {
	return "exec"
}

// Validate validates the provider.
func (s *Spec) Validate() error {
	if s.Name == "" {
		return fmt.Errorf("invalid %s command spec: missing name", s.Type())
	} else if err := s.Parameters.Validate(); err != nil {
		return err
	}

	return nil
}

func (s *Spec) parameterizedNameAndArgs(cmd *cobra.Command, args []string) (string, []string, error) {
	name := s.Name
	rawArgs := make([]string, len(s.Args))
	copy(rawArgs, s.Args)

	if err := s.Parameters.ResolveValues(cmd, args); err != nil {
		return "", nil, err
	}

	name = s.Parameters.InjectValues(name)
	for i := range rawArgs {
		rawArgs[i] = s.Parameters.InjectValues(rawArgs[i])
	}

	// strip out empty arguments
	resolved := []string{}
	for _, arg := range rawArgs {
		if arg != "" {
			resolved = append(resolved, arg)
		}
	}

	return name, resolved, nil
}
