package main

import (
	"fmt"
	"os"

	"github.com/goccy/go-yaml"
	"github.com/jefflinse/clic"
	"github.com/spf13/cobra"
)

func convert(cmd *cobra.Command, args []string) error {
	appSpec, err := clic.LoadSpec(resolveLocation(args[0]), forceFormat(cmd))
	if err != nil {
		return err
	}

	if err := appSpec.Validate(); err != nil {
		return fmt.Errorf("invalid spec: %w", err)
	}

	data, err := yaml.Marshal(appSpec)
	if err != nil {
		return fmt.Errorf("failed to marshal clic spec: %w", err)
	}

	output, _ := cmd.Flags().GetString("output")
	if output == "" {
		fmt.Print(string(data))
		return nil
	}

	if err := os.WriteFile(output, data, 0644); err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "wrote clic spec to %s\n", output)
	return nil
}
