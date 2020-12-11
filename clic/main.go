package main

import (
	"github.com/jefflinse/clic/appwriter"
	"github.com/jefflinse/clic/spec"
)

// Version is stamped by the build process.
var Version string

func main() {
	content := []byte(
		`name: sample
commands:
  - name: greet
    description: prints a greeting message
    exec:
      path: echo
      args: ["-e", "hello {{params.name}}!"]
      params:
        - name: name
          type: string
          required: true
  - name: greet
    description: prints a greeting message
    exec:
      path: chastize
      args: ["-e", "{{params.explative}}, {{params.name}}!"]
      params:
        - name: name
          type: string
          required: true
        - name: explative
          default: fuck you
`)

	app, err := spec.NewApp(content)
	if err != nil {
		panic(err)
	}

	if err := app.Validate(); err != nil {
		panic(err)
	}

	prod := appwriter.NewGo(app)
	if err := prod.Generate(); err != nil {
		panic(err)
	}

	return
}
