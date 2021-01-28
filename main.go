package main

import (
	"os"

	"github.com/jefflinse/clic/cmd"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{
		Out:        os.Stderr,
		PartsOrder: []string{"message"},
	})
	cmd.Execute()
}
