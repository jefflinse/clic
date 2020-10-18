package provider_test

import (
	"testing"

	"github.com/jefflinse/handyman/provider"
	"github.com/stretchr/testify/assert"
)

func TestNewParameterSpec(t *testing.T) {
	tests := []struct {
		name    string
		content string
		valid   bool
	}{
		{
			name:    "succeeds on valid JSON",
			content: `{"name":"cmd","description":"the cmd"}`,
			valid:   true,
		},
		{
			name:    "succeeds on valid YAML",
			content: "name: cmd\ndescription: the cmd",
			valid:   true,
		},
		{
			name:    "json parsing succeeds even if provider type isn't recognized",
			content: `{"name":"cmd","description":"the cmd","invalid":{"foo":"bar"}}`,
			valid:   true,
		},
		{
			name:    "yaml parsing succeeds even if provider type isn't recognized",
			content: "name: cmd\ndescription: the cmd\ninvalid:\n  foo: bar",
			valid:   true,
		},
		{
			name:    "fails on empty content",
			content: ``,
			valid:   false,
		},
		{
			name:    "fails on invalid JSON",
			content: `{"name":"cmd","description:"the cmd"`,
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
			s, err := provider.NewParameter([]byte(test.content))
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

func TestParameter_Validate(t *testing.T) {
	tests := []struct {
		name  string
		param provider.Parameter
		valid bool
	}{
		{
			name: "valid bool",
			param: provider.Parameter{
				Name:        "param",
				Description: "the param",
				Type:        provider.BoolParamType,
			},
			valid: true,
		},
		{
			name: "valid int",
			param: provider.Parameter{
				Name:        "param",
				Description: "the param",
				Type:        provider.IntParamType,
			},
			valid: true,
		},
		{
			name: "valid number",
			param: provider.Parameter{
				Name:        "param",
				Description: "the param",
				Type:        provider.NumberParamType,
			},
			valid: true,
		},
		{
			name: "valid string",
			param: provider.Parameter{
				Name:        "param",
				Description: "the param",
				Type:        provider.StringParamType,
			},
			valid: true,
		},
		{
			name:  "invalid, missing name",
			param: provider.Parameter{Description: "the param", Type: provider.StringParamType},
			valid: false,
		},
		{
			name:  "valid, missing description",
			param: provider.Parameter{Name: "param", Type: provider.StringParamType},
			valid: true,
		},
		{
			name:  "invalid, missing type",
			param: provider.Parameter{Name: "param", Description: "the param"},
			valid: false,
		},
		{
			name:  "invalid, required but default specified",
			param: provider.Parameter{Name: "param", Description: "the param", Type: "string", Required: true, Default: "value"},
			valid: false,
		},
		{
			name: "invalid parameter type",
			param: provider.Parameter{
				Name:        "param",
				Description: "the param",
				Type:        "invalid",
			},
			valid: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.param.Validate()
			if test.valid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestNewInvalidParameterSpecError(t *testing.T) {
	err := provider.NewInvalidParameterSpecError("the reason")
	assert.EqualError(t, err, "invalid parameter spec: the reason")
}
