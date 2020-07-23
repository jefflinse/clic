package spec_test

import (
	"testing"

	"github.com/jefflinse/handyman/provider"
	"github.com/jefflinse/handyman/provider/noop"
	"github.com/jefflinse/handyman/spec"
	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli/v2"
)

func TestNewCommandSpec(t *testing.T) {
	tests := []struct {
		name  string
		json  string
		valid bool
	}{
		{
			name:  "succeeds on valid JSON",
			json:  `{"name":"cmd","description":"the cmd"}`,
			valid: true,
		},
		{
			name:  "succeeds even if provider type isn't recognized",
			json:  `{"name":"cmd","description":"the cmd","invalid":{"foo":"bar"}}`,
			valid: true,
		},
		{
			name:  "fails on invalid JSON",
			json:  `{"name":"cmd","description:"the cmd"`,
			valid: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			s, err := spec.NewCommandSpec([]byte(test.json))
			if test.valid {
				assert.NoError(t, err)
				assert.NotNil(t, s)
			} else {
				assert.Error(t, err)
				assert.Nil(t, s)
			}
		})
	}
}

func TestCommand_CLICommand(t *testing.T) {
	noopProvider := func() provider.Provider {
		prov, _ := noop.New(nil)
		return prov
	}

	tests := []struct {
		name     string
		cmd      *spec.Command
		validate func(cliCmd *cli.Command)
	}{
		{
			name: "assigns name and usage",
			cmd:  &spec.Command{Name: "foo", Description: "bar", Provider: noopProvider()},
			validate: func(cliCmd *cli.Command) {
				assert.Equal(t, "foo", cliCmd.Name)
				assert.Equal(t, "bar", cliCmd.Usage)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cliCmd := test.cmd.CLICommand()
			assert.NotNil(t, cliCmd)
			test.validate(cliCmd)
		})
	}
}

func TestCommand_Validate(t *testing.T) {
	tests := []struct {
		name  string
		json  string
		valid bool
	}{
		{
			name:  "is valid when a known provider is specified",
			json:  `{"name":"cmd","description":"the cmd","noop":{}}`,
			valid: true,
		},
		{
			name:  "is invalid when missing name",
			json:  `{"description":"cmd","noop":{}}`,
			valid: false,
		},
		{
			name:  "is invalid when missing description",
			json:  `{"name":"cmd","noop":{}}`,
			valid: false,
		},
		{
			name:  "is valid when missing provider",
			json:  `{"name":"cmd","description":"the cmd"}`,
			valid: false,
		},
		{
			name:  "is invalid when an unknown provider is specified",
			json:  `{"name":"cmd","description":"the cmd","invalid":{"foo":"bar"}}`,
			valid: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cmd, err := spec.NewCommandSpec([]byte(test.json))
			assert.NoError(t, err)

			err = cmd.Validate()
			if test.valid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestNewInvalidCommandSpecError(t *testing.T) {
	err := spec.NewInvalidCommandSpecError("the reason")
	assert.EqualError(t, err, "invalid command spec: the reason")
}
