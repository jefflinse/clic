package builder

import "github.com/jefflinse/clic/spec"

// A Builder takes source files and produces an app.
// The app is a single file that can be executed from the command line.
type Builder interface {
	Build() (string, error)
}

// Output contains information about the built app.
type Output struct {
	Path string
	Type string
	Spec spec.App
}
