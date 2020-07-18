// The Runner takes a Handyman spec and runs it as a CLI application, passing all remaining arguments to the app.
package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/jefflinse/handyman"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: %s spec-file [args]", os.Args[0])
	}

	content, err := ioutil.ReadFile(os.Args[1])
	fmt.Println(err)
	fatalOn(err, "failed to read spec file")

	app, err := handyman.NewApp(content)
	fatalOn(err, "failed to create app")

	args := []string{}
	if len(os.Args) > 2 {
		args = append(args, os.Args[2:]...)
	}

	app.Run(args)
}

func fatalOn(err error, reason string) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s", reason, err.Error())
		os.Exit(1)
	}
}
