package ioutil_test

import (
	"testing"

	"github.com/jefflinse/handyman/ioutil"
	"github.com/stretchr/testify/assert"
)

type foo struct {
	Foo string `json:"foo" yaml:"foo"`
}

func TestIntermarshal(t *testing.T) {
	tests := []struct {
		name          string
		source        interface{}
		target        interface{}
		expectSuccess bool
	}{
		{
			name:          "returns nil when successfully marshalling and unmarshalling",
			source:        map[string]interface{}{"foo": "value"},
			target:        foo{},
			expectSuccess: true,
		},
		{
			name:          "returns error when failing to marshal source",
			source:        make(chan int),
			target:        foo{},
			expectSuccess: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ioutil.Intermarshal(test.source, test.target)
			if test.expectSuccess {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestUnmarshal(t *testing.T) {
	tests := []struct {
		name          string
		content       string
		target        foo
		expectSuccess bool
	}{
		{
			name:          "can unmarshal valid JSON",
			content:       `{"foo": "value"}`,
			target:        foo{},
			expectSuccess: true,
		},
		{
			name:          "can unmarshal valid YAML",
			content:       `foo: value`,
			target:        foo{},
			expectSuccess: true,
		},
		{
			name:          "fails with invalid JSON",
			content:       `{"foo": "value"`,
			target:        foo{},
			expectSuccess: false,
		},
		{
			name:          "fails with invalid YAML",
			content:       `foo`,
			target:        foo{},
			expectSuccess: false,
		},
		{
			name:          "fails if data is empty",
			content:       ``,
			target:        foo{},
			expectSuccess: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ioutil.Unmarshal([]byte(test.content), &test.target)
			if test.expectSuccess {
				assert.NoError(t, err)
				assert.Equal(t, "value", test.target.Foo)
			} else {
				assert.Error(t, err)
			}
		})
	}
}
