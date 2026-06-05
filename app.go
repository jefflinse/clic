package clic

import (
	"github.com/jefflinse/clic/spec"
	"github.com/spf13/cobra"
)

// App is a configured app.
type App struct {
	rootCmd *cobra.Command
	spec    *spec.App
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

// Run runs the clic app with the provided arguments.
func (app App) Run(args []string) error {
	app.rootCmd.SetArgs(args)
	return app.rootCmd.Execute()
}

// newAppFromSpec creates a new clic app from the provided specification.
func newAppFromSpec(appSpec *spec.App) (*App, error) {
	rootCmd := &cobra.Command{
		Use:           appSpec.Name,
		Short:         appSpec.Description,
		SilenceUsage:  true,
		SilenceErrors: false,
	}

	for _, commandSpec := range appSpec.Commands {
		rootCmd.AddCommand(commandSpec.CLICommand())
	}

	return &App{rootCmd: rootCmd, spec: appSpec}, nil
}
