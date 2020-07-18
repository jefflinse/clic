package spec_test

import (
	"testing"

	"github.com/jefflinse/handyman/spec"
	"github.com/stretchr/testify/assert"
)

func TestNewAppSpec(t *testing.T) {
	tests := []struct {
		name  string
		json  string
		valid bool
	}{
		{
			name:  "valid JSON, no commands",
			json:  `{"name":"app","description":"the app"}`,
			valid: true,
		},
		{
			name:  "valid JSON, with empty commands",
			json:  `{"name":"app","description":"the app","commands":[]}`,
			valid: true,
		},
		{
			name:  "valid JSON, with one command",
			json:  `{"name":"app","description":"the app","commands":[{"name":"cmd","description":"a command"}]}`,
			valid: true,
		},
		{
			name:  "valid JSON, with multiple commands",
			json:  `{"name":"app","description":"the app","commands":[{"name":"cmd1","description":"a command"},{"name":"cmd2","description":"another command"}]}`,
			valid: true,
		},
		{
			name:  "invalid JSON",
			json:  `{"name":"app","description:"the app"}`,
			valid: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			s, err := spec.NewAppSpec([]byte(test.json))
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

func TestApp_Validate(t *testing.T) {
	tests := []struct {
		name  string
		app   spec.App
		valid bool
	}{
		{
			name: "valid, no commands",
			app: spec.App{
				Name:        "app",
				Description: "the app",
			},
			valid: true,
		},
		{
			name: "valid, with empty commands",
			app: spec.App{
				Name:        "app",
				Description: "the app",
				Commands:    []*spec.Command{},
			},
			valid: true,
		},
		{
			name: "valid, with one valid command",
			app: spec.App{
				Name:        "app",
				Description: "the app",
				Commands: []*spec.Command{
					&spec.Command{
						Name:        "cmd",
						Description: "the cmd",
					},
				},
			},
			valid: true,
		},
		{
			name: "valid, with multiple valid commands",
			app: spec.App{
				Name:        "app",
				Description: "the app",
				Commands: []*spec.Command{
					&spec.Command{
						Name:        "cmd1",
						Description: "a cmd",
					},
					&spec.Command{
						Name:        "cmd2",
						Description: "another cmd",
					},
				},
			},
			valid: true,
		},
		{
			name:  "invalid, missing name",
			app:   spec.App{Description: "the app"},
			valid: false,
		},
		{
			name:  "invalid, missing description",
			app:   spec.App{Name: "app"},
			valid: false,
		},
		{
			name: "invalid, command is invalid",
			app: spec.App{
				Name:        "app",
				Description: "the app",
				Commands: []*spec.Command{
					&spec.Command{
						Name: "cmd1",
					},
				},
			},
			valid: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.app.Validate()
			if test.valid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestNewCommandSpec(t *testing.T) {
	tests := []struct {
		name  string
		json  string
		valid bool
	}{
		{
			name:  "valid JSON, no subcommands",
			json:  `{"name":"cmd","description":"the cmd"}`,
			valid: true,
		},
		{
			name:  "valid JSON, with empty subcommands",
			json:  `{"name":"cmd","description":"the cmd","subcommands":[]}`,
			valid: true,
		},
		{
			name:  "valid JSON, with one subcommand",
			json:  `{"name":"cmd","description":"the cmd","commands":[{"name":"sub","description":"a subcommand"}]}`,
			valid: true,
		},
		{
			name:  "valid JSON, with multiple subcommand",
			json:  `{"name":"cmd","description":"the cmd","commands":[{"name":"sub1","description":"a subcommand"},{"name":"sub2","description":"another subcommand"}]}`,
			valid: true,
		},
		{
			name:  "invalid JSON",
			json:  `{"name":"cmd","description:"the cmd"}`,
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

func TestCommand_Validate(t *testing.T) {
	tests := []struct {
		name    string
		command spec.Command
		valid   bool
	}{
		{
			name: "valid, no subcommands",
			command: spec.Command{
				Name:        "cmd",
				Description: "the cmd",
			},
			valid: true,
		},
		{
			name: "valid, with empty subcommands",
			command: spec.Command{
				Name:        "cmd",
				Description: "the cmd",
				Subcommands: []*spec.Command{},
			},
			valid: true,
		},
		{
			name: "valid, with one valid subcommand",
			command: spec.Command{
				Name:        "cmd",
				Description: "the cmd",
				Subcommands: []*spec.Command{
					&spec.Command{
						Name:        "cmd",
						Description: "the cmd",
					},
				},
			},
			valid: true,
		},
		{
			name: "valid, with multiple valid subcommands",
			command: spec.Command{
				Name:        "cmd",
				Description: "the cmd",
				Subcommands: []*spec.Command{
					&spec.Command{
						Name:        "cmd1",
						Description: "a cmd",
					},
					&spec.Command{
						Name:        "cmd2",
						Description: "another cmd",
					},
				},
			},
			valid: true,
		},
		{
			name:    "invalid, missing name",
			command: spec.Command{Description: "the cmd"},
			valid:   false,
		},
		{
			name:    "invalid, missing description",
			command: spec.Command{Name: "cmd"},
			valid:   false,
		},
		{
			name: "invalid, subcommand is invalid",
			command: spec.Command{
				Name:        "cmd",
				Description: "the cmd",
				Subcommands: []*spec.Command{
					&spec.Command{
						Name: "subcmd",
					},
				},
			},
			valid: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.command.Validate()
			if test.valid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}