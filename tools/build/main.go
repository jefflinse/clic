// The compiler isn't a true compiler; it just takes a Handyman spec, generates a Go
// source file with the spec contents statically defined, and compiles it into a Go binary.
package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"
	"text/template"

	"github.com/jefflinse/handyman/spec"
)

const (
	// Template for main.go that gets stamped and compiled as the native binary.
	codegenTemplateFile = "codegen/main.template"
)

var codePath string

func main() {
	if len(os.Args) < 2 {
		usage()
	}

	specFilePath := os.Args[1]

	info, err := os.Stat(specFilePath)
	if os.IsNotExist(err) {
		exitWithError("file not found")
	} else if info.IsDir() {
		exitWithError("spec must be a file")
	}

	data, err := ioutil.ReadFile(specFilePath)
	if err != nil {
		exitWithError(err.Error())
	}

	// validate the app spec
	appSpec, err := spec.NewAppSpec(data)
	if err != nil {
		exitWithError(err.Error())
	} else if err := appSpec.Validate(); err != nil {
		exitWithError(err.Error())
	}

	// codegen and compilation
	if err := generateAppBinary(appSpec.Name, data); err != nil {
		exitWithError(err.Error())
	}
}

func generateAppBinary(name string, specData []byte) error {
	var err error

	codePath, err = ioutil.TempDir("", fmt.Sprint("handyman-", name, "-"))
	if err != nil {
		return err
	}

	code, err := ioutil.ReadFile(codegenTemplateFile)
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

	if err := ioutil.WriteFile(path.Join(codePath, "main.go"), []byte(b.String()), 0777); err != nil {
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

func exitWithError(message string) {
	fmt.Fprintln(os.Stderr, message)
	os.Exit(1)
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

func usage() {
	exitWithError(fmt.Sprintf("usage: %s [spec-file]", os.Args[0]))
}
