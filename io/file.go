package io

import (
	"fmt"
	"os"
)

// DirectoryExists returns whether the given file or directory exists.
func DirectoryExists(path string) (bool, error) {
	info, err := os.Stat(path)
	if err == nil {
		if info.IsDir() {
			return true, nil
		}

		return false, fmt.Errorf("%s is not a directory", path)
	}

	if os.IsNotExist(err) {
		return false, nil
	}

	return false, err
}

// CreateDirectory creates a directory, removing any existing directory if 'force' is true.
func CreateDirectory(path string, force bool) error {
	if exists, err := DirectoryExists(path); exists {
		if force {
			if rmErr := os.RemoveAll(path); rmErr != nil {
				return err
			}
		}
	} else if err != nil {
		return err
	}

	return os.MkdirAll(path, 0755)
}
