package spec_test

import (
	"testing"

	"github.com/jefflinse/handyman/spec"
	"github.com/stretchr/testify/assert"
)

func TestNewParameterSpec(t *testing.T) {
	tests := []struct {
		name  string
		json  string
		valid bool
	}{
		{
			name:  "valid parameter, required not specified",
			json:  `{"name":"param","description":"the param","type":"string"}`,
			valid: true,
		},
		{
			name:  "valid parameter, required is specified",
			json:  `{"name":"param","description":"the param","type":"string","required":true}`,
			valid: true,
		},
		{
			name:  "invalid JSON, required specified as string",
			json:  `{"name":"param","description:"the param","type":"string","required":"true"}`,
			valid: false,
		},
		{
			name:  "invalid JSON, malformed",
			json:  `{"name":"param","description:"the param","type":"string"`,
			valid: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			s, err := spec.NewParamterSpec([]byte(test.json))
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
		param spec.Parameter
		valid bool
	}{
		{
			name: "valid bool",
			param: spec.Parameter{
				Name:        "param",
				Description: "the param",
				Type:        spec.BoolParamType,
			},
			valid: true,
		},
		{
			name: "valid int",
			param: spec.Parameter{
				Name:        "param",
				Description: "the param",
				Type:        spec.IntParamType,
			},
			valid: true,
		},
		{
			name: "valid number",
			param: spec.Parameter{
				Name:        "param",
				Description: "the param",
				Type:        spec.NumberParamType,
			},
			valid: true,
		},
		{
			name: "valid string",
			param: spec.Parameter{
				Name:        "param",
				Description: "the param",
				Type:        spec.StringParamType,
			},
			valid: true,
		},
		{
			name:  "invalid, missing name",
			param: spec.Parameter{Description: "the param", Type: spec.StringParamType},
			valid: false,
		},
		{
			name:  "invalid, missing description",
			param: spec.Parameter{Name: "param", Type: spec.StringParamType},
			valid: false,
		},
		{
			name:  "invalid, missing type",
			param: spec.Parameter{Name: "param", Description: "the param"},
			valid: false,
		},
		{
			name: "invalid parameter type",
			param: spec.Parameter{
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
	err := spec.NewInvalidParameterSpecError("the reason")
	assert.EqualError(t, err, "invalid parameter spec: the reason")
}
