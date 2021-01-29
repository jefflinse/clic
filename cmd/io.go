package cmd

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/jefflinse/clic/spec"
	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
)

var execRunner func(cmd *exec.Cmd) error
var fs afero.Fs

// Creates a directory, removing any existing directory if 'force' is true.
func createDirectory(path string, force bool) error {
	exists, err := afero.Exists(fs, path)
	if err != nil {
		return err
	}

	if exists {
		isDir, err := afero.IsDir(fs, path)
		if err != nil {
			return err
		}

		if !isDir {
			return fmt.Errorf("cannot create directory '%s': directory exists", path)
		}

		if force {
			if rmErr := fs.RemoveAll(path); rmErr != nil {
				return err
			}
		}

	}

	return fs.MkdirAll(path, 0755)
}

// Reads all eligible app spec files in the specified directory and returns the merged app.
func newAppFromDirectory(path string) (spec.App, error) {
	var specFiles []string
	for extension := range specFileUnmarshalers {
		files, _ := filepath.Glob(filepath.Join(path, "*."+extension))
		specFiles = append(specFiles, files...)
	}

	var specs []spec.App
	for _, file := range specFiles {
		appSpec, err := newAppFromFile(file)
		if err != nil {
			return spec.App{}, err
		}
		specs = append(specs, appSpec)
	}

	log.Debug().Str("type", "directory").Str("path", path).Msg("read app spec")
	return spec.MergeAppSpecs(specs...)
}

// Reads the specified app spec file and returns an app.
func newAppFromFile(path string) (spec.App, error) {
	content, err := afero.ReadFile(fs, path)
	if err != nil {
		return spec.App{}, err
	}

	extension := strings.Split(filepath.Base(path), ".")[1]
	if unmarshaler, ok := specFileUnmarshalers[extension]; ok {
		var app spec.App
		if err := unmarshaler(content, &app); err != nil {
			return app, err
		}

		log.Debug().Str("type", "file").Str("path", path).Msg("read app spec")
		return app, nil
	}

	return spec.App{}, fmt.Errorf("unsupported file extension '%s'", extension)
}

// Reads an app spec from the specified file or directory.
func newAppFromPath(path string) (spec.App, error) {
	isDir, err := afero.IsDir(fs, path)
	if err != nil {
		return spec.App{}, err
	}

	if isDir {
		return newAppFromDirectory(path)
	}

	exists, err := afero.Exists(fs, path)
	if err != nil {
		return spec.App{}, err
	}

	if exists {
		return newAppFromFile(path)
	}

	return spec.App{}, fmt.Errorf("'%s' does not exist", path)
}
