// The Runner takes a Handyman spec and runs it as a CLI application, passing all remaining arguments to the app.
package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/jefflinse/handyman"
	"github.com/jefflinse/handyman/registry"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: %s spec-file [args]", os.Args[0])
		os.Exit(1)
	}

	specFile := os.Args[1]
	var content []byte
	if !fileExists(specFile) {
		// spec file not found, check the registry
		appName := os.Args[1]
		reg, err := registry.Load()
		fatalOn(err, "failed to load registry")

		if path, ok := reg[appName]; ok {
			specFile = path
		} else {
			fatalOn(fmt.Errorf("'%s' is not a valid spec file path or registered app name", appName), "error")
		}
	}

	content, err := ioutil.ReadFile(specFile)
	fatalOn(err, "failed to read spec file")

	app, err := handyman.NewApp(content)
	fatalOn(err, "failed to create app")

	// join the runner binary and spec filename into
	// a single string to be used as arg[0] of the app
	args := []string{}
	if len(os.Args) > 2 {
		args = append(args, os.Args[2:]...)
	}

	if err := app.Run(args); err != nil {
		fmt.Fprintf(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func fatalOn(err error, reason string) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s", reason, err.Error())
		os.Exit(1)
	}
}

func fileExists(name string) bool {
	info, err := os.Stat(name)
	if os.IsNotExist(err) {
		return false
	}

	return !info.IsDir()
}
