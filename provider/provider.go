package provider

import (
	"encoding/json"

	"github.com/urfave/cli/v2"
)

// A Provider defines what happens when a command is invoked on the command line.
type Provider interface {
	CLIActionFn() cli.ActionFunc
	CLIFlags() []cli.Flag
	Type() string
	Validate() error
}

// Intermarshal marshals the (unknown) provider object to JSON and then unmarshals it back to the target type.
func Intermarshal(provider interface{}, target interface{}) error {
	data, err := json.Marshal(provider)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, &target); err != nil {
		return err
	}

	return nil
}
