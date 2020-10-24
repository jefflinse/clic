package main

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/jefflinse/handyman/registry"
	"github.com/jefflinse/handyman/spec"
	"github.com/urfave/cli/v2"
)

func listRegistry(hmCtx *cli.Context) error {
	if hmCtx.NArg() > 0 {
		cli.ShowCommandHelpAndExit(hmCtx, "list-registry", 1)
	}

	reg, err := registry.Load()
	if err != nil {
		return fmt.Errorf("error loading registry: %w", err)
	}

	longestNameLen := 0
	for name := range reg {
		if len(name) > longestNameLen {
			longestNameLen = len(name)
		}
	}

	for name, path := range reg {
		paddingLen := longestNameLen - len(name)
		fmt.Printf("%s: %s%s\n", name, strings.Repeat(" ", paddingLen), path)
	}

	fmt.Println()
	return nil
}

func register(hmCtx *cli.Context) error {
	if hmCtx.NArg() != 1 {
		cli.ShowCommandHelpAndExit(hmCtx, "register", 1)
	}

	reg, err := registry.Load()
	if err != nil {
		return fmt.Errorf("error loading registry: %w", err)
	}

	path := hmCtx.Args().First()
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("invalid file path: %w", err)
	}

	content, err := ioutil.ReadFile(absPath)
	if err != nil {
		return fmt.Errorf("failed to read spec file: %w", err)
	}

	appSpec, err := spec.NewAppSpec(content)
	if err != nil {
		return fmt.Errorf("failed to parse app spec: %w", err)
	}

	if err := appSpec.Validate(); err != nil {
		return fmt.Errorf("invalid spec: %w", err)
	}

	if err := reg.Add(appSpec.Name, absPath); err != nil {
		return fmt.Errorf("failed to register app: %w", err)
	}

	return nil
}

func pruneRegistry(hmCtx *cli.Context) error {
	if hmCtx.NArg() > 0 {
		cli.ShowCommandHelpAndExit(hmCtx, "prune-registry", 1)
	}

	reg, err := registry.Load()
	if err != nil {
		return fmt.Errorf("error loading registry: %w", err)
	}

	numPruned, err := reg.Prune()
	if err != nil {
		return err
	}

	fmt.Printf("removed %d stale app registration(s)\n", numPruned)
	return nil
}

func unregister(hmCtx *cli.Context) error {
	if hmCtx.NArg() != 1 {
		cli.ShowCommandHelpAndExit(hmCtx, "unregister", 1)
	}

	reg, err := registry.Load()
	if err != nil {
		return fmt.Errorf("error loading registry: %w", err)
	}

	return reg.Remove(hmCtx.Args().First())
}
