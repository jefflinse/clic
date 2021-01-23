package spec

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/jefflinse/clic/io"
)

// An App represents a clic app.
type App struct {
	Name     string    `json:"name"`
	Commands []Command `json:"commands"`
}

// NewAppFromFile creates a new clic app from the specified spec file.
func NewAppFromFile(path string) (App, error) {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return App{}, err
	}

	return NewApp(content)
}

// NewApp creates a new clic app from the provided spec content.
func NewApp(data []byte) (App, error) {
	app := App{}
	return app, io.Unmarshal(data, &app)
}

// TraceString prints the app hierarchy.
func (a App) TraceString() string {
	b := strings.Builder{}

	b.WriteString(a.Name + "\n")
	for _, command := range a.Commands {
		b.WriteString("  ")
		b.WriteString(command.TraceString())
	}

	return b.String()
}

// Validate returns an error if the app spec is invalid.
func (a App) Validate() (App, error) {
	if a.Name == "" {
		return a, fmt.Errorf("invalid app spec: missing name")
	}

	vcs := []Command{}
	for _, c := range a.Commands {
		vc, err := c.Validate()
		if err != nil {
			return a, err
		}

		vcs = append(vcs, vc)
	}

	return App{
		Name:     a.Name,
		Commands: vcs,
	}, nil
}
