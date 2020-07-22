package spec

import (
	"encoding/json"
	"fmt"

	"github.com/jefflinse/handyman/commands"
	"github.com/jefflinse/handyman/commands/exec"
	"github.com/jefflinse/handyman/commands/lambda"
	"github.com/jefflinse/handyman/commands/noop"
	"github.com/urfave/cli/v2"
)

// A Command specifes an action or a set of subcommands.
type Command struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Executor    commands.Executor `json:"-"`
}

var requiredCommandFields = []string{
	"name",
	"description",
}

var commandMap = map[string]func(interface{}) (commands.Executor, error){
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

func (command Command) CLICommand() *cli.Command {
	return &cli.Command{
		Name:   command.Name,
		Usage:  command.Description,
		Action: command.Executor.CLIActionFn(),
		Flags:  command.Executor.CLIFlags(),
	}
}

// Validate validates a Command spec.
func (command Command) Validate() error {
	if command.Name == "" {
		return NewInvalidCommandSpecError("missing name")
	} else if command.Description == "" {
		return NewInvalidCommandSpecError("missing description")
	}

	return command.Executor.Validate()
}

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

	executorName := "noop"
	if len(content) < len(requiredCommandFields) {
		return fmt.Errorf("too few fields for command")
	} else if len(content) > len(requiredCommandFields) {
		potentialExecutorNames := []string{}
		for key := range content {
			keyIsRequiredField := false
			for _, field := range requiredCommandFields {
				if key == field {
					keyIsRequiredField = true
					break
				}
			}

			if keyIsRequiredField {
				continue
			}

			potentialExecutorNames = append(potentialExecutorNames, key)
		}

		if len(potentialExecutorNames) != 1 {
			return fmt.Errorf("invalid type data")
		}

		executorName = potentialExecutorNames[0]
	}

	executorCtor, ok := commandMap[executorName]
	if !ok {
		return fmt.Errorf("invalid type '%s'", executorName)
	}

	executor, err := executorCtor(content[executorName])
	if err != nil {
		return fmt.Errorf("can't create executor: %w", err)
	}

	c.Executor = executor

	return nil
}

// NewInvalidCommandSpecError creates a new error indicating that a command spec is invalid.
func NewInvalidCommandSpecError(reason string) error {
	return fmt.Errorf("invalid command spec: %s", reason)
}
