package io

import "os"

// FileExists returns true if a file exists and is not a directory.
func FileExists(name string) bool {
	info, err := os.Stat(name)
	if os.IsNotExist(err) {
		return false
	}

	return !info.IsDir()
}
