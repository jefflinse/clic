package spec

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/jefflinse/clic/io"
	"github.com/rs/zerolog/log"
)

var specFileUnmarshalers map[string]func(data []byte, v interface{}) error = map[string]func(data []byte, v interface{}) error{
	"json": json.Unmarshal,
	"yaml": yaml.Unmarshal,
	"yml":  yaml.Unmarshal,
}

// An App represents a clic app.
type App struct {
	Name     string    `json:"name"`
	Commands []Command `json:"commands"`
}

// NewAppFromPath create a new app spec from the specified file or directory.
func NewAppFromPath(path string) (App, error) {
	if t, err := io.PathType(path); err != nil {
		return App{}, err
	} else if t == io.Nonexistent {
		return App{}, fmt.Errorf("path '%s' does not exist", path)
	} else if t == io.File {
		log.Debug().Str("type", "file").Str("path", path).Msg("app spec")
		return newAppFromFile(path)
	} else if t == io.Directory {
		log.Debug().Str("type", "directory").Str("path", path).Msg("app spec")
		return newAppFromDirectory(path)
	}

	return App{}, fmt.Errorf("unexepcted error regarding path '%s'", path)
}

func newAppFromDirectory(path string) (App, error) {
	var specFiles []string
	for extension := range specFileUnmarshalers {
		files, _ := filepath.Glob(filepath.Join(path, "*."+extension))
		specFiles = append(specFiles, files...)
	}

	var specs []App
	for _, file := range specFiles {
		spec, err := newAppFromFile(file)
		if err != nil {
			return App{}, err
		}
		specs = append(specs, spec)
	}

	return mergeAppSpecs(specs...)
}

func newAppFromFile(path string) (App, error) {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return App{}, err
	}

	extension := strings.Split(filepath.Base(path), ".")[1]
	if unmarshaler, ok := specFileUnmarshalers[extension]; ok {
		var app App
		if err := unmarshaler(content, &app); err != nil {
			return app, err
		}

		return app, nil
	}

	return App{}, fmt.Errorf("unsupported file extension '%s'", extension)
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

// mergeAppSpecs merges multiple app specs into a single one.
func mergeAppSpecs(specs ...App) (App, error) {
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
