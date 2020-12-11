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
	log.Println("creating Go writer")
	return &Go{spec: app}
}

// WriteFiles creates all source code files for the app.
func (g Go) WriteFiles(targetDir string) (*Output, error) {
	exe, err := os.Executable()
	if err != nil {
		return nil, err
	}

	appTemplateFile := path.Join(filepath.Dir(exe), "..", "writer", "templates", "go.t")
	log.Println("loading Go app template from", appTemplateFile)
	appTemplate, err := template.New(path.Base(appTemplateFile)).ParseFiles(appTemplateFile)
	if err != nil {
		return nil, err
	}

	log.Println("generating Go source files")
	sourceFile := "app.go"
	sourceFilePath := path.Join(targetDir, sourceFile)
	log.Println("+", sourceFile)
	f, err := os.Create(sourceFilePath)
	if err != nil {
		return nil, err
	}

	log.Println("executing template replacements")
	if err := appTemplate.Execute(f, &g.spec); err != nil {
		return nil, err
	}

	return &Output{Dir: targetDir, FileNames: []string{sourceFile}}, nil
}

// Build produces a runnable app from the source code files.
func (g Go) Build() error {
	return nil
}
