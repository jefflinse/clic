package exec

import (
	"fmt"
	osexec "os/exec"
	"strings"

	"github.com/jefflinse/handyman/command"
	"github.com/urfave/cli/v2"
)

// Spec describes the executor.
type Spec struct {
	Path string `json:"path"`
}

// New creates a new executor.
func New(v interface{}) (command.Executor, error) {
	s := Spec{}
	if err := command.Intermarshal(v, &s); err != nil {
		return nil, err
	}

	return &s, nil
}

// CLIActionFn creates a CLI action fuction.
func (s Spec) CLIActionFn() cli.ActionFunc {
	command := osexec.Command("/bin/bash", "-c", s.Path)
	output := strings.Builder{}
	command.Stdout = &output
	command.Stderr = &output
	return func(ctx *cli.Context) error {
		err := command.Run()
		fmt.Print(output.String())
		return err
	}
}

// CLIFlags creates a set of CLI flags.
func (s Spec) CLIFlags() []cli.Flag {
	return nil
}

// Type returns the type.
func (s Spec) Type() string {
	return "exec"
}

// Validate validates the executor.
func (s Spec) Validate() error {
	if s.Path == "" {
		return fmt.Errorf("invalid %s command spec: missing path", s.Type())
	}

	return nil
}
