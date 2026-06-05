package openapi_test

import (
	"testing"

	"github.com/jefflinse/clic/openapi"
	"github.com/jefflinse/clic/provider"
	"github.com/jefflinse/clic/provider/rest"
	"github.com/jefflinse/clic/spec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const petstore = `
openapi: 3.0.0
info:
  title: Pet Store
  description: |-
    A sample API.
    Second line ignored.
servers:
  - url: https://api.example.com/v1
components:
  securitySchemes:
    bearerAuth:
      type: http
      scheme: bearer
paths:
  /pets:
    get:
      summary: list pets
      parameters:
        - name: limit
          in: query
          schema:
            type: integer
    post:
      summary: create a pet
      requestBody:
        content:
          application/json:
            schema:
              type: object
  /pets/{id}:
    get:
      summary: get a pet
      parameters:
        - name: id
          in: path
          required: true
          schema: {type: string}
    put:
      summary: replace a pet
      parameters:
        - {name: id, in: path, required: true, schema: {type: string}}
      requestBody:
        content: {application/json: {schema: {type: object}}}
    patch:
      summary: update a pet
      parameters:
        - {name: id, in: path, required: true, schema: {type: string}}
    delete:
      summary: delete a pet
      parameters:
        - {name: id, in: path, required: true, schema: {type: string}}
  /pets/{id}/vaccinate:
    post:
      summary: vaccinate a pet
      parameters:
        - {name: id, in: path, required: true, schema: {type: string}}
  /users/{id}/posts:
    get:
      summary: list a user's posts
      parameters:
        - {name: id, in: path, required: true, schema: {type: string}}
    post:
      summary: create a post for a user
      parameters:
        - {name: id, in: path, required: true, schema: {type: string}}
      requestBody:
        content: {application/json: {schema: {type: object}}}
`

func find(cmds []*spec.Command, name string) *spec.Command {
	for _, c := range cmds {
		if c.Name == name {
			return c
		}
	}
	return nil
}

func restOf(t *testing.T, cmd *spec.Command) *rest.Spec {
	t.Helper()
	require.NotNil(t, cmd)
	require.NotNil(t, cmd.Provider)
	s, ok := cmd.Provider.(*rest.Spec)
	require.True(t, ok, "provider should be *rest.Spec")
	return s
}

func TestCompile_AppMetadata(t *testing.T) {
	app, err := openapi.Compile([]byte(petstore))
	require.NoError(t, err)
	require.NoError(t, app.Validate())

	assert.Equal(t, "pet-store", app.Name)
	assert.Equal(t, "A sample API.", app.Description)
	assert.Equal(t, "https://api.example.com/v1", app.Server)
	require.NotNil(t, app.Auth)
	assert.Equal(t, provider.AuthBearer, app.Auth.Type)
}

func TestCompile_CRUDVerbs(t *testing.T) {
	app, err := openapi.Compile([]byte(petstore))
	require.NoError(t, err)

	pets := find(app.Commands, "pets")
	require.NotNil(t, pets)

	verbs := []string{}
	for _, c := range pets.Subcommands {
		if c.Provider != nil {
			verbs = append(verbs, c.Name)
		}
	}
	// list/create on the collection; get/replace/update/delete on the item; vaccinate action
	assert.ElementsMatch(t, []string{"list", "create", "get", "replace", "update", "delete", "vaccinate"}, verbs)
}

func TestCompile_ItemParamsArePositional(t *testing.T) {
	app, err := openapi.Compile([]byte(petstore))
	require.NoError(t, err)

	pets := find(app.Commands, "pets")
	get := restOf(t, find(pets.Subcommands, "get"))

	assert.Equal(t, "GET", get.Method)
	assert.Equal(t, "/pets/{id}", get.Endpoint)
	assert.Equal(t, "https://api.example.com/v1", get.BaseURL)
	require.Len(t, get.PathParams, 1)
	assert.Equal(t, "id", get.PathParams[0].Name)
	assert.True(t, get.PathParams[0].Required)
	assert.Empty(t, get.QueryParams)
}

func TestCompile_QueryParamsAreFlags(t *testing.T) {
	app, err := openapi.Compile([]byte(petstore))
	require.NoError(t, err)

	pets := find(app.Commands, "pets")
	list := restOf(t, find(pets.Subcommands, "list"))

	require.Len(t, list.QueryParams, 1)
	assert.Equal(t, "limit", list.QueryParams[0].Name)
	assert.Equal(t, provider.IntParamType, list.QueryParams[0].Type)
	assert.False(t, list.QueryParams[0].Required)
	assert.Empty(t, list.PathParams)
}

func TestCompile_RequestBodyEnablesRawBody(t *testing.T) {
	app, err := openapi.Compile([]byte(petstore))
	require.NoError(t, err)

	pets := find(app.Commands, "pets")
	assert.True(t, restOf(t, find(pets.Subcommands, "create")).RawBody)
	assert.False(t, restOf(t, find(pets.Subcommands, "get")).RawBody)
}

func TestCompile_PutPatchCollision(t *testing.T) {
	app, err := openapi.Compile([]byte(petstore))
	require.NoError(t, err)

	pets := find(app.Commands, "pets")
	// PATCH -> update, PUT -> replace (because both exist)
	assert.Equal(t, "PATCH", restOf(t, find(pets.Subcommands, "update")).Method)
	assert.Equal(t, "PUT", restOf(t, find(pets.Subcommands, "replace")).Method)
}

func TestCompile_ActionVerb(t *testing.T) {
	app, err := openapi.Compile([]byte(petstore))
	require.NoError(t, err)

	pets := find(app.Commands, "pets")
	vaccinate := restOf(t, find(pets.Subcommands, "vaccinate"))
	assert.Equal(t, "POST", vaccinate.Method)
	assert.Equal(t, "/pets/{id}/vaccinate", vaccinate.Endpoint)
	require.Len(t, vaccinate.PathParams, 1)
}

func TestCompile_NestedResource(t *testing.T) {
	app, err := openapi.Compile([]byte(petstore))
	require.NoError(t, err)

	users := find(app.Commands, "users")
	require.NotNil(t, users)
	posts := find(users.Subcommands, "posts")
	require.NotNil(t, posts)

	// posts has 2 methods + no children -> treated as a sub-resource, not an action
	list := restOf(t, find(posts.Subcommands, "list"))
	create := restOf(t, find(posts.Subcommands, "create"))
	assert.Equal(t, "GET", list.Method)
	assert.Equal(t, "POST", create.Method)
	require.Len(t, list.PathParams, 1)
	assert.Equal(t, "id", list.PathParams[0].Name)
}

func TestCompile_RejectsSwagger2(t *testing.T) {
	_, err := openapi.Compile([]byte(`{"swagger":"2.0","info":{"title":"x"}}`))
	assert.Error(t, err)
}

func TestCompile_RootPath(t *testing.T) {
	// a "/" path produces zero segments; it must not panic
	doc := `
openapi: 3.0.0
info: {title: Root API}
paths:
  /:
    get:
      summary: api root
`
	app, err := openapi.Compile([]byte(doc))
	require.NoError(t, err)
	require.NoError(t, app.Validate())
	require.NotNil(t, find(app.Commands, "list"))
}

func TestCompile_NameCollision(t *testing.T) {
	// POST /pet and POST /pet/{id} both map to "create"; names must be unique
	doc := `
openapi: 3.0.0
info: {title: Pets}
paths:
  /pet:
    post:
      summary: add a pet
  /pet/{id}:
    post:
      summary: update a pet via form
      parameters:
        - {name: id, in: path, required: true, schema: {type: string}}
`
	app, err := openapi.Compile([]byte(doc))
	require.NoError(t, err)
	require.NoError(t, app.Validate())

	pet := find(app.Commands, "pet")
	require.NotNil(t, pet)

	names := map[string]int{}
	for _, c := range pet.Subcommands {
		names[c.Name]++
	}
	for name, count := range names {
		assert.Equal(t, 1, count, "command %q should be unique", name)
	}
	assert.Len(t, pet.Subcommands, 2)
}
