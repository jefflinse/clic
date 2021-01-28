package gobuilder

import (
	"bufio"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/jefflinse/clic/builder"
	"github.com/jefflinse/clic/writer"
	"github.com/rs/zerolog/log"
)

// The Go builder builds a native binary from Go source code.
type Go struct {
	sources *writer.Output
}

// New creates a new Go builder.
func New(sources *writer.Output) *Go {
	return &Go{sources: sources}
}

// Build compiles the Go file(s) in the source directory into a binary.
func (g Go) Build(outputFile string) (*builder.Output, error) {
	log.Info().Msg("building Go app")
	log.Debug().Str("path", outputFile).Msg("output file")

	if err := g.runGo("mod", "init", g.sources.Spec.Name); err != nil {
		return nil, err
	}

	if err := g.runGo("get"); err != nil {
		return nil, err
	}

	if err := g.runGo("build", "-o", outputFile); err != nil {
		return nil, err
	}

	return &builder.Output{Type: "Go", Path: outputFile, Spec: g.sources.Spec}, nil
}

func (g Go) runGo(args ...string) error {
	log.Debug().Str("cmd", "go "+strings.Join(args, " ")).Msg("executing Go command")
	command := exec.Command("go", args...)
	command.Dir = g.sources.Dir
	command.Env = os.Environ()

	stderr, err := command.StderrPipe()
	if err != nil {
		log.Fatal().Err(err).Msg("could not get stderr pipe")
	}

	stdout, err := command.StdoutPipe()
	if err != nil {
		log.Fatal().Err(err).Msg("could not get stdout pipe")
	}

	go func() {
		merged := io.MultiReader(stderr, stdout)
		scanner := bufio.NewScanner(merged)
		for scanner.Scan() {
			log.Trace().Msg(scanner.Text())
		}
	}()

	if err := command.Run(); err != nil {
		return err
	}

	return nil
}
