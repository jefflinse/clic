// The compiler isn't a true compiler; it just takes a clic spec, generates a Go
// source file with the spec contents statically defined, and compiles it into a Go binary.
package main

import (
	"fmt"
	goioutil "io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"
	"text/template"

	"github.com/jefflinse/clic/ioutil"
	"github.com/jefflinse/clic/spec"
	"github.com/urfave/cli/v2"
)

const (
	// Template for main.go that gets stamped and compiled as the native binary.
	codegenTemplateFile = "codegen/main.template"
)

var codePath string

func build(clicCtx *cli.Context) error {
	if clicCtx.NArg() != 1 {
		cli.ShowCommandHelpAndExit(clicCtx, "build", 1)
	}

	specFilePath := clicCtx.Args().First()
	if !ioutil.FileExists(specFilePath) {
		return fmt.Errorf("file not found")
	}

	data, err := goioutil.ReadFile(specFilePath)
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
	var err error

	codePath, err = goioutil.TempDir("", fmt.Sprint("clic-", name, "-"))
	if err != nil {
		return err
	}

	code, err := goioutil.ReadFile(codegenTemplateFile)
	if err != nil {
		return err
	}

	tpl, err := template.New("main").Parse(string(code))
	if err != nil {
		return err
	}

	tplData := map[string]string{"SpecData": string(specData)}
	b := strings.Builder{}
	if err := tpl.Execute(&b, tplData); err != nil {
		return err
	}

	if err := goioutil.WriteFile(path.Join(codePath, "main.go"), []byte(b.String()), 0777); err != nil {
		return err
	}

	appPath, _ := os.Getwd()

	// go mod init
	if err := runBashCmd(fmt.Sprintf(`go mod init %s`, name)); err != nil {
		return err
	}

	// go get
	if err := runBashCmd(`go get`); err != nil {
		return err
	}

	// go build
	if err := runBashCmd(fmt.Sprintf(`go build -o %s/%s`, appPath, name)); err != nil {
		return err
	}

	if err := os.RemoveAll(codePath); err != nil {
		return err
	}

	return nil
}

func runBashCmd(name string) error {
	stderr := strings.Builder{}
	bashCmd := fmt.Sprintf("cd %s && %s", codePath, name)
	command := exec.Command("/bin/bash", "-c", bashCmd)
	command.Env = os.Environ()
	command.Stdout = os.Stdout
	command.Stderr = &stderr

	if err := command.Run(); err != nil {
		fmt.Fprint(os.Stderr, stderr.String())
	}

	return nil
}
