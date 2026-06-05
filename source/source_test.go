package source_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/jefflinse/clic/source"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsURL(t *testing.T) {
	assert.True(t, source.IsURL("http://example.com/spec.yml"))
	assert.True(t, source.IsURL("https://example.com/spec.yml"))
	assert.False(t, source.IsURL("/path/to/spec.yml"))
	assert.False(t, source.IsURL("spec.yml"))
}

func TestLoad_File(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "spec.yml")
	require.NoError(t, os.WriteFile(path, []byte("name: app"), 0644))

	data, err := source.Load(path)
	assert.NoError(t, err)
	assert.Equal(t, "name: app", string(data))
}

func TestLoad_File_NotFound(t *testing.T) {
	_, err := source.Load(filepath.Join(t.TempDir(), "missing.yml"))
	assert.Error(t, err)
}

func TestLoad_URL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("openapi: 3.0.0"))
	}))
	defer server.Close()

	data, err := source.Load(server.URL)
	assert.NoError(t, err)
	assert.Equal(t, "openapi: 3.0.0", string(data))
}

func TestLoad_URL_Non200(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	_, err := source.Load(server.URL)
	assert.Error(t, err)
}
