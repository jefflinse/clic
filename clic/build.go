// The compiler isn't a true compiler; it just takes a clic spec, generates a Go
// source file with the spec contents statically defined, and compiles it into a Go binary.
package main

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/jefflinse/clic/ioutil"
	"github.com/jefflinse/clic/spec"
	"github.com/spf13/cobra"
)

// codegenTemplate is the main.go source that gets stamped and compiled as the native
// binary. It's embedded into the clic binary so that builds work from any directory.
//
//go:embed codegen/main.template
var codegenTemplate string

func buildCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "build <specfile>",
		Short: "bakes a clic spec into a native Go binary",
		Args:  cobra.ExactArgs(1),
		RunE:  build,
	}
}

func build(cmd *cobra.Command, args []string) error {
	specFilePath := args[0]
	if !ioutil.FileExists(specFilePath) {
		return fmt.Errorf("file not found")
	}

	data, err := os.ReadFile(specFilePath)
	if err != nil {
		return fmt.Errorf("failed to read spec: %w", err)
	}

	// validate the app spec
	appSpec, err := spec.NewAppSpec(data)
	if err != nil {
		return fmt.Errorf("failed to create app: %w", err)
	} else if err := appSpec.Validate(); err != nil {
		return fmt.Errorf("invalid spec: %w", err)
	}

	// codegen and compilation
	if err := generateAppBinary(appSpec.Name, data); err != nil {
		return fmt.Errorf("failed to build app: %w", err)
	}

	return nil
}

func generateAppBinary(name string, specData []byte) error {
	codePath, err := os.MkdirTemp("", fmt.Sprint("clic-", name, "-"))
	if err != nil {
		return err
	}
	defer os.RemoveAll(codePath)

	tpl, err := template.New("main").Parse(codegenTemplate)
	if err != nil {
		return err
	}

	tplData := map[string]string{"SpecData": string(specData)}
	b := strings.Builder{}
	if err := tpl.Execute(&b, tplData); err != nil {
		return err
	}

	if err := os.WriteFile(filepath.Join(codePath, "main.go"), []byte(b.String()), 0644); err != nil {
		return err
	}

	appPath, err := os.Getwd()
	if err != nil {
		return err
	}

	steps := []string{
		fmt.Sprintf("go mod init %s", name),
		"go mod tidy",
		fmt.Sprintf("go build -o %s", filepath.Join(appPath, name)),
	}
	for _, step := range steps {
		if err := runBashCmd(codePath, step); err != nil {
			return err
		}
	}

	return nil
}

func runBashCmd(dir, name string) error {
	stderr := strings.Builder{}
	command := exec.Command("/bin/bash", "-c", name)
	command.Dir = dir
	command.Env = os.Environ()
	command.Stdout = os.Stdout
	command.Stderr = &stderr

	if err := command.Run(); err != nil {
		return fmt.Errorf("%q failed: %w\n%s", name, err, stderr.String())
	}

	return nil
}
