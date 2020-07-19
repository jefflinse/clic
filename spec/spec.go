package spec

import (
	"encoding/json"
	"fmt"
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
	Type        string     `json:"type"`
	Exec        string     `json:"exec,omitempty"`
	Subcommands []*Command `json:"subcommands,omitempty"`
}

const (
	// ExecCommandType is a command that executes a local command.
	ExecCommandType = "EXEC"

	// SubcommandsCommandType is a command that contains one or more subcommands.
	SubcommandsCommandType = "SUBCOMMANDS"

	// NoopCommandType is a command that does nothing.
	NoopCommandType = "NOOP"
)

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
	} else if command.Type == "" {
		return NewInvalidSpecError("missing command type")
	} else {
		switch command.Type {
		case ExecCommandType:
			if command.Exec == "" {
				return NewInvalidSpecError("missing command spec")
			}
		case NoopCommandType:
		case SubcommandsCommandType:
			if command.Subcommands == nil || len(command.Subcommands) == 0 {
				return NewInvalidSpecError("missing command subcommands")
			}
		default:
			return NewInvalidSpecError(fmt.Sprintf("unknown command type: %s", command.Type))
		}
	}

	for _, command := range command.Subcommands {
		if err := command.Validate(); err != nil {
			return err
		}
	}

	return nil
}
