package spec

import (
	"encoding/json"
)

// An App specifies a complete Handyman application.
type App struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Commands    []*Command `json:"commands"`
}

// A Command specifes an action or a set of subcommands.
type Command struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Subcommands []*Command `json:"subcommands,omitempty"`
}

// NewAppSpec creates a new App from the provided spec.
func NewAppSpec(content []byte) (*App, error) {
	app := &App{}
	if err := json.Unmarshal(content, app); err != nil {
		return nil, err
	}

	return app, nil
}

// Validate validates an App spec.
func (app App) Validate() error {
	if app.Name == "" {
		return NewInvalidSpecError("missing app name")
	} else if app.Description == "" {
		return NewInvalidSpecError("missing app description")
	}

	for _, command := range app.Commands {
		if err := command.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// NewCommandSpec creates a new Command from the provided spec.
func NewCommandSpec(content []byte) (*Command, error) {
	command := &Command{}
	if err := json.Unmarshal(content, command); err != nil {
		return nil, err
	}

	return command, nil
}

// Validate validates a Command spec.
func (command Command) Validate() error {
	if command.Name == "" {
		return NewInvalidSpecError("missing command name")
	} else if command.Description == "" {
		return NewInvalidSpecError("missing command description")
	}

	for _, command := range command.Subcommands {
		if err := command.Validate(); err != nil {
			return err
		}
	}

	return nil
}
