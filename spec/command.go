package spec

import (
	"encoding/json"
	"fmt"

	"github.com/jefflinse/handyman/command"
	"github.com/jefflinse/handyman/command/exec"
	"github.com/jefflinse/handyman/command/lambda"
	"github.com/jefflinse/handyman/command/noop"
	"github.com/urfave/cli/v2"
)

// A Command specifes an action or a set of subcommands.
type Command struct {
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Executor    command.Executor `json:"-"`
}

var requiredCommandFields = []string{
	"name",
	"description",
}

var commandMap = map[string]func(interface{}) (command.Executor, error){
	"exec":   exec.New,
	"lambda": lambda.New,
	"noop":   noop.New,
}

// NewCommandSpec creates a new Command from the provided spec.
func NewCommandSpec(content []byte) (*Command, error) {
	command := &Command{}
	if err := json.Unmarshal(content, command); err != nil {
		return nil, err
	}

	return command, nil
}

// CLICommand creates a CLI command for this command.
func (c Command) CLICommand() *cli.Command {
	return &cli.Command{
		Name:   c.Name,
		Usage:  c.Description,
		Action: c.Executor.CLIActionFn(),
		Flags:  c.Executor.CLIFlags(),
	}
}

// UnmarshalJSON unmarshals the specified JSON data into the command.
func (c *Command) UnmarshalJSON(data []byte) error {
	type commandMetadata struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}

	metadata := commandMetadata{}
	if err := json.Unmarshal(data, &metadata); err != nil {
		return err
	}

	c.Name = metadata.Name
	c.Description = metadata.Description

	content := map[string]interface{}{}
	if err := json.Unmarshal(data, &content); err != nil {
		return err
	}

	// the executor type is the remaining non-required field name
	if len(content) == len(requiredCommandFields) {
		// don't bother looking for an executor if not enough fields are provided
		return nil
	} else if len(content) > len(requiredCommandFields) {
		for key := range content {
			isRequiredField := false
			for _, field := range requiredCommandFields {
				if key == field {
					isRequiredField = true
					break
				}
			}

			// use the first available executor we match
			if !isRequiredField {
				if executorCtor, ok := commandMap[key]; ok {
					executor, err := executorCtor(content[key])
					if err != nil {
						return err
					}

					c.Executor = executor
					return nil
				}
			}
		}
	}

	return nil
}

// Validate validates a Command spec.
func (c Command) Validate() error {
	if c.Name == "" {
		return NewInvalidCommandSpecError("missing name")
	} else if c.Description == "" {
		return NewInvalidCommandSpecError("missing description")
	} else if c.Executor == nil {
		return NewInvalidCommandSpecError("missing executor")
	}

	return c.Executor.Validate()
}

// NewInvalidCommandSpecError creates a new error indicating that a command spec is invalid.
func NewInvalidCommandSpecError(reason string) error {
	return fmt.Errorf("invalid command spec: %s", reason)
}
