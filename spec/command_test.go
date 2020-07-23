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
