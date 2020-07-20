package spec

import (
	"encoding/json"
	"fmt"
)

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
		return NewInvalidCommandSpecError("missing name")
	} else if command.Description == "" {
		return NewInvalidCommandSpecError("missing description")
	} else if command.Type == "" {
		return NewInvalidCommandSpecError("missing type")
	} else {
		switch command.Type {
		case ExecCommandType:
			if command.Exec == "" {
				return NewInvalidCommandSpecError("missing exec")
			}
		case LambdaCommandType:
			if command.LambdaARN == "" {
				return NewInvalidCommandSpecError("missing lambda ARN")
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
				return NewInvalidCommandSpecError("missing subcommands")
			}

			for _, command := range command.Subcommands {
				if err := command.Validate(); err != nil {
					return err
				}
			}
		default:
			return NewInvalidCommandSpecError(fmt.Sprintf("unknown command type: %s", command.Type))
		}
	}

	return nil
}

// NewInvalidCommandSpecError creates a new error indicating that a command spec is invalid.
func NewInvalidCommandSpecError(reason string) error {
	return fmt.Errorf("invalid command spec: %s", reason)
}
