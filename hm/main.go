package main

import (
	"fmt"
	goioutil "io/ioutil"
	"os"

	"github.com/jefflinse/handyman"
	"github.com/jefflinse/handyman/ioutil"
	"github.com/jefflinse/handyman/registry"
	"github.com/jefflinse/handyman/spec"
	"github.com/urfave/cli/v2"
)

// Version is stamped by the build process.
var Version string

func main() {
	app := &cli.App{
		Name:                  "hm",
		HelpName:              "hm",
		Usage:                 "the handyman CLI",
		Commands:              commands(),
		HideHelp:              true,
		CustomAppHelpTemplate: handyman.AppHelpTemplate(),
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
			Usage:              "bakes a handyman spec into a native Go binary",
			CustomHelpTemplate: handyman.CommandHelpTemplate(),
			ArgsUsage:          "specfile",
			Flags:              []cli.Flag{},
			Action:             build,
		},
		{
			Name:               "prune-registry",
			Usage:              "removes registered apps whose spec files no longer exist",
			CustomHelpTemplate: handyman.CommandHelpTemplate(),
			Flags:              []cli.Flag{},
			Action:             pruneRegistry,
		},
		{
			Name:               "register",
			Usage:              "registers an app with the specified path",
			CustomHelpTemplate: handyman.CommandHelpTemplate(),
			ArgsUsage:          "specfile",
			Flags:              []cli.Flag{},
			Action:             register,
		},
		{
			Name:               "list-registry",
			Usage:              "lists registered apps",
			CustomHelpTemplate: handyman.CommandHelpTemplate(),
			Flags:              []cli.Flag{},
			Action:             listRegistry,
		},
		{
			Name:               "run",
			Usage:              "directly run a handyman spec",
			CustomHelpTemplate: handyman.CommandHelpTemplate(),
			ArgsUsage:          "specfile",
			Flags:              []cli.Flag{},
			Action:             run,
		},
		{
			Name:               "unregister",
			Usage:              "unregisters an app with the specified name",
			CustomHelpTemplate: handyman.CommandHelpTemplate(),
			ArgsUsage:          "name",
			Flags:              []cli.Flag{},
			Action:             unregister,
		},
		{
			Name:               "validate",
			Usage:              "validate a handyman spec",
			CustomHelpTemplate: handyman.CommandHelpTemplate(),
			ArgsUsage:          "specfile",
			Flags:              []cli.Flag{},
			Action:             validate,
		},
		{
			Name:               "version",
			Usage:              "print the current handyman CLI version",
			CustomHelpTemplate: handyman.CommandHelpTemplate(),
			Flags:              []cli.Flag{},
			Action: func(ctx *cli.Context) error {
				fmt.Printf("handyman CLI version %s\n", Version)
				return nil
			},
		},
	}
}

func run(hmCtx *cli.Context) error {
	if hmCtx.NArg() < 1 {
		cli.ShowCommandHelpAndExit(hmCtx, "run", 1)
	}

	specFile := hmCtx.Args().First()
	var content []byte
	if !ioutil.FileExists(specFile) {
		// spec file not found, check the registry
		appName := hmCtx.Args().First()
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

	app, err := handyman.NewApp(content)
	if err != nil {
		return fmt.Errorf("failed to create app: %w", err)
	}

	// join the runner binary and spec filename into
	// a single string to be used as arg[0] of the app
	if err := app.Run(hmCtx.Args().Tail()); err != nil {
		fmt.Fprintf(os.Stderr, err.Error())
		os.Exit(1)
	}

	return nil
}

func validate(hmCtx *cli.Context) error {
	if hmCtx.NArg() != 1 {
		cli.ShowCommandHelpAndExit(hmCtx, "validate", 1)
	}

	content, err := goioutil.ReadFile(hmCtx.Args().First())
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
