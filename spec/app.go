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
		return NewInvalidAppSpecError("missing name")
	} else if app.Description == "" {
		return NewInvalidAppSpecError("missing description")
	}

	for _, command := range app.Commands {
		if err := command.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// NewInvalidAppSpecError creates a new error indicating that an app spec is invalid.
func NewInvalidAppSpecError(reason string) error {
	return fmt.Errorf("invalid app spec: %s", reason)
}
