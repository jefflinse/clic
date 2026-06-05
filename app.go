package clic

import (
	"context"

	"github.com/jefflinse/clic/provider"
	"github.com/jefflinse/clic/spec"
	"github.com/spf13/cobra"
)

// App is a configured app.
type App struct {
	rootCmd *cobra.Command
	spec    *spec.App
}

// NewApp creates a new clic app from the provided clic spec (JSON or YAML).
func NewApp(specification []byte) (*App, error) {
	appSpec, err := spec.NewAppSpec(specification)
	if err != nil {
		return nil, err
	}

	return NewAppFromSpec(appSpec)
}

// NewAppFromSpec creates a new clic app from an already-parsed spec. This is the
// entry point for callers that compile a spec from another format (e.g. OpenAPI).
func NewAppFromSpec(appSpec *spec.App) (*App, error) {
	if err := appSpec.Validate(); err != nil {
		return nil, err
	}

	return newAppFromSpec(appSpec)
}

// Run runs the clic app with the provided arguments.
func (app App) Run(args []string) error {
	app.rootCmd.SetArgs(args)

	ctx := context.Background()
	if app.spec.Auth != nil {
		ctx = provider.WithAuth(ctx, app.spec.Auth)
	}

	return app.rootCmd.ExecuteContext(ctx)
}

// newAppFromSpec builds the cobra command tree for the given spec.
func newAppFromSpec(appSpec *spec.App) (*App, error) {
	rootCmd := &cobra.Command{
		Use:           appSpec.Name,
		Short:         appSpec.Description,
		SilenceUsage:  true,
		SilenceErrors: false,
	}

	if appSpec.Server != "" {
		provider.RegisterServerFlag(rootCmd, appSpec.Server)
	}
	if appSpec.Auth != nil {
		appSpec.Auth.RegisterFlags(rootCmd)
	}

	for _, commandSpec := range appSpec.Commands {
		rootCmd.AddCommand(commandSpec.CLICommand())
	}

	return &App{rootCmd: rootCmd, spec: appSpec}, nil
}
