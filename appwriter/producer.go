package appwriter

import "io/ioutil"

// A Producer takes an app spec and produces a runnable CLI application.
type Producer interface {
	GenerateFiles() error
	BuildApp() error
}

func createSourceCodeDir() (string, error) {
	return ioutil.TempDir("", "clic.gen.go")
}
