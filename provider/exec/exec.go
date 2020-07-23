package exec

import (
	"fmt"
	"os"
	osexec "os/exec"
	"strings"

	"github.com/jefflinse/handyman/provider"
	"github.com/urfave/cli/v2"
)

// Spec describes the provider.
type Spec struct {
	Path string `json:"path"`
}

// New creates a new provider.
func New(v interface{}) (provider.Provider, error) {
	s := Spec{}
	if err := provider.Intermarshal(v, &s); err != nil {
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
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}

		fmt.Fprint(os.Stdout, output.String())
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

// Validate validates the provider.
func (s Spec) Validate() error {
	if s.Path == "" {
		return fmt.Errorf("invalid %s command spec: missing path", s.Type())
	}

	return nil
}
