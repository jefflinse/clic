package spec_test

import (
	"testing"

	"github.com/jefflinse/handyman/spec"
	"github.com/stretchr/testify/assert"
)

func TestNewAppSpec(t *testing.T) {
	tests := []struct {
		name    string
		content string
		valid   bool
	}{
		{
			name:    "succeeds on valid JSON",
			content: `{"name":"app","description":"the app"}`,
			valid:   true,
		},
		{
			name:    "succeeds on valid YAML",
			content: "name: app\ndescription: the app",
			valid:   true,
		},
		{
			name:    "fails on empty content",
			content: ``,
			valid:   false,
		},
		{
			name:    "fails on invalid JSON",
			content: `{"name":"app","description:"the app"}`,
			valid:   false,
		},
		{
			name:    "fails on invalid YAML",
			content: "name",
			valid:   false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			s, err := spec.NewAppSpec([]byte(test.content))
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
		json  string
		valid bool
	}{
		{
			name:  "is valid with just name and description",
			json:  `{"name":"app","description":"app"}`,
			valid: true,
		},
		{
			name:  "is valid with empty command set",
			json:  `{"name":"app","description":"app","commands":[]}`,
			valid: true,
		},
		{
			name:  "is valid with one command",
			json:  `{"name":"app","description":"app","commands":[{"name":"cmd","description":"cmd","noop":{}}]}`,
			valid: true,
		},
		{
			name:  "is valid with multiple valid commands",
			json:  `{"name":"app","description":"app","commands":[{"name":"cmd1","description":"cmd1","noop":{}},{"name":"cmd2","description":"cmd2","noop":{}}]}`,
			valid: true,
		},
		{
			name:  "is invalid when missing name",
			json:  `{"description":"app"}`,
			valid: false,
		},
		{
			name:  "is invalid when missing description",
			json:  `{"name":"app"}`,
			valid: false,
		},
		{
			name:  "is invalid when any command is invalid",
			json:  `{"name":"app","description":"app","commands":[{"name":"cmd1"},{"name":"cmd2","description":"cmd2","noop":{}}]}`,
			valid: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			app, err := spec.NewAppSpec([]byte(test.json))
			assert.NoError(t, err)

			err = app.Validate()
			if test.valid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestNewInvalidAppSpecError(t *testing.T) {
	err := spec.NewInvalidAppSpecError("the reason")
	assert.EqualError(t, err, "invalid app spec: the reason")
}
