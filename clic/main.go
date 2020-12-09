package main

import (
	"fmt"
	goioutil "io/ioutil"
	"os"

	"github.com/jefflinse/clic"
	"github.com/jefflinse/clic/ioutil"
	"github.com/jefflinse/clic/registry"
	"github.com/jefflinse/clic/spec"
	"github.com/urfave/cli/v2"
)

// Version is stamped by the build process.
var Version string

func main() {
	app := &cli.App{
		Name:                  "clic",
		HelpName:              "clic",
		Usage:                 "the clic CLI",
		Commands:              commands(),
		HideHelp:              true,
		CustomAppHelpTemplate: clic.AppHelpTemplate(),
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func commands() []*cli.Command {
	return []*cli.Command{
		{
			Name:               "build",
			Usage:              "bakes a clic spec into a native Go binary",
			CustomHelpTemplate: clic.CommandHelpTemplate(),
			ArgsUsage:          "specfile",
			Flags:              []cli.Flag{},
			Action:             build,
		},
		{
			Name:               "prune-registry",
			Usage:              "removes registered apps whose spec files no longer exist",
			CustomHelpTemplate: clic.CommandHelpTemplate(),
			Flags:              []cli.Flag{},
			Action:             pruneRegistry,
		},
		{
			Name:               "register",
			Usage:              "registers an app with the specified path",
			CustomHelpTemplate: clic.CommandHelpTemplate(),
			ArgsUsage:          "specfile",
			Flags:              []cli.Flag{},
			Action:             register,
		},
		{
			Name:               "list-registry",
			Usage:              "lists registered apps",
			CustomHelpTemplate: clic.CommandHelpTemplate(),
			Flags:              []cli.Flag{},
			Action:             listRegistry,
		},
		{
			Name:               "run",
			Usage:              "directly run a clic spec",
			CustomHelpTemplate: clic.CommandHelpTemplate(),
			ArgsUsage:          "specfile",
			Flags:              []cli.Flag{},
			Action:             run,
		},
		{
			Name:               "unregister",
			Usage:              "unregisters an app with the specified name",
			CustomHelpTemplate: clic.CommandHelpTemplate(),
			ArgsUsage:          "name",
			Flags:              []cli.Flag{},
			Action:             unregister,
		},
		{
			Name:               "validate",
			Usage:              "validate a clic spec",
			CustomHelpTemplate: clic.CommandHelpTemplate(),
			ArgsUsage:          "specfile",
			Flags:              []cli.Flag{},
			Action:             validate,
		},
		{
			Name:               "version",
			Usage:              "print the current clic CLI version",
			CustomHelpTemplate: clic.CommandHelpTemplate(),
			Flags:              []cli.Flag{},
			Action: func(ctx *cli.Context) error {
				fmt.Printf("clic CLI version %s\n", Version)
				return nil
			},
		},
	}
}

func run(clicCtx *cli.Context) error {
	if clicCtx.NArg() < 1 {
		cli.ShowCommandHelpAndExit(clicCtx, "run", 1)
	}

	specFile := clicCtx.Args().First()
	var content []byte
	if !ioutil.FileExists(specFile) {
		// spec file not found, check the registry
		appName := clicCtx.Args().First()
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

	content, err := goioutil.ReadFile(specFile)
	if err != nil {
		return fmt.Errorf("failed to read spec file: %w", err)
	}

	app, err := clic.NewApp(content)
	if err != nil {
		return fmt.Errorf("failed to create app: %w", err)
	}

	// join the runner binary and spec filename into
	// a single string to be used as arg[0] of the app
	if err := app.Run(clicCtx.Args().Tail()); err != nil {
		fmt.Fprintf(os.Stderr, err.Error())
		os.Exit(1)
	}

	return nil
}

func validate(clicCtx *cli.Context) error {
	if clicCtx.NArg() != 1 {
		cli.ShowCommandHelpAndExit(clicCtx, "validate", 1)
	}

	content, err := goioutil.ReadFile(clicCtx.Args().First())
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
