package gobuilder

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"

	"github.com/jefflinse/clic/builder"
	"github.com/jefflinse/clic/writer"
)

// The Go builder builds a native binary from Go source code.
type Go struct {
	sources *writer.Output
}

// New creates a new Go builder.
func New(sources *writer.Output) *Go {
	log.Println("creating Go builder")
	return &Go{sources: sources}
}

// Build compiles the Go file(s) in the source directory into a binary.
func (g Go) Build(outputFile string) (*builder.Output, error) {
	// go mod init
	if err := g.runBashCmd(fmt.Sprintf(`go mod init %s`, g.sources.Spec.Name)); err != nil {
		return nil, err
	}

	// go get
	if err := g.runBashCmd(`go get`); err != nil {
		return nil, err
	}

	// go build
	if err := g.runBashCmd(fmt.Sprintf(`go build -o %s`, outputFile)); err != nil {
		return nil, err
	}

	return &builder.Output{Type: "go", Path: outputFile}, nil
}

func (g Go) runBashCmd(cmd string) error {
	bashCmd := fmt.Sprintf("cd %s && %s", g.sources.Dir, cmd)
	command := exec.Command("/bin/bash", "-c", bashCmd)
	command.Env = os.Environ()

	stderr, err := command.StderrPipe()
	if err != nil {
		log.Fatalf("could not get stderr pipe: %v", err)
	}

	stdout, err := command.StdoutPipe()
	if err != nil {
		log.Fatalf("could not get stdout pipe: %v", err)
	}

	go func() {
		merged := io.MultiReader(stderr, stdout)
		scanner := bufio.NewScanner(merged)
		for scanner.Scan() {
			log.Printf(scanner.Text())
		}
	}()

	if err := command.Run(); err != nil {
		return err
	}

	return nil
}
