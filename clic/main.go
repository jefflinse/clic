package main

import (
	"fmt"
	"os"

	"github.com/jefflinse/clic"
	"github.com/jefflinse/clic/ioutil"
	"github.com/jefflinse/clic/registry"
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
		Use:           "clic",
		Short:         "the clic CLI",
		Version:       Version,
		SilenceUsage:  true,
		SilenceErrors: false,
	}

	root.AddCommand(
		buildCmd(),
		runCmd(),
		validateCmd(),
		registerCmd(),
		unregisterCmd(),
		listRegistryCmd(),
		pruneRegistryCmd(),
		versionCmd(),
	)

	return root
}

func runCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run <specfile> [args...]",
		Short: "directly run a clic spec",
		Args:  cobra.MinimumNArgs(1),
		RunE:  run,
	}

	// stop parsing flags after the spec file so the rest are passed through to the app
	cmd.Flags().SetInterspersed(false)

	return cmd
}

func validateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "validate <specfile>",
		Short: "validate a clic spec",
		Args:  cobra.ExactArgs(1),
		RunE:  validate,
	}
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

func run(cmd *cobra.Command, args []string) error {
	specFile := args[0]
	if !ioutil.FileExists(specFile) {
		// spec file not found, check the registry
		appName := args[0]
		reg, err := registry.Load()
		if err != nil {
			return fmt.Errorf("failed to load registry: %w", err)
		}

		if path, ok := reg[appName]; ok {
			specFile = path
		} else {
			return fmt.Errorf("'%s' is not a valid spec file path or registered app name", appName)
		}
	}

	content, err := os.ReadFile(specFile)
	if err != nil {
		return fmt.Errorf("failed to read spec file: %w", err)
	}

	app, err := clic.NewApp(content)
	if err != nil {
		return fmt.Errorf("failed to create app: %w", err)
	}

	// the app reports its own errors via cobra; exit non-zero without re-reporting
	if err := app.Run(args[1:]); err != nil {
		os.Exit(1)
	}

	return nil
}

func validate(cmd *cobra.Command, args []string) error {
	content, err := os.ReadFile(args[0])
	if err != nil {
		return fmt.Errorf("failed to read spec file: %w", err)
	}

	appSpec, err := spec.NewAppSpec(content)
	if err != nil {
		return fmt.Errorf("failed to parse spec file: %w", err)
	}

	if err := appSpec.Validate(); err != nil {
		return err
	}

	return nil
}
