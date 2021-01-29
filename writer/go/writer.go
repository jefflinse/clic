package gowriter

import (
	"path"
	"strings"
	"text/template"

	"github.com/jefflinse/clic/spec"
	"github.com/jefflinse/clic/writer"
	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
)

// Go is the Golang writer.
type Go struct {
	spec spec.App
	fs   afero.Fs
}

type app struct {
	Name     string
	Commands []command
}

type command struct {
	Name  string
	Args  []arg
	Flags []flag
	Exec  spec.Exec
	REST  spec.REST
}

func (c command) NArgs() int {
	return len(c.Args)
}

type arg struct {
	Name        string
	Description string
}

type flag struct {
	Name        string
	Description string
	Type        string
	Default     string
}

// New creates a new Go writer.
func New(app spec.App, fs afero.Fs) *Go {
	return &Go{spec: app, fs: fs}
}

// Content returns a model of the app to be used in the Go templates.
func (g Go) content() app {
	content := app{
		Name: g.spec.Name,
	}

	for _, cmd := range g.spec.Commands {
		c := command{
			Name:  cmd.Name,
			Args:  []arg{},
			Flags: []flag{},
			Exec:  cmd.Exec,
			REST:  cmd.REST,
		}

		for _, param := range cmd.Provider().GetParameters() {
			if param.As == spec.ArgParameter {
				a := arg{
					Name:        asArgName(param.Name),
					Description: param.Description,
				}

				c.Args = append(c.Args, a)
			} else {
				f := flag{
					Name:        asFlagName(param.Name),
					Description: param.Description,
					Type:        strings.Title(param.Type),
					Default:     param.Default,
				}

				c.Flags = append(c.Flags, f)
			}
		}

		content.Commands = append(content.Commands, c)
	}

	return content
}

// WriteFiles creates all source code files for the app.
func (g Go) WriteFiles(targetDir string) (*writer.Output, error) {
	t, err := template.New("app").Parse(appTemplate)
	if err != nil {
		return nil, err
	}

	log.Info().Msg("writing Go source files")
	log.Debug().Str("path", targetDir).Msg("source files")

	sourceFile := "app.go"
	sourceFilePath := path.Join(targetDir, sourceFile)
	f, err := g.fs.Create(sourceFilePath)
	if err != nil {
		return nil, err
	}

	if err := t.Execute(f, g.content()); err != nil {
		return nil, err
	}

	return &writer.Output{
		Spec:      g.spec,
		Dir:       targetDir,
		FileNames: []string{sourceFile},
	}, nil
}

func asArgName(str string) string {
	return strings.ReplaceAll(strings.ToLower(str), "_", "-")
}

func asFlagName(str string) string {
	return strings.ReplaceAll(strings.ToLower(str), "_", "-")
}

func asParamName(str string) string {
	return strings.ReplaceAll(strings.ToLower(str), "-", "_")
}
