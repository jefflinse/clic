// Package form defines a declarative, UI-agnostic description of an input form.
//
// A Field tree is the intermediate representation between a schema (e.g. an
// OpenAPI request body) and a concrete renderer. It deliberately depends on
// nothing else in clic — no OpenAPI types, no terminal/TUI library — so the
// same description can drive a simple huh form today and a richer bubbletea
// experience later, and so it can be serialized as part of a compiled spec.
package form

// FieldType identifies the kind of input a Field represents.
type FieldType string

const (
	// StringField is free-form text.
	StringField FieldType = "string"

	// IntegerField is a whole number.
	IntegerField FieldType = "integer"

	// NumberField is a decimal number.
	NumberField FieldType = "number"

	// BooleanField is a true/false toggle.
	BooleanField FieldType = "boolean"

	// EnumField is a choice among a fixed set of values (see Field.Enum).
	EnumField FieldType = "enum"

	// ObjectField is a nested group of fields (see Field.Fields).
	ObjectField FieldType = "object"

	// ArrayField is a repeated value of a single element type (see Field.Item).
	ArrayField FieldType = "array"
)

// A Field describes a single input within a form. Fields nest: an ObjectField
// carries child Fields, and an ArrayField carries an Item describing its
// element type.
type Field struct {
	// Name is the property key this field maps to in the assembled payload.
	Name string `json:"name" yaml:"name"`

	// Title is the human-facing label. When empty, renderers fall back to Name.
	Title string `json:"title,omitempty" yaml:"title,omitempty"`

	// Description is optional help text shown alongside the input.
	Description string `json:"description,omitempty" yaml:"description,omitempty"`

	// Type is the kind of input this field represents.
	Type FieldType `json:"type" yaml:"type"`

	// Required reports whether a value must be provided.
	Required bool `json:"required,omitempty" yaml:"required,omitempty"`

	// Default is the value to pre-populate, if any.
	Default any `json:"default,omitempty" yaml:"default,omitempty"`

	// Format is an optional semantic hint from the source schema (for example
	// "email", "date-time", or "uuid") that a renderer may use for validation.
	Format string `json:"format,omitempty" yaml:"format,omitempty"`

	// Enum lists the allowed values when Type is EnumField.
	Enum []string `json:"enum,omitempty" yaml:"enum,omitempty"`

	// Fields are the child fields when Type is ObjectField.
	Fields []Field `json:"fields,omitempty" yaml:"fields,omitempty"`

	// Item describes the element type when Type is ArrayField.
	Item *Field `json:"item,omitempty" yaml:"item,omitempty"`
}

// Label returns the human-facing label for the field, preferring Title and
// falling back to Name.
func (f Field) Label() string {
	if f.Title != "" {
		return f.Title
	}
	return f.Name
}
