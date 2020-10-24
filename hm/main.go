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
			Name:  "registry",
			Usage: "add or remove registered apps",
			Subcommands: []*cli.Command{
				{
					Name:      "add",
					Usage:     "registers an app with the specified path",
					ArgsUsage: "specfile",
					Flags:     []cli.Flag{},
					Action: func(ctx *cli.Context) error {
						return nil
					},
				},
				{
					Name:  "list",
					Usage: "lists registered apps",
					Flags: []cli.Flag{},
					Action: func(ctx *cli.Context) error {
						return nil
					},
				},
				{
					Name:      "rm",
					Usage:     "unregisters an app with the specified name",
					ArgsUsage: "name",
					Flags:     []cli.Flag{},
					Action: func(ctx *cli.Context) error {
						return nil
					},
				},
			},
			Flags: []cli.Flag{},
		},
		{
			Name:      "run",
			Usage:     "directly run a handyman spec",
			ArgsUsage: "specfile",
			Flags:     []cli.Flag{},
			Action:    run,
		},
		{
			Name:      "validate",
			Usage:     "validate a handyman spec",
			ArgsUsage: "specfile",
			Flags:     []cli.Flag{},
			Action:    validate,
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
