package clic

import (
	"github.com/jefflinse/clic/spec"
	"github.com/urfave/cli/v2"
)

// App is a configured app.
type App struct {
	cliApp *cli.App
	spec   *spec.App
}

// NewApp creates a new clic app from the provided JSON specification.
func NewApp(specification []byte) (*App, error) {
	appSpec, err := spec.NewAppSpec(specification)
	if err != nil {
		return nil, err
	}

	if err := appSpec.Validate(); err != nil {
		return nil, err
	}

	return newAppFromSpec(appSpec)
}

// Run runs the clic app.
func (app App) Run(args []string) error {
	arguments := append([]string{app.spec.Name}, args...)
	return app.cliApp.Run(arguments)
}

// NewApp creates a new clic app from the provided specification
func newAppFromSpec(appSpec *spec.App) (*App, error) {
	cliApp := &cli.App{
		Name:                  appSpec.Name,
		HelpName:              appSpec.Name,
		Usage:                 appSpec.Description,
		Commands:              make([]*cli.Command, 0),
		HideHelp:              true,
		CustomAppHelpTemplate: AppHelpTemplate(),
	}

	for _, commandSpec := range appSpec.Commands {
		cliCmd := commandSpec.CLICommand()
		cliCmd.CustomHelpTemplate = CommandHelpTemplate()
		cliApp.Commands = append(cliApp.Commands, cliCmd)
	}

	return &App{cliApp: cliApp, spec: appSpec}, nil
}

// AppHelpTemplate defines the layout of app help.
func AppHelpTemplate() string {
	return `{{.Name}}{{if .Usage}} - {{.Usage}}{{end}}

usage:
	{{if .UsageText}}{{.UsageText}}{{else}}{{.HelpName}} {{if .VisibleFlags}}[global options]{{end}}{{if .Commands}} command [command options]{{end}} {{if .ArgsUsage}}{{.ArgsUsage}}{{else}}[arguments...]{{end}}{{end}}{{if .Version}}{{if not .HideVersion}}

version {{.Version}}{{end}}{{end}}{{if .VisibleCommands}}
 
commands:{{range .VisibleCategories}}{{if .Name}}
	{{.Name}}:{{range .VisibleCommands}}
	  {{join .Names ", "}}{{"\t"}}{{.Usage}}{{end}}{{else}}{{range .VisibleCommands}}
	{{join .Names ", "}}{{"\t"}}{{.Usage}}{{end}}{{end}}{{end}}{{end}}{{if .VisibleFlags}}

global options:
	{{range $index, $option := .VisibleFlags}}{{if $index}}
	{{end}}{{$option}}{{end}}{{end}}
`
}

// CommandHelpTemplate defines the layout of command help.
func CommandHelpTemplate() string {
	return `{{.HelpName}} - {{.Usage}}

usage:
	{{if .UsageText}}{{.UsageText}}{{else}}{{.HelpName}}{{if .VisibleFlags}} [options]{{end}} {{if .ArgsUsage}}{{.ArgsUsage}}{{else}}[arguments...]{{end}}{{end}}{{if .Category}}

description:
	{{.Description}}{{end}}{{if .VisibleFlags}}

options:
	{{range .VisibleFlags}}{{.}}
	{{end}}{{end}}
`
}
