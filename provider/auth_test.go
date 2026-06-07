package provider

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestApply_OAuth2UsesBearerToken(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "https://api.example.com/pets", nil)
	scheme := &AuthScheme{Type: AuthOAuth2, Flow: FlowClientCredentials}

	scheme.Apply(req, &Options{Token: "abc123"})
	assert.Equal(t, "Bearer abc123", req.Header.Get("Authorization"))
}

func TestApply_OAuth2NoTokenNoHeader(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "https://api.example.com/pets", nil)
	scheme := &AuthScheme{Type: AuthOAuth2}

	scheme.Apply(req, &Options{})
	assert.Empty(t, req.Header.Get("Authorization"))
}

func TestApply_BearerStillWorks(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "https://api.example.com/pets", nil)
	scheme := &AuthScheme{Type: AuthBearer}

	scheme.Apply(req, &Options{Token: "tok"})
	assert.Equal(t, "Bearer tok", req.Header.Get("Authorization"))
}
