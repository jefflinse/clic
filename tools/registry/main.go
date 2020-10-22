// The Runner takes a Handyman spec and runs it as a CLI application, passing all remaining arguments to the app.
package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/jefflinse/handyman/registry"
	"github.com/jefflinse/handyman/spec"
)

func main() {
	reg, err := registry.Load()
	fatalOn(err, "error loading registry")

	args := os.Args[1:]
	if len(args) == 0 {
		// invoked without any args, show list of registered apps
		listEntries(reg)
	} else {
		command := strings.ToLower(args[0])
		if command == "list" {
			listEntries(reg)
		} else {
			if len(args) == 1 {
				fatalOn(fmt.Errorf("too few arguments"), "error")
			}

			if command == "add" {
				path := args[1]
				absPath, err := filepath.Abs(path)
				fatalOn(err, "bad file path")

				content, err := ioutil.ReadFile(absPath)
				fatalOn(err, "failed to read spec file")

				appSpec, err := spec.NewAppSpec(content)
				fatalOn(err, "failed to parse spec file")

				fatalOn(appSpec.Validate(), "spec validation failed")
				fatalOn(reg.Add(appSpec.Name, absPath), "failed to register app")
			} else if command == "rm" {
				appName := args[1]
				fatalOn(reg.Remove(appName), "failed to unregister app")
			} else {
				fatalOn(fmt.Errorf("unknown command '%s'", command), "error")
			}
		}
	}
}

func listEntries(reg registry.Registry) {
	longestNameLen := 0
	for name := range reg {
		if len(name) > longestNameLen {
			longestNameLen = len(name)
		}
	}

	for name, path := range reg {
		paddingLen := longestNameLen - len(name)
		fmt.Printf("%s: %s%s\n", name, strings.Repeat(" ", paddingLen), path)
	}

	fmt.Println()
}

func fatalOn(err error, reason string) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s", reason, err.Error())
		os.Exit(1)
	}
}
