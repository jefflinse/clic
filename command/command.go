package command

import (
	"encoding/json"

	"github.com/urfave/cli/v2"
)

// An Executor defines what happens when a command is invoked on the command line.
type Executor interface {
	CLIActionFn() cli.ActionFunc
	CLIFlags() []cli.Flag
	Type() string
	Validate() error
}

func Intermarshal(executor interface{}, target interface{}) error {
	data, err := json.Marshal(executor)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, &target); err != nil {
		return err
	}

	return nil
}
