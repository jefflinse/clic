package spec_test

import (
	"testing"

	"github.com/jefflinse/clic/spec"
	"github.com/stretchr/testify/assert"
)

func TestDetectFormat(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    spec.Format
	}{
		{
			name:    "openapi 3.x JSON",
			content: `{"openapi":"3.0.0","info":{"title":"x"}}`,
			want:    spec.FormatOpenAPI,
		},
		{
			name:    "openapi 3.x YAML",
			content: "openapi: 3.1.0\ninfo:\n  title: x",
			want:    spec.FormatOpenAPI,
		},
		{
			name:    "swagger 2.0 detected as openapi",
			content: `{"swagger":"2.0"}`,
			want:    spec.FormatOpenAPI,
		},
		{
			name:    "clic spec with commands",
			content: `{"name":"app","description":"d","commands":[]}`,
			want:    spec.FormatClic,
		},
		{
			name:    "clic spec YAML with name only",
			content: "name: app\ndescription: d",
			want:    spec.FormatClic,
		},
		{
			name:    "unrecognized",
			content: `{"foo":"bar"}`,
			want:    spec.FormatUnknown,
		},
		{
			name:    "garbage",
			content: `not valid at all: : :`,
			want:    spec.FormatUnknown,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, spec.DetectFormat([]byte(test.content)))
		})
	}
}
