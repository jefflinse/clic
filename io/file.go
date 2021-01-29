package io

import (
	"fmt"

	"github.com/spf13/afero"
)

var fs afero.Fs

const (
	// Nonexistent indicates that the path does not exist.
	Nonexistent = iota

	// File indicates that the path is a file.
	File

	// Directory indicates that the path is a diectory.
	Directory
)

// Init initializes a file system for IO operations.
func Init(dryRun bool) afero.Fs {
	if dryRun {
		fs = afero.NewCopyOnWriteFs(afero.NewOsFs(), afero.NewMemMapFs())
	} else {
		fs = afero.NewOsFs()
	}

	return fs
}

// PathType returns whether the specified path is a file, a directory, or does not exist.
func PathType(path string) (int, error) {
	exists, err := afero.Exists(fs, path)
	if err != nil {
		return -1, err
	} else if !exists {
		return Nonexistent, nil
	}

	isDir, err := afero.DirExists(fs, path)
	if err != nil {
		return -1, err
	} else if isDir {
		return Directory, nil
	}

	return File, nil
}

// CreateDirectory creates a directory, removing any existing directory if 'force' is true.
func CreateDirectory(path string, force bool) error {
	if t, err := PathType(path); err == nil && t == Directory {
		if force {
			if rmErr := fs.RemoveAll(path); rmErr != nil {
				return err
			}
		} else {
			return fmt.Errorf("cannot create directory '%s': directory exists", path)
		}
	} else if err == nil && t == File {
		return fmt.Errorf("cannot create directory '%s'; file exists", path)
	} else if err != nil {
		return err
	}

	return fs.MkdirAll(path, 0755)
}
