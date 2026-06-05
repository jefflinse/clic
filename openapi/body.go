package openapi

import (
	"fmt"
	"maps"
	"sort"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/jefflinse/clic/form"
)

// requestBodySchema returns the JSON schema of an operation's request body,
// preferring application/json and falling back to any JSON-flavored media type.
// It returns nil when there is no usable JSON body schema.
func requestBodySchema(rb *openapi3.RequestBodyRef) *openapi3.Schema {
	if rb == nil || rb.Value == nil {
		return nil
	}

	if mt := rb.Value.Content.Get("application/json"); mt != nil {
		return deref(mt.Schema)
	}
	for mime, mt := range rb.Value.Content {
		if strings.Contains(mime, "json") && mt.Schema != nil {
			return deref(mt.Schema)
		}
	}
	return nil
}

// BodyFields maps an OpenAPI request-body schema into a UI-agnostic form.Field
// tree. It is a pure transformation: feed it a parsed schema and it yields the
// fields a renderer (a huh form today, a richer TUI later) can present.
//
// An object schema expands into one field per property, in alphabetical order
// (OpenAPI property order is not preserved by the parser). A non-object schema
// is represented as a single field named "body".
func BodyFields(schema *openapi3.Schema) []form.Field {
	schema = mergeAllOf(schema)
	if schema == nil {
		return nil
	}

	if isObject(schema) {
		return objectFields(schema)
	}

	return []form.Field{fieldFrom("body", schema, true)}
}

// objectFields builds a field for each property of an object schema, marking
// those listed in the schema's required set.
func objectFields(schema *openapi3.Schema) []form.Field {
	required := map[string]bool{}
	for _, name := range schema.Required {
		required[name] = true
	}

	names := make([]string, 0, len(schema.Properties))
	for name := range schema.Properties {
		names = append(names, name)
	}
	sort.Strings(names)

	fields := make([]form.Field, 0, len(names))
	for _, name := range names {
		prop := deref(schema.Properties[name])
		if prop == nil {
			continue
		}
		fields = append(fields, fieldFrom(name, prop, required[name]))
	}
	return fields
}

// fieldFrom builds a single field from a schema, recursing into nested objects
// and array element types.
func fieldFrom(name string, schema *openapi3.Schema, required bool) form.Field {
	schema = mergeAllOf(schema)

	field := form.Field{
		Name:        name,
		Title:       schema.Title,
		Description: firstLine(schema.Description),
		Required:    required,
		Default:     schema.Default,
		Format:      schema.Format,
	}

	switch {
	case len(schema.Enum) > 0:
		field.Type = form.EnumField
		field.Enum = enumStrings(schema.Enum)
	case isObject(schema):
		field.Type = form.ObjectField
		field.Fields = objectFields(schema)
	case isArray(schema):
		field.Type = form.ArrayField
		if item := deref(schema.Items); item != nil {
			elem := fieldFrom(name, item, false)
			field.Item = &elem
		}
	case has(schema, "integer"):
		field.Type = form.IntegerField
	case has(schema, "number"):
		field.Type = form.NumberField
	case has(schema, "boolean"):
		field.Type = form.BooleanField
	default:
		field.Type = form.StringField
	}

	return field
}

// mergeAllOf folds an allOf composition into a single schema by unioning the
// properties and required sets of its subschemas. Other compositions (oneOf,
// anyOf) are left untyped and fall through to a free-form string field.
func mergeAllOf(schema *openapi3.Schema) *openapi3.Schema {
	if schema == nil || len(schema.AllOf) == 0 {
		return schema
	}

	merged := *schema
	merged.AllOf = nil

	props := openapi3.Schemas{}
	maps.Copy(props, schema.Properties)
	required := append([]string{}, schema.Required...)

	for _, ref := range schema.AllOf {
		sub := mergeAllOf(deref(ref))
		if sub == nil {
			continue
		}
		maps.Copy(props, sub.Properties)
		required = append(required, sub.Required...)
		if merged.Type == nil {
			merged.Type = sub.Type
		}
	}

	merged.Properties = props
	merged.Required = required
	return &merged
}

// enumStrings renders a schema's enum values as display strings.
func enumStrings(values []any) []string {
	out := make([]string, 0, len(values))
	for _, v := range values {
		if v == nil {
			continue
		}
		out = append(out, fmt.Sprintf("%v", v))
	}
	return out
}

// deref returns the schema a ref points to, or nil.
func deref(ref *openapi3.SchemaRef) *openapi3.Schema {
	if ref == nil {
		return nil
	}
	return ref.Value
}

// has reports whether a schema declares the given primitive type.
func has(schema *openapi3.Schema, t string) bool {
	return schema.Type != nil && schema.Type.Is(t)
}

// isObject reports whether a schema describes an object. A schema with no
// explicit type but with declared properties is treated as an object.
func isObject(schema *openapi3.Schema) bool {
	return has(schema, "object") || (schema.Type == nil && len(schema.Properties) > 0)
}

// isArray reports whether a schema describes an array. A schema with no
// explicit type but with an items schema is treated as an array.
func isArray(schema *openapi3.Schema) bool {
	return has(schema, "array") || (schema.Type == nil && schema.Items != nil)
}
