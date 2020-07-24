// The Runner takes a Handyman spec and runs it as a CLI application, passing all remaining arguments to the app.
package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/jefflinse/handyman/spec"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: %s spec-file", os.Args[0])
		os.Exit(1)
	}

	content, err := ioutil.ReadFile(os.Args[1])
	fatalOn(err, "failed to read spec file")

	appSpec, err := spec.NewAppSpec(content)
	fatalOn(err, "failed to parse spec file")

	fatalOn(appSpec.Validate(), "spec validation failed")
}

func fatalOn(err error, reason string) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s", reason, err.Error())
		os.Exit(1)
	}
}
