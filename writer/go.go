package writer

import (
	"log"
	"os"
	"path"
	"path/filepath"
	"text/template"

	"github.com/jefflinse/clic/spec"
)

// Go is the Golang producer.
type Go struct {
	spec *spec.App
}

// NewGo creates a new Go producer.
func NewGo(app *spec.App) *Go {
	return &Go{spec: app}
}

// Generate creates all source code files for the app.
func (g Go) Generate() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}

	appTemplateFile := path.Join(filepath.Dir(exe), "..", "appwriter", "templates", "go.t")
	log.Println("loading app template from", appTemplateFile)
	appTemplate, err := template.New(path.Base(appTemplateFile)).ParseFiles(appTemplateFile)
	if err != nil {
		return err
	}

	if err := appTemplate.Execute(os.Stdout, &g.spec); err != nil {
		return err
	}

	return nil
}

// Build produces a runnable app from the source code files.
func (g Go) Build() error {
	return nil
}
