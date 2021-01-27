package io

import (
	"fmt"
	"os"
)

const (
	// Nonexistent indicates that the path does not exist.
	Nonexistent = iota

	// File indicates that the path is a file.
	File

	// Directory indicates that the path is a diectory.
	Directory
)

// PathType returns whether the specified path is a file, a direcotry, or does not exist.
func PathType(path string) (int, error) {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return Nonexistent, nil
	} else if err == nil {
		if info.IsDir() {
			return Directory, nil
		}

		return File, nil
	}

	return -1, err
}

// CreateDirectory creates a directory, removing any existing directory if 'force' is true.
func CreateDirectory(path string, force bool) error {
	if t, err := PathType(path); err == nil && t == Directory {
		if force {
			if rmErr := os.RemoveAll(path); rmErr != nil {
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

	return os.MkdirAll(path, 0755)
}
