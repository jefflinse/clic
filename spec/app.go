package spec

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/jefflinse/clic/io"
)

var specFileExtensions []string = []string{
	"*.json", "*.yaml", "*.yml",
}

// An App represents a clic app.
type App struct {
	Name     string    `json:"name"`
	Commands []Command `json:"commands"`
}

// MergeAppSpecs merges multiple app specs into a single one.
func MergeAppSpecs(specs ...App) (App, error) {
	if len(specs) == 0 {
		panic("MergeAppSpecs() called with too few app specs")
	}

	merged := specs[0]
	var err error
	for i := 1; i < len(specs); i++ {
		merged, err = specs[i].MergeInto(merged)
		if err != nil {
			return App{}, err
		}
	}

	return merged, nil
}

// NewAppFromDirectory creates a new clic app from the specified directory containing spec files.
func NewAppFromDirectory(path string) (App, error) {
	if exists, err := io.DirectoryExists(path); err != nil {
		return App{}, err
	} else if !exists {
		return App{}, fmt.Errorf("path '%s' does not exist", path)
	}

	var specFiles []string
	for _, pattern := range specFileExtensions {
		files, _ := filepath.Glob(filepath.Join(path, pattern))
		specFiles = append(specFiles, files...)
	}

	var specs []App
	for _, file := range specFiles {
		spec, err := NewAppFromFile(file)
		if err != nil {
			return App{}, err
		}
		specs = append(specs, spec)
	}

	return MergeAppSpecs(specs...)
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

// MergeInto merges this app spec into another one, returning the combined spec.
func (a App) MergeInto(other App) (App, error) {
	if a.Name != other.Name {
		return a, fmt.Errorf("failed to merge app specs: names '%s' and '%s' do not match", a.Name, other.Name)
	}

	merged := other
	for _, incoming := range a.Commands {
		for _, current := range merged.Commands {
			if incoming.Name == current.Name {
				return App{}, fmt.Errorf("failed to merge app specs: multiple definitions for '%s' command", current.Name)
			}
			merged.Commands = append(merged.Commands, incoming)
		}
	}

	return merged, nil
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
