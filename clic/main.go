package main

import (
	"io/ioutil"
	"log"

	"github.com/jefflinse/clic/builder"
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

	log.Println("validating app spec")
	if err := app.Validate(); err != nil {
		panic(err)
	}

	w := writer.NewGo(app)
	srcDir, _ := ioutil.TempDir("", "")
	written, err := w.WriteFiles(srcDir)
	if err != nil {
		panic(err)
	} else {
		log.Println("source files written to", written.Dir)
	}

	b := builder.NewGo(app, written)
	bin, _ := ioutil.TempFile("", app.Name)
	built, err := b.Build(bin)
	if err != nil {
		panic(err)
	} else {
		log.Println(built.Type, "app built as", built.Path)
	}

	return
}
