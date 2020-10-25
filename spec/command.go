package spec

import (
	"encoding/json"
	"fmt"

	"github.com/goccy/go-yaml"
	"github.com/jefflinse/handyman/ioutil"
	"github.com/jefflinse/handyman/provider"
	"github.com/jefflinse/handyman/provider/exec"
	"github.com/jefflinse/handyman/provider/lambda"
	"github.com/jefflinse/handyman/provider/noop"
	"github.com/jefflinse/handyman/provider/rest"
	"github.com/urfave/cli/v2"
)

// A Command specifes an action or a set of subcommands.
type Command struct {
	Name        string            `json:"name"                  yaml:"name"`
	Description string            `json:"description"           yaml:"description"`
	Provider    provider.Provider `json:"-"                     yaml:"-"`
	Subcommands []*Command        `json:"subcommands,omitempty" yaml:"subcommands,omitempty"`
}

var requiredCommandFields = []string{
	"name",
	"description",
}

var commandMap = map[string]func(interface{}) (provider.Provider, error){
	"exec":   exec.New,
	"lambda": lambda.New,
	"noop":   noop.New,
	"rest":   rest.New,
}

// NewCommandSpec creates a new Command from the provided spec.
func NewCommandSpec(content []byte) (*Command, error) {
	if len(content) == 0 {
		return nil, NewInvalidCommandSpecError("spec is empty")
	}

	command := &Command{}
	if content[0] == '{' {
		// assume JSON
		if err := json.Unmarshal(content, command); err != nil {
			return nil, err
		}
	} else {
		if err := yaml.Unmarshal(content, command); err != nil {
			return nil, err
		}
	}

	return command, nil
}

// CLICommand creates a CLI command for this command.
func (c *Command) CLICommand() *cli.Command {
	cliCmd := &cli.Command{
		Name:  c.Name,
		Usage: c.Description,
	}

	if len(c.Subcommands) > 0 {
		cliCmd.Subcommands = []*cli.Command{}
		for _, subcommand := range c.Subcommands {
			cliCmd.Subcommands = append(cliCmd.Subcommands, subcommand.CLICommand())
		}
	} else {
		cliCmd.Action = c.Provider.CLIActionFn()
		cliCmd.ArgsUsage = c.Provider.ArgsUsage()
		cliCmd.Flags = c.Provider.CLIFlags()
	}

	return cliCmd
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

	// either we have subcommands, or the provider type
	// is the remaining non-required field name
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

			if !isRequiredField {
				if key == "subcommands" {
					subcommands, ok := content[key].([]interface{})
					if !ok {
						return fmt.Errorf("cannot coerce subcommands to []interface{}")
					}

					for _, data := range subcommands {
						subcommand := &Command{}
						if err := ioutil.Intermarshal(data, subcommand); err != nil {
							return fmt.Errorf("failed to parse subcommands: %w", err)
						}

						c.Subcommands = append(c.Subcommands, subcommand)
					}

				} else if providerCtor, ok := commandMap[key]; ok {
					provider, err := providerCtor(content[key])
					if err != nil {
						return err
					}

					c.Provider = provider
				}
			}
		}
	}

	return nil
}

// UnmarshalYAML unmarshals the specified YAML data into the command.
func (c *Command) UnmarshalYAML(data []byte) error {
	type commandMetadata struct {
		Name        string `yaml:"name"`
		Description string `yaml:"description"`
	}

	metadata := commandMetadata{}
	if err := yaml.Unmarshal(data, &metadata); err != nil {
		return err
	}

	c.Name = metadata.Name
	c.Description = metadata.Description

	content := map[string]interface{}{}
	if err := yaml.Unmarshal(data, &content); err != nil {
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

			if !isRequiredField {
				if key == "subcommands" {
					subcommands, ok := content[key].([]interface{})
					if !ok {
						return fmt.Errorf("cannot coerce subcommands to []interface{}")
					}

					for _, data := range subcommands {
						subcommand := &Command{}
						if err := ioutil.Intermarshal(data, subcommand); err != nil {
							return fmt.Errorf("failed to parse subcommands: %w", err)
						}

						c.Subcommands = append(c.Subcommands, subcommand)
					}

				} else if providerCtor, ok := commandMap[key]; ok {
					provider, err := providerCtor(content[key])
					if err != nil {
						return err
					}

					c.Provider = provider
				}
			}
		}
	}

	return nil
}

// Validate validates a Command spec.
func (c *Command) Validate() error {
	if c.Name == "" {
		return NewInvalidCommandSpecError("missing name")
	} else if c.Description == "" {
		return NewInvalidCommandSpecError("missing description")
	} else if c.Provider == nil && len(c.Subcommands) == 0 {
		return NewInvalidCommandSpecError("missing provider or subcommands")
	} else if c.Provider != nil && len(c.Subcommands) > 0 {
		return NewInvalidCommandSpecError("cannot specify both provider and subcommands")
	}

	if c.Provider != nil {
		return c.Provider.Validate()
	}

	for _, subcommand := range c.Subcommands {
		if err := subcommand.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// NewInvalidCommandSpecError creates a new error indicating that a command spec is invalid.
func NewInvalidCommandSpecError(reason string) error {
	return fmt.Errorf("invalid command spec: %s", reason)
}
