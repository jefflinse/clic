package main

import (
	"fmt"
	"os"

	"github.com/jefflinse/clic"
	"github.com/jefflinse/clic/ioutil"
	"github.com/jefflinse/clic/provider"
	"github.com/jefflinse/clic/registry"
	"github.com/jefflinse/clic/source"
	"github.com/jefflinse/clic/spec"
	"github.com/spf13/cobra"
)

// Version is stamped by the build process.
var Version string

func main() {
	if err := rootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

func rootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "clic [spec] [args...]",
		Short:         "the clic CLI",
		Version:       Version,
		Args:          cobra.ArbitraryArgs,
		SilenceUsage:  true,
		SilenceErrors: false,
		// auto-mode: `clic <spec> [args...]` detects the format and runs it
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			return runSpec(cmd, args, spec.FormatUnknown)
		},
	}

	// clic's own global flags are parsed before the spec; everything after the
	// spec passes through to the app as its own argv
	provider.RegisterGlobalFlags(root.PersistentFlags(), "")
	root.Flags().SetInterspersed(false)

	root.AddCommand(
		buildCmd(),
		runCmd(),
		convertCmd(),
		validateCmd(),
		registerCmd(),
		unregisterCmd(),
		listRegistryCmd(),
		pruneRegistryCmd(),
		loginCmd(),
		logoutCmd(),
		testCmd(),
		mockCmd(),
		versionCmd(),
	)

	return root
}

func runCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run <spec> [args...]",
		Short: "run a clic or OpenAPI spec directly",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSpec(cmd, args, forceFormat(cmd))
		},
	}

	addFormatFlags(cmd)
	// stop parsing flags after the spec so the rest pass through to the app
	cmd.Flags().SetInterspersed(false)

	return cmd
}

func convertCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "convert <spec>",
		Short: "compile a spec (e.g. OpenAPI) to a clic spec",
		Args:  cobra.ExactArgs(1),
		RunE:  convert,
	}

	addFormatFlags(cmd)
	cmd.Flags().StringP("output", "o", "", "write the clic spec to a file instead of stdout")

	return cmd
}

func validateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate <spec>",
		Short: "validate a clic or OpenAPI spec",
		Args:  cobra.ExactArgs(1),
		RunE:  validate,
	}

	addFormatFlags(cmd)
	return cmd
}

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "print the current clic CLI version",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("clic CLI version %s\n", Version)
			return nil
		},
	}
}

// runSpec loads the spec at args[0] and runs it, passing args[1:] to the app.
// clic's global flags are resolved from cmd and threaded to the app via the
// context, keeping them out of the app's own flag namespace.
func runSpec(cmd *cobra.Command, args []string, force spec.Format) error {
	appSpec, err := clic.LoadSpec(resolveLocation(args[0]), force)
	if err != nil {
		return err
	}

	opts := provider.ResolveOptions(cmd.Flags())

	// the global -i flag (before the spec) opens the interactive studio instead
	// of running a single command headlessly. The studio handles its own OAuth2
	// login, so only the headless path resolves a token up front.
	if opts.Interactive {
		ctx := provider.WithOptions(cmd.Context(), opts)
		if appSpec.Auth != nil {
			ctx = provider.WithAuth(ctx, appSpec.Auth)
		}
		return launchStudio(ctx, appSpec, opts, args[0], args[1:])
	}

	if err := resolveOAuth(cmd.Context(), appSpec.Auth, opts); err != nil {
		return err
	}

	ctx := provider.WithOptions(cmd.Context(), opts)
	if appSpec.Auth != nil {
		ctx = provider.WithAuth(ctx, appSpec.Auth)
	}

	app, err := clic.NewAppFromSpec(appSpec)
	if err != nil {
		return fmt.Errorf("failed to create app: %w", err)
	}

	// the app reports its own errors via cobra; exit non-zero without re-reporting
	if err := app.RunContext(ctx, args[1:]); err != nil {
		os.Exit(1)
	}

	return nil
}

func validate(cmd *cobra.Command, args []string) error {
	appSpec, err := clic.LoadSpec(resolveLocation(args[0]), forceFormat(cmd))
	if err != nil {
		return err
	}

	return appSpec.Validate()
}

// resolveLocation maps a spec argument to a loadable location, falling back to
// the registry when it is neither a URL nor an existing file.
func resolveLocation(location string) string {
	if source.IsURL(location) || ioutil.FileExists(location) {
		return location
	}

	if reg, err := registry.Load(); err == nil {
		if path, ok := reg[location]; ok {
			return path
		}
	}

	return location
}

func addFormatFlags(cmd *cobra.Command) {
	cmd.Flags().Bool("openapi", false, "force interpreting the spec as OpenAPI")
	cmd.Flags().Bool("spec", false, "force interpreting the spec as a clic spec")
}

func forceFormat(cmd *cobra.Command) spec.Format {
	if openapi, _ := cmd.Flags().GetBool("openapi"); openapi {
		return spec.FormatOpenAPI
	}
	if clicFormat, _ := cmd.Flags().GetBool("spec"); clicFormat {
		return spec.FormatClic
	}
	return spec.FormatUnknown
}
