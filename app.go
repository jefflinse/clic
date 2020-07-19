package handyman

import (
	"github.com/jefflinse/handyman/spec"
	"github.com/urfave/cli/v2"
)

// App is a configured app.
type App struct {
	cliApp *cli.App
	spec   *spec.App
}

// NewApp creates a new Handyman app from the provided JSON specification.
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

// Run runs the Handyman app.
func (app App) Run(args []string) error {
	arguments := append([]string{app.spec.Name}, args...)
	return app.cliApp.Run(arguments)
}

// NewApp creates a new Handyman app from the provided specification
func newAppFromSpec(appSpec *spec.App) (*App, error) {
	cliApp := &cli.App{
		Name:     appSpec.Name,
		HelpName: appSpec.Name,
		Usage:    appSpec.Description,
		Commands: make([]*cli.Command, 0),
		HideHelp: true,
	}

	for _, commandSpec := range appSpec.Commands {
		cliApp.Commands = append(cliApp.Commands, newCommandFromSpec(commandSpec))
	}

	return &App{cliApp: cliApp, spec: appSpec}, nil
}
