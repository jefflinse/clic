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

// NewApp creates a new standalone clic app from the provided clic spec (JSON or
// YAML). A standalone app is the top-level CLI for itself, so it owns clic's
// global flags (--server, -i, auth) directly.
func NewApp(specification []byte) (*App, error) {
	appSpec, err := spec.NewAppSpec(specification)
	if err != nil {
		return nil, err
	}

	return newApp(appSpec, true)
}

// NewAppFromSpec creates a clic app from an already-parsed spec, for callers
// (such as the clic launcher) that compile a spec from another format and
// manage clic's global flags themselves, threading them in via RunContext.
func NewAppFromSpec(appSpec *spec.App) (*App, error) {
	return newApp(appSpec, false)
}

// Run runs the clic app with the provided arguments and a background context.
func (app App) Run(args []string) error {
	return app.RunContext(context.Background(), args)
}

// RunContext runs the clic app with the provided arguments and a caller-supplied
// context, which may already carry clic options (see provider.WithOptions). The
// spec's auth scheme, if any, is attached before execution.
func (app App) RunContext(ctx context.Context, args []string) error {
	app.rootCmd.SetArgs(args)

	if app.spec.Auth != nil {
		ctx = provider.WithAuth(ctx, app.spec.Auth)
	}

	return app.rootCmd.ExecuteContext(ctx)
}

// newApp builds the cobra command tree for the given spec. In standalone mode
// it registers clic's global flags and resolves them into the context before
// each command runs; otherwise the launcher supplies those options via context.
func newApp(appSpec *spec.App, standalone bool) (*App, error) {
	if err := appSpec.Validate(); err != nil {
		return nil, err
	}

	rootCmd := &cobra.Command{
		Use:           appSpec.Name,
		Short:         appSpec.Description,
		SilenceUsage:  true,
		SilenceErrors: false,
	}

	if standalone {
		provider.RegisterGlobalFlags(rootCmd.PersistentFlags(), appSpec.Server)
		rootCmd.PersistentPreRunE = func(cmd *cobra.Command, _ []string) error {
			cmd.SetContext(provider.WithOptions(cmd.Context(), provider.ResolveOptions(cmd.Flags())))
			return nil
		}
	}

	for _, commandSpec := range appSpec.Commands {
		rootCmd.AddCommand(commandSpec.CLICommand())
	}

	return &App{rootCmd: rootCmd, spec: appSpec}, nil
}
