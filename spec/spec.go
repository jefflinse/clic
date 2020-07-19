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
	Name                    string       `json:"name"`
	Description             string       `json:"description"`
	Type                    string       `json:"type"`
	Exec                    string       `json:"exec,omitempty"`
	LambdaARN               string       `json:"lambda_arn,omitempty"`
	LambdaRequestParameters []*Parameter `json:"lambda_request_parameters,omitempty"`
	Subcommands             []*Command   `json:"subcommands,omitempty"`
}

// A Parameter specifies a command parameter.
type Parameter struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"`
	Required    bool   `json:"required,omitempty"`
}

const (
	// ExecCommandType is a command that executes a local command.
	ExecCommandType = "EXEC"

	// LambdaCommandType is a command that invokes an AWS Lambda function.
	LambdaCommandType = "LAMBDA"

	// NoopCommandType is a command that does nothing.
	NoopCommandType = "NOOP"

	// SubcommandsCommandType is a command that contains one or more subcommands.
	SubcommandsCommandType = "SUBCOMMANDS"
)

const (
	// StringParamType is a string parameter.
	StringParamType = "string"
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
				return NewInvalidSpecError("missing command exec")
			}
		case LambdaCommandType:
			if command.LambdaARN == "" {
				return NewInvalidSpecError("missing command lambda ARN")
			}

			if command.LambdaRequestParameters != nil {
				for _, param := range command.LambdaRequestParameters {
					if err := param.Validate(); err != nil {
						return err
					}
				}
			}
		case NoopCommandType:
		case SubcommandsCommandType:
			if command.Subcommands == nil || len(command.Subcommands) == 0 {
				return NewInvalidSpecError("missing command subcommands")
			}

			for _, command := range command.Subcommands {
				if err := command.Validate(); err != nil {
					return err
				}
			}
		default:
			return NewInvalidSpecError(fmt.Sprintf("unknown command type: %s", command.Type))
		}
	}

	return nil
}

// NewParamterSpec creates a new Parameter from the provided spec.
func NewParamterSpec(content []byte) (*Parameter, error) {
	param := &Parameter{}
	if err := json.Unmarshal(content, param); err != nil {
		return nil, err
	}

	return param, nil
}

// Validate validates a Parameter.
func (param Parameter) Validate() error {
	if param.Name == "" {
		return NewInvalidSpecError("missing parameter name")
	} else if param.Description == "" {
		return NewInvalidSpecError("missing parameter description")
	} else if param.Type == "" {
		return NewInvalidSpecError("missing parameter type")
	} else {
		switch param.Type {
		case StringParamType:
		default:
			return NewInvalidSpecError(fmt.Sprintf("unknown parameter type: %s", param.Type))
		}
	}

	return nil
}
