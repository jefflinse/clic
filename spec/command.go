package spec

import (
	"encoding/json"
	"fmt"

	"github.com/jefflinse/handyman/provider"
	"github.com/jefflinse/handyman/provider/exec"
	"github.com/jefflinse/handyman/provider/lambda"
	"github.com/jefflinse/handyman/provider/noop"
	"github.com/urfave/cli/v2"
)

// A Command specifes an action or a set of subcommands.
type Command struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Provider    provider.Provider `json:"-"`
}

var requiredCommandFields = []string{
	"name",
	"description",
}

var commandMap = map[string]func(interface{}) (provider.Provider, error){
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
		Action: c.Provider.CLIActionFn(),
		Flags:  c.Provider.CLIFlags(),
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

	// the provider type is the remaining non-required field name
	if len(content) == len(requiredCommandFields) {
		// don't bother looking for a provider if not enough fields are provided
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

			// use the first available provider we match
			if !isRequiredField {
				if providerCtor, ok := commandMap[key]; ok {
					provider, err := providerCtor(content[key])
					if err != nil {
						return err
					}

					c.Provider = provider
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
	} else if c.Provider == nil {
		return NewInvalidCommandSpecError("missing provider")
	}

	return c.Provider.Validate()
}

// NewInvalidCommandSpecError creates a new error indicating that a command spec is invalid.
func NewInvalidCommandSpecError(reason string) error {
	return fmt.Errorf("invalid command spec: %s", reason)
}
