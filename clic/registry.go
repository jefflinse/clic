package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jefflinse/clic/registry"
	"github.com/jefflinse/clic/spec"
	"github.com/spf13/cobra"
)

func registerCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "register <specfile>",
		Short: "registers an app with the specified path",
		Args:  cobra.ExactArgs(1),
		RunE:  register,
	}
}

func unregisterCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "unregister <name>",
		Short: "unregisters an app with the specified name",
		Args:  cobra.ExactArgs(1),
		RunE:  unregister,
	}
}

func listRegistryCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list-registry",
		Short: "lists registered apps",
		Args:  cobra.NoArgs,
		RunE:  listRegistry,
	}
}

func pruneRegistryCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "prune-registry",
		Short: "removes registered apps whose spec files no longer exist",
		Args:  cobra.NoArgs,
		RunE:  pruneRegistry,
	}
}

func listRegistry(cmd *cobra.Command, args []string) error {
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

func register(cmd *cobra.Command, args []string) error {
	reg, err := registry.Load()
	if err != nil {
		return fmt.Errorf("error loading registry: %w", err)
	}

	absPath, err := filepath.Abs(args[0])
	if err != nil {
		return fmt.Errorf("invalid file path: %w", err)
	}

	content, err := os.ReadFile(absPath)
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

func pruneRegistry(cmd *cobra.Command, args []string) error {
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

func unregister(cmd *cobra.Command, args []string) error {
	reg, err := registry.Load()
	if err != nil {
		return fmt.Errorf("error loading registry: %w", err)
	}

	return reg.Remove(args[0])
}
