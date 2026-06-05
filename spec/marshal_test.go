package spec_test

import (
	"encoding/json"
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/jefflinse/clic/spec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const roundTripSpec = `name: petstore
description: pet store
server: https://api.example.com/v1
auth:
  type: bearer
commands:
  - name: pets
    description: manage pets
    subcommands:
      - name: get
        description: get a pet by id
        rest:
          base_url: https://api.example.com/v1
          endpoint: /pets/{id}
          method: GET
          path_params:
            - name: id
              type: string
              required: true
`

func TestApp_RoundTrip(t *testing.T) {
	app, err := spec.NewAppSpec([]byte(roundTripSpec))
	require.NoError(t, err)
	require.NoError(t, app.Validate())

	assertReparsed := func(t *testing.T, data []byte) {
		t.Helper()
		got, err := spec.NewAppSpec(data)
		require.NoError(t, err)
		require.NoError(t, got.Validate())

		assert.Equal(t, "petstore", got.Name)
		assert.Equal(t, "https://api.example.com/v1", got.Server)
		require.NotNil(t, got.Auth)
		assert.Equal(t, "bearer", got.Auth.Type)

		require.Len(t, got.Commands, 1)
		pets := got.Commands[0]
		assert.Equal(t, "pets", pets.Name)
		require.Len(t, pets.Subcommands, 1)

		get := pets.Subcommands[0]
		require.NotNil(t, get.Provider)
		assert.Equal(t, "rest", get.Provider.Type())
	}

	t.Run("yaml", func(t *testing.T) {
		data, err := yaml.Marshal(app)
		require.NoError(t, err)
		assertReparsed(t, data)
	})

	t.Run("json", func(t *testing.T) {
		data, err := json.Marshal(app)
		require.NoError(t, err)
		assertReparsed(t, data)
	})
}
