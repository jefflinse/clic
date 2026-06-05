package openapi_test

import (
	"github.com/jefflinse/clic/openapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestCompile_OpenAPI31(t *testing.T) {
	doc := `
openapi: 3.1.0
info:
  title: V31 API
  version: "1.0"
servers:
  - url: https://api.example.com
paths:
  /things:
    get:
      summary: list things
      parameters:
        - name: q
          in: query
          schema:
            type: string
`
	app, err := openapi.Compile([]byte(doc))
	require.NoError(t, err)
	require.NoError(t, app.Validate())
	assert.Equal(t, "v31-api", app.Name)
	things := find(app.Commands, "things")
	require.NotNil(t, things)
	require.NotNil(t, find(things.Subcommands, "list"))
}
