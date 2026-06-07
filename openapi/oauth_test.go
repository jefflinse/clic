package openapi_test

import (
	"testing"

	"github.com/jefflinse/clic/openapi"
	"github.com/jefflinse/clic/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func oauthSpec(flows string) string {
	return `
openapi: 3.0.0
info:
  title: Secured API
  description: An API behind OAuth2.
servers:
  - url: https://api.example.com
components:
  securitySchemes:
    oauth:
      type: oauth2
      flows:
` + flows + `
paths:
  /things:
    get:
      summary: list things
`
}

func TestCompile_OAuth2ClientCredentials(t *testing.T) {
	app, err := openapi.Compile([]byte(oauthSpec(`        clientCredentials:
          tokenUrl: https://auth.example.com/token
          scopes:
            read: read things
            write: write things`)))
	require.NoError(t, err)
	require.NotNil(t, app.Auth)
	assert.Equal(t, provider.AuthOAuth2, app.Auth.Type)
	assert.Equal(t, provider.FlowClientCredentials, app.Auth.Flow)
	assert.Equal(t, "https://auth.example.com/token", app.Auth.TokenURL)
	assert.Equal(t, []string{"read", "write"}, app.Auth.Scopes) // sorted
}

func TestCompile_OAuth2AuthorizationCode(t *testing.T) {
	app, err := openapi.Compile([]byte(oauthSpec(`        authorizationCode:
          authorizationUrl: https://auth.example.com/authorize
          tokenUrl: https://auth.example.com/token
          scopes:
            openid: identity`)))
	require.NoError(t, err)
	require.NotNil(t, app.Auth)
	assert.Equal(t, provider.FlowAuthorizationCode, app.Auth.Flow)
	assert.Equal(t, "https://auth.example.com/authorize", app.Auth.AuthURL)
	assert.Equal(t, "https://auth.example.com/token", app.Auth.TokenURL)
}

func TestCompile_OAuth2PrefersClientCredentials(t *testing.T) {
	app, err := openapi.Compile([]byte(oauthSpec(`        clientCredentials:
          tokenUrl: https://auth.example.com/token
          scopes: {}
        authorizationCode:
          authorizationUrl: https://auth.example.com/authorize
          tokenUrl: https://auth.example.com/token
          scopes: {}`)))
	require.NoError(t, err)
	require.NotNil(t, app.Auth)
	assert.Equal(t, provider.FlowClientCredentials, app.Auth.Flow)
}
