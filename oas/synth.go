package oas

import "github.com/getkin/kin-openapi/openapi3"

// maxSynthDepth bounds recursion so a self-referential schema cannot loop
// forever during synthesis.
const maxSynthDepth = 12

// Synthesize builds an example value for a schema, suitable for a mock response
// body. It prefers an explicit example, default, or first enum value; otherwise
// it constructs a value from the schema's type, recursing into object
// properties and array items. It never panics on an absent or malformed schema,
// returning nil instead.
func Synthesize(schema *openapi3.SchemaRef) any {
	return synth(schema, 0)
}

func synth(ref *openapi3.SchemaRef, depth int) any {
	if ref == nil || ref.Value == nil || depth > maxSynthDepth {
		return nil
	}
	s := ref.Value

	// honor explicit example/default/enum first
	if s.Example != nil {
		return s.Example
	}
	if s.Default != nil {
		return s.Default
	}
	if len(s.Enum) > 0 {
		return s.Enum[0]
	}

	switch {
	case s.Type.Is("object") || (s.Type.IsEmpty() && len(s.Properties) > 0):
		obj := map[string]any{}
		for name, prop := range s.Properties {
			obj[name] = synth(prop, depth+1)
		}
		return obj

	case s.Type.Is("array"):
		return []any{synth(s.Items, depth+1)}

	case s.Type.Is("integer"):
		return 0
	case s.Type.Is("number"):
		return 0
	case s.Type.Is("boolean"):
		return false

	case s.Type.Is("string"):
		return sampleString(s.Format)

	default:
		// unknown/empty type with no properties: emptiest sensible value
		return nil
	}
}

// sampleString returns a representative value for a string schema, honoring
// common formats so synthesized bodies pass their own format validation.
func sampleString(format string) string {
	switch format {
	case "date-time":
		return "2026-01-01T00:00:00Z"
	case "date":
		return "2026-01-01"
	case "email":
		return "user@example.com"
	case "uuid":
		return "00000000-0000-0000-0000-000000000000"
	case "uri", "url":
		return "https://example.com"
	case "byte":
		return "ZXhhbXBsZQ=="
	case "hostname":
		return "example.com"
	case "ipv4":
		return "127.0.0.1"
	case "ipv6":
		return "::1"
	default:
		return "string"
	}
}
