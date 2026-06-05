package openapi_test

import (
	"encoding/json"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/goccy/go-yaml"
	"github.com/jefflinse/clic/form"
	"github.com/jefflinse/clic/openapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// schemaFromYAML parses a single OpenAPI schema document for testing the mapper
// in isolation from the rest of the compiler.
func schemaFromYAML(t *testing.T, doc string) *openapi3.Schema {
	t.Helper()
	var raw any
	require.NoError(t, yaml.Unmarshal([]byte(doc), &raw))
	data, err := json.Marshal(raw)
	require.NoError(t, err)
	s := &openapi3.Schema{}
	require.NoError(t, s.UnmarshalJSON(data))
	return s
}

func fieldByName(fields []form.Field, name string) (form.Field, bool) {
	for _, f := range fields {
		if f.Name == name {
			return f, true
		}
	}
	return form.Field{}, false
}

func TestBodyFields_NilSchema(t *testing.T) {
	assert.Nil(t, openapi.BodyFields(nil))
}

func TestBodyFields_ScalarTypes(t *testing.T) {
	schema := schemaFromYAML(t, `
type: object
properties:
  name: {type: string}
  age: {type: integer}
  weight: {type: number}
  good: {type: boolean}
`)

	fields := openapi.BodyFields(schema)
	require.Len(t, fields, 4)

	// properties are emitted in alphabetical order
	assert.Equal(t, []string{"age", "good", "name", "weight"}, names(fields))

	byType := map[string]form.FieldType{}
	for _, f := range fields {
		byType[f.Name] = f.Type
	}
	assert.Equal(t, form.StringField, byType["name"])
	assert.Equal(t, form.IntegerField, byType["age"])
	assert.Equal(t, form.NumberField, byType["weight"])
	assert.Equal(t, form.BooleanField, byType["good"])
}

func TestBodyFields_RequiredAndMetadata(t *testing.T) {
	schema := schemaFromYAML(t, `
type: object
required: [name]
properties:
  name:
    type: string
    title: Full Name
    description: |-
      the pet's name
      ignored second line
    format: email
    default: rex
`)

	fields := openapi.BodyFields(schema)
	name, ok := fieldByName(fields, "name")
	require.True(t, ok)

	assert.True(t, name.Required)
	assert.Equal(t, "Full Name", name.Title)
	assert.Equal(t, "Full Name", name.Label())
	assert.Equal(t, "the pet's name", name.Description)
	assert.Equal(t, "email", name.Format)
	assert.Equal(t, "rex", name.Default)
}

func TestBodyFields_Enum(t *testing.T) {
	schema := schemaFromYAML(t, `
type: object
properties:
  status:
    type: string
    enum: [available, pending, sold]
`)

	status, ok := fieldByName(openapi.BodyFields(schema), "status")
	require.True(t, ok)
	assert.Equal(t, form.EnumField, status.Type)
	assert.Equal(t, []string{"available", "pending", "sold"}, status.Enum)
}

func TestBodyFields_NestedObject(t *testing.T) {
	schema := schemaFromYAML(t, `
type: object
properties:
  owner:
    type: object
    required: [email]
    properties:
      email: {type: string}
      phone: {type: string}
`)

	owner, ok := fieldByName(openapi.BodyFields(schema), "owner")
	require.True(t, ok)
	assert.Equal(t, form.ObjectField, owner.Type)
	require.Len(t, owner.Fields, 2)

	email, ok := fieldByName(owner.Fields, "email")
	require.True(t, ok)
	assert.True(t, email.Required)
	assert.Equal(t, form.StringField, email.Type)
}

func TestBodyFields_Array(t *testing.T) {
	schema := schemaFromYAML(t, `
type: object
properties:
  tags:
    type: array
    items: {type: string}
`)

	tags, ok := fieldByName(openapi.BodyFields(schema), "tags")
	require.True(t, ok)
	assert.Equal(t, form.ArrayField, tags.Type)
	require.NotNil(t, tags.Item)
	assert.Equal(t, form.StringField, tags.Item.Type)
}

func TestBodyFields_AllOfMerge(t *testing.T) {
	schema := schemaFromYAML(t, `
allOf:
  - type: object
    required: [id]
    properties:
      id: {type: integer}
  - type: object
    properties:
      name: {type: string}
`)

	fields := openapi.BodyFields(schema)
	assert.Equal(t, []string{"id", "name"}, names(fields))

	id, ok := fieldByName(fields, "id")
	require.True(t, ok)
	assert.True(t, id.Required)
	assert.Equal(t, form.IntegerField, id.Type)
}

func TestBodyFields_NonObjectBody(t *testing.T) {
	// a bare array body collapses to a single "body" field
	schema := schemaFromYAML(t, `
type: array
items: {type: string}
`)

	fields := openapi.BodyFields(schema)
	require.Len(t, fields, 1)
	assert.Equal(t, "body", fields[0].Name)
	assert.Equal(t, form.ArrayField, fields[0].Type)
}

func TestCompile_CarriesBodyFields(t *testing.T) {
	doc := `
openapi: 3.0.0
info: {title: Pets}
paths:
  /pets:
    post:
      summary: create a pet
      requestBody:
        content:
          application/json:
            schema:
              type: object
              required: [name]
              properties:
                name: {type: string}
                age: {type: integer}
`
	app, err := openapi.Compile([]byte(doc))
	require.NoError(t, err)

	pets := find(app.Commands, "pets")
	create := restOf(t, find(pets.Subcommands, "create"))

	assert.True(t, create.RawBody)
	require.Len(t, create.Body, 2)
	assert.Equal(t, []string{"age", "name"}, names(create.Body))

	name, ok := fieldByName(create.Body, "name")
	require.True(t, ok)
	assert.True(t, name.Required)
	assert.Equal(t, form.StringField, name.Type)
}

func names(fields []form.Field) []string {
	out := make([]string, len(fields))
	for i, f := range fields {
		out[i] = f.Name
	}
	return out
}
