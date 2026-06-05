package spec

import (
	"encoding/json"
	"fmt"

	"github.com/goccy/go-yaml"
	"github.com/jefflinse/clic/ioutil"
	"github.com/jefflinse/clic/provider"
	"github.com/jefflinse/clic/provider/exec"
	"github.com/jefflinse/clic/provider/lambda"
	"github.com/jefflinse/clic/provider/noop"
	"github.com/jefflinse/clic/provider/rest"
	"github.com/spf13/cobra"
)

// A Command specifes an action or a set of subcommands.
type Command struct {
	Name        string            `json:"name"                  yaml:"name"`
	Description string            `json:"description"           yaml:"description"`
	Provider    provider.Provider `json:"-"                     yaml:"-"`
	Subcommands []*Command        `json:"subcommands,omitempty" yaml:"subcommands,omitempty"`
}

type contentUnmarshaler func(data []byte, target any) error

var requiredCommandFields = []string{
	"name",
	"description",
}

var commandMap = map[string]func(any) (provider.Provider, error){
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

// CLICommand creates a cobra command for this command.
func (c *Command) CLICommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   c.Name,
		Short: c.Description,
	}

	if len(c.Subcommands) > 0 {
		for _, subcommand := range c.Subcommands {
			cmd.AddCommand(subcommand.CLICommand())
		}
	} else if c.Provider != nil {
		c.Provider.Configure(cmd)
	}

	return cmd
}

// UnmarshalJSON unmarshals the specified JSON data into the command.
func (c *Command) UnmarshalJSON(data []byte) error {
	return c.unmarshalContent(json.Unmarshal, data)
}

// UnmarshalYAML unmarshals the specified YAML data into the command.
func (c *Command) UnmarshalYAML(data []byte) error {
	return c.unmarshalContent(yaml.Unmarshal, data)
}

// MarshalJSON marshals the command to JSON, writing the provider's config under
// its type key (e.g. "rest") so the result round-trips back through parsing.
func (c *Command) MarshalJSON() ([]byte, error) {
	out := map[string]any{
		"name":        c.Name,
		"description": c.Description,
	}
	if c.Provider != nil {
		out[c.Provider.Type()] = c.Provider
	}
	if len(c.Subcommands) > 0 {
		out["subcommands"] = c.Subcommands
	}

	return json.Marshal(out)
}

// MarshalYAML marshals the command to YAML, writing the provider's config under
// its type key (e.g. "rest") in a stable, human-friendly field order.
func (c *Command) MarshalYAML() (any, error) {
	out := yaml.MapSlice{
		{Key: "name", Value: c.Name},
		{Key: "description", Value: c.Description},
	}
	if c.Provider != nil {
		out = append(out, yaml.MapItem{Key: c.Provider.Type(), Value: c.Provider})
	}
	if len(c.Subcommands) > 0 {
		out = append(out, yaml.MapItem{Key: "subcommands", Value: c.Subcommands})
	}

	return out, nil
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

func (c *Command) unmarshalContent(unmarshaler contentUnmarshaler, data []byte) error {
	type commandMetadata struct {
		Name        string `json:"name"        yaml:"name"`
		Description string `json:"description" yaml:"description"`
	}

	metadata := commandMetadata{}
	if err := unmarshaler(data, &metadata); err != nil {
		return err
	}

	c.Name = metadata.Name
	c.Description = metadata.Description

	content := map[string]any{}
	if err := unmarshaler(data, &content); err != nil {
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
					subcommands, ok := content[key].([]any)
					if !ok {
						return fmt.Errorf("cannot coerce subcommands to []any")
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
