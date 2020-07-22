package exec

import (
	"encoding/json"
	"fmt"
	osexec "os/exec"
	"strings"

	"github.com/jefflinse/handyman/command"
	"github.com/urfave/cli/v2"
)

type Spec struct {
	Path string `json:"path"`
}

func New(v interface{}) (command.Executor, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}

	s := Spec{}
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}

	return s, nil
}

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

func (s Spec) CLIFlags() []cli.Flag {
	return nil
}

func (s Spec) Type() string {
	return "exec"
}

func (s Spec) Validate() error {
	if s.Path == "" {
		return fmt.Errorf("invalid %s command spec: missing path", s.Type())
	}

	return nil
}
