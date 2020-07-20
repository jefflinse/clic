package spec_test

import (
	"testing"

	"github.com/jefflinse/handyman/spec"
	"github.com/stretchr/testify/assert"
)

func TestNewCommandSpec(t *testing.T) {
	tests := []struct {
		name  string
		json  string
		valid bool
	}{
		{
			name:  "valid JSON, no subcommands",
			json:  `{"name":"cmd","description":"the cmd","type":"NOOP"}`,
			valid: true,
		},
		{
			name:  "valid JSON, with empty subcommands",
			json:  `{"name":"cmd","description":"the cmd","type":"NOOP","subcommands":[]}`,
			valid: true,
		},
		{
			name:  "valid JSON, with one subcommand",
			json:  `{"name":"cmd","description":"the cmd","type":"NOOP","commands":[{"name":"sub","description":"a subcommand"}]}`,
			valid: true,
		},
		{
			name:  "valid JSON, with multiple subcommand",
			json:  `{"name":"cmd","description":"the cmd","type":"NOOP","commands":[{"name":"sub1","description":"a subcommand"},{"name":"sub2","description":"another subcommand"}]}`,
			valid: true,
		},
		{
			name:  "invalid JSON",
			json:  `{"name":"cmd","description:"the cmd","type":"NOOP"}`,
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
			name: "valid, noop type",
			command: spec.Command{
				Name:        "cmd",
				Description: "the cmd",
				Type:        spec.NoopCommandType,
			},
			valid: true,
		},
		{
			name: "valid, subcommand type with one valid subcommand",
			command: spec.Command{
				Name:        "cmd",
				Description: "the cmd",
				Type:        spec.SubcommandsCommandType,
				Subcommands: []*spec.Command{
					{
						Name:        "cmd",
						Description: "the cmd",
						Type:        spec.NoopCommandType,
					},
				},
			},
			valid: true,
		},
		{
			name: "valid, subcommand type with multiple valid subcommands",
			command: spec.Command{
				Name:        "cmd",
				Description: "the cmd",
				Type:        spec.SubcommandsCommandType,
				Subcommands: []*spec.Command{
					{
						Name:        "cmd1",
						Description: "a cmd",
						Type:        spec.NoopCommandType,
					},
					{
						Name:        "cmd2",
						Description: "another cmd",
						Type:        spec.NoopCommandType,
					},
				},
			},
			valid: true,
		},
		{
			name: "valid, exec type with exec",
			command: spec.Command{
				Name:        "cmd",
				Description: "the cmd",
				Type:        spec.ExecCommandType,
				Exec:        "echo hello",
			},
			valid: true,
		},
		{
			name: "valid, lambda type with lambda ARN, no parameters",
			command: spec.Command{
				Name:        "cmd",
				Description: "the cmd",
				Type:        spec.LambdaCommandType,
				LambdaARN:   "aws:arn:region:account:lambda:function:name:version",
			},
			valid: true,
		},
		{
			name: "valid, lambda type with lambda ARN, with one parameter",
			command: spec.Command{
				Name:        "cmd",
				Description: "the cmd",
				Type:        spec.LambdaCommandType,
				LambdaARN:   "aws:arn:region:account:lambda:function:name:version",
				LambdaRequestParameters: []*spec.Parameter{
					{
						Name:        "param",
						Description: "the param",
						Type:        spec.StringParamType,
					},
				},
			},
			valid: true,
		},
		{
			name: "valid, lambda type with lambda ARN, with multiple parameters",
			command: spec.Command{
				Name:        "cmd",
				Description: "the cmd",
				Type:        spec.LambdaCommandType,
				LambdaARN:   "aws:arn:region:account:lambda:function:name:version",
				LambdaRequestParameters: []*spec.Parameter{
					{
						Name:        "param1",
						Description: "the first param",
						Type:        spec.StringParamType,
					},
					{
						Name:        "param2",
						Description: "the second param",
						Type:        spec.StringParamType,
						Required:    true,
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
			name:    "invalid, missing type",
			command: spec.Command{Name: "cmd", Description: "the cmd"},
			valid:   false,
		},
		{
			name: "invalid, subcommand is invalid",
			command: spec.Command{
				Name:        "cmd",
				Description: "the cmd",
				Type:        spec.SubcommandsCommandType,
				Subcommands: []*spec.Command{
					{
						Name: "subcmd",
					},
				},
			},
			valid: false,
		},
		{
			name: "invalid, subcommand type without subcommands",
			command: spec.Command{
				Name:        "cmd",
				Description: "the cmd",
				Type:        spec.SubcommandsCommandType,
			},
			valid: false,
		},
		{
			name: "invalid, subcommand type with empty subcommands",
			command: spec.Command{
				Name:        "cmd",
				Description: "the cmd",
				Type:        spec.SubcommandsCommandType,
				Subcommands: []*spec.Command{},
			},
			valid: false,
		},
		{
			name: "invalid, exec type without exec",
			command: spec.Command{
				Name:        "cmd",
				Description: "the cmd",
				Type:        spec.ExecCommandType,
			},
			valid: false,
		},
		{
			name: "invalid, lambda type without lambda",
			command: spec.Command{
				Name:        "cmd",
				Description: "the cmd",
				Type:        spec.LambdaCommandType,
			},
			valid: false,
		},
		{
			name: "invalid, lambda type with invalid parameter",
			command: spec.Command{
				Name:        "cmd",
				Description: "the cmd",
				Type:        spec.LambdaCommandType,
				LambdaARN:   "aws:arn:region:account:lambda:function:name:version",
				LambdaRequestParameters: []*spec.Parameter{
					{
						Name:        "param",
						Description: "the param",
						Type:        "invalid",
					},
				},
			},
			valid: false,
		},
		{
			name: "invalid command type",
			command: spec.Command{
				Name:        "cmd",
				Description: "the cmd",
				Type:        "INVALID",
				Subcommands: []*spec.Command{},
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

func TestNewInvalidCommandSpecError(t *testing.T) {
	err := spec.NewInvalidCommandSpecError("the reason")
	assert.EqualError(t, err, "invalid command spec: the reason")
}
