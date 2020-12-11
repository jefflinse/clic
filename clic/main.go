package main

import (
	"github.com/jefflinse/clic/spec"
	"github.com/jefflinse/clic/writer"
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
  - name: chastize
    description: insults someone
    exec:
      path: echo
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

	prod := writer.NewGo(app)
	if err := prod.Generate(); err != nil {
		panic(err)
	}

	return
}
