package writer

import (
	"github.com/jefflinse/clic/spec"
)

// A Writer takes an app spec and produces source code for a CLI app.
type Writer interface {
	WriteFiles(path string) (Output, error)
}

// Output contains information about the generated source files.
type Output struct {
	Dir       string
	FileNames []string
	Spec      spec.App
}
