package spec

import (
	"github.com/jefflinse/clic/ioutil"
)

// Format identifies the kind of spec a document represents.
type Format int

const (
	// FormatUnknown indicates the spec format could not be determined.
	FormatUnknown Format = iota

	// FormatClic indicates a native clic spec.
	FormatClic

	// FormatOpenAPI indicates an OpenAPI (or Swagger) spec.
	FormatOpenAPI
)

// String returns a human-readable name for the format.
func (f Format) String() string {
	switch f {
	case FormatClic:
		return "clic"
	case FormatOpenAPI:
		return "openapi"
	default:
		return "unknown"
	}
}

// DetectFormat inspects spec content and reports whether it is an OpenAPI
// document or a native clic spec. OpenAPI is identified by a top-level
// "openapi" or "swagger" key; a clic spec by its "commands" or "name" keys.
func DetectFormat(data []byte) Format {
	probe := map[string]any{}
	if err := ioutil.Unmarshal(data, &probe); err != nil {
		return FormatUnknown
	}

	if _, ok := probe["openapi"]; ok {
		return FormatOpenAPI
	}
	if _, ok := probe["swagger"]; ok {
		return FormatOpenAPI
	}
	if _, ok := probe["commands"]; ok {
		return FormatClic
	}
	if _, ok := probe["name"]; ok {
		return FormatClic
	}

	return FormatUnknown
}
