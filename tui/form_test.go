package tui

import (
	"testing"

	"github.com/jefflinse/clic/form"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// set finds the binding for a named field and stores a raw string value, as huh
// would after the user types into the input.
func set(bindings []*binding, name, value string) {
	for _, b := range bindings {
		if b.field.Name == name {
			b.str = value
		}
	}
}

func TestAssemble_TypedScalars(t *testing.T) {
	bindings := newBindings([]form.Field{
		{Name: "name", Type: form.StringField, Required: true},
		{Name: "age", Type: form.IntegerField},
		{Name: "weight", Type: form.NumberField},
	})
	set(bindings, "name", "rex")
	set(bindings, "age", "3")
	set(bindings, "weight", "12.5")

	body := assemble(bindings)
	assert.Equal(t, "rex", body["name"])
	assert.Equal(t, 3, body["age"])
	assert.Equal(t, 12.5, body["weight"])
}

func TestAssemble_OmitsEmptyOptional(t *testing.T) {
	bindings := newBindings([]form.Field{
		{Name: "name", Type: form.StringField, Required: true},
		{Name: "nickname", Type: form.StringField},
	})
	set(bindings, "name", "rex")

	body := assemble(bindings)
	assert.Contains(t, body, "name")
	assert.NotContains(t, body, "nickname", "empty optional fields should be omitted")
}

func TestAssemble_KeepsEmptyRequired(t *testing.T) {
	// a required field left empty is still included (the form's validation is
	// what prevents submission; assembly should not silently drop it)
	bindings := newBindings([]form.Field{
		{Name: "name", Type: form.StringField, Required: true},
	})

	body := assemble(bindings)
	assert.Contains(t, body, "name")
	assert.Equal(t, "", body["name"])
}

func TestAssemble_Boolean(t *testing.T) {
	bindings := newBindings([]form.Field{
		{Name: "vaccinated", Type: form.BooleanField},
	})
	bindings[0].boolean = true

	body := assemble(bindings)
	assert.Equal(t, true, body["vaccinated"])
}

func TestAssemble_NestedObject(t *testing.T) {
	bindings := newBindings([]form.Field{
		{
			Name: "owner",
			Type: form.ObjectField,
			Fields: []form.Field{
				{Name: "email", Type: form.StringField, Required: true},
				{Name: "phone", Type: form.StringField},
			},
		},
	})
	bindings[0].children[0].str = "a@b.com"

	body := assemble(bindings)
	owner, ok := body["owner"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "a@b.com", owner["email"])
	assert.NotContains(t, owner, "phone")
}

func TestAssemble_OmitsEmptyNestedObject(t *testing.T) {
	bindings := newBindings([]form.Field{
		{
			Name: "owner",
			Type: form.ObjectField,
			Fields: []form.Field{
				{Name: "email", Type: form.StringField},
			},
		},
	})

	body := assemble(bindings)
	assert.NotContains(t, body, "owner", "an object with only empty optionals is omitted")
}

func TestAssemble_TypedArray(t *testing.T) {
	bindings := newBindings([]form.Field{
		{
			Name: "scores",
			Type: form.ArrayField,
			Item: &form.Field{Type: form.IntegerField},
		},
	})
	set(bindings, "scores", "10\n20\n\n30")

	body := assemble(bindings)
	assert.Equal(t, []any{10, 20, 30}, body["scores"])
}

func TestNewBinding_AppliesStringDefault(t *testing.T) {
	bindings := newBindings([]form.Field{
		{Name: "status", Type: form.StringField, Default: "available"},
	})
	assert.Equal(t, "available", assemble(bindings)["status"])
}
