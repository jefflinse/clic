package lambda_test

import (
	"testing"

	"github.com/jefflinse/handyman/provider/lambda"
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
			s, err := lambda.NewParameter([]byte(test.json))
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
		param lambda.Parameter
		valid bool
	}{
		{
			name: "valid bool",
			param: lambda.Parameter{
				Name:        "param",
				Description: "the param",
				Type:        lambda.BoolParamType,
			},
			valid: true,
		},
		{
			name: "valid int",
			param: lambda.Parameter{
				Name:        "param",
				Description: "the param",
				Type:        lambda.IntParamType,
			},
			valid: true,
		},
		{
			name: "valid number",
			param: lambda.Parameter{
				Name:        "param",
				Description: "the param",
				Type:        lambda.NumberParamType,
			},
			valid: true,
		},
		{
			name: "valid string",
			param: lambda.Parameter{
				Name:        "param",
				Description: "the param",
				Type:        lambda.StringParamType,
			},
			valid: true,
		},
		{
			name:  "invalid, missing name",
			param: lambda.Parameter{Description: "the param", Type: lambda.StringParamType},
			valid: false,
		},
		{
			name:  "invalid, missing description",
			param: lambda.Parameter{Name: "param", Type: lambda.StringParamType},
			valid: false,
		},
		{
			name:  "invalid, missing type",
			param: lambda.Parameter{Name: "param", Description: "the param"},
			valid: false,
		},
		{
			name: "invalid parameter type",
			param: lambda.Parameter{
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
	err := lambda.NewInvalidParameterSpecError("the reason")
	assert.EqualError(t, err, "invalid parameter spec: the reason")
}
