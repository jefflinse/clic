package main

import (
	"fmt"

	"github.com/jefflinse/clic/spec"
)

// Version is stamped by the build process.
var Version string

func main() {
	content := []byte(
		`name: sample
commands:
  - name: mycommand
    exec:
      path: echo
      args: ["-e", "hello world"]
`)

	app, err := spec.NewApp(content)
	if err != nil {
		panic(err)
	}

	if err := app.Validate(); err != nil {
		panic(err)
	}

	fmt.Print(app.TraceString())

	return
}
