package exec

import (
	"context"
	"fmt"
	"os"
	osexec "os/exec"
	"strings"
	"time"

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

// Preview reports the resolved command line and the headless CLI arguments that
// reproduce it, without running anything.
func (s *Spec) Preview(_ context.Context, in provider.Inputs) (*provider.RequestPreview, error) {
	s.Parameters.Assign(in.Scalars["params"])
	name, args := s.resolvedNameAndArgs()
	return &provider.RequestPreview{
		Kind:    provider.ResultText,
		Display: strings.TrimSpace(name + " " + strings.Join(args, " ")),
		CLIArgs: cliArgs(s.Parameters),
	}, nil
}

// cliArgs renders a parameter set as headless CLI arguments: required parameters
// are positional (in declared order), optional parameters are flags.
func cliArgs(params provider.ParameterSet) []string {
	var args []string
	for _, p := range params.Required() {
		args = append(args, fmt.Sprintf("%v", p.Value()))
	}
	for _, p := range params.Optional() {
		if v := fmt.Sprintf("%v", p.Value()); v != "" {
			args = append(args, "--"+p.CLIFlagName()+"="+v)
		}
	}
	return args
}

func (s *Spec) parameterizedNameAndArgs(cmd *cobra.Command, args []string) (string, []string, error) {
	if err := s.Parameters.ResolveValues(cmd, args); err != nil {
		return "", nil, err
	}

	name, resolved := s.resolvedNameAndArgs()
	return name, resolved, nil
}

// resolvedNameAndArgs substitutes the already-assigned parameter values into
// the command name and arguments, dropping any argument that resolves to empty.
func (s *Spec) resolvedNameAndArgs() (string, []string) {
	name := s.Parameters.InjectValues(s.Name)

	resolved := []string{}
	for _, arg := range s.Args {
		if injected := s.Parameters.InjectValues(arg); injected != "" {
			resolved = append(resolved, injected)
		}
	}

	return name, resolved
}

// Summary describes the command in one line, e.g. "git status".
func (s *Spec) Summary() string {
	return strings.TrimSpace(s.Name + " " + strings.Join(s.Args, " "))
}

// Sections describes the command's parameters for interactive entry.
func (s *Spec) Sections() []provider.Section {
	if len(s.Parameters) == 0 {
		return nil
	}
	return []provider.Section{{Key: "params", Title: "Arguments", Fields: s.Parameters.Fields()}}
}

// Execute assigns the collected parameter values, runs the command, and returns
// its combined output (stdout+stderr) as a text result. The process exit code
// is reported in the result rather than terminating clic.
func (s *Spec) Execute(ctx context.Context, in provider.Inputs) (*provider.Result, error) {
	s.Parameters.Assign(in.Scalars["params"])
	name, args := s.resolvedNameAndArgs()

	command := osexec.CommandContext(ctx, name, args...)
	command.Env = os.Environ()

	start := time.Now()
	out, err := command.CombinedOutput()
	latency := time.Since(start)

	status := 0
	if exitErr, ok := err.(*osexec.ExitError); ok {
		status = exitErr.ProcessState.ExitCode()
	} else if err != nil {
		return nil, err
	}

	return &provider.Result{
		Kind:        provider.ResultText,
		RequestLine: strings.TrimSpace(name + " " + strings.Join(args, " ")),
		Status:      status,
		Latency:     latency,
		Body:        out,
	}, nil
}
