package clic_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jefflinse/clic"
	"github.com/jefflinse/clic/provider"
	"github.com/jefflinse/clic/spec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewApp(t *testing.T) {
	tests := []struct {
		name        string
		json        string
		expectError bool
	}{
		{
			name:        "succeeds when no commands are present",
			json:        `{"name":"app","description":"the app"}`,
			expectError: false,
		},
		{
			name:        "succeeds with a valid command",
			json:        `{"name":"app","description":"the app","commands":[{"name":"cmd","description":"the cmd","noop":{}}]}`,
			expectError: false,
		},
		{
			name:        "fails on invalid JSON",
			json:        `{"name":"app","description}`,
			expectError: true,
		},
		{
			name:        "fails when spec is invalid",
			json:        `{"name":"app"}`,
			expectError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			app, err := clic.NewApp([]byte(test.json))
			if test.expectError {
				assert.Error(t, err)
				assert.Nil(t, app)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, app)
			}
		})
	}
}

func TestApp_Run(t *testing.T) {
	app, err := clic.NewApp([]byte(`{"name":"app","description":"the app"}`))
	assert.NoError(t, err)
	assert.NoError(t, app.Run([]string{}))
}

// TestApp_StandaloneServerOverride verifies that a standalone app (as produced
// by a built binary) resolves the global --server flag into the context so the
// rest provider targets the override, with the flag appearing before the
// command's own args.
func TestApp_StandaloneServerOverride(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		fmt.Fprintln(w, "pong")
	}))
	defer srv.Close()

	doc := `{"name":"api","description":"x","commands":[{"name":"ping","description":"ping","rest":{"endpoint":"/ping","method":"GET"}}]}`
	app, err := clic.NewApp([]byte(doc))
	require.NoError(t, err)

	require.NoError(t, app.Run([]string{"ping", "--server", srv.URL}))
	assert.Equal(t, "/ping", gotPath, "request should have reached the --server override")
}

// TestApp_LauncherOptionsViaContext verifies the launcher path: options are
// supplied through the context (not as app flags) and still reach the provider.
func TestApp_LauncherOptionsViaContext(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		fmt.Fprintln(w, "pong")
	}))
	defer srv.Close()

	doc := `{"name":"api","description":"x","commands":[{"name":"ping","description":"ping","rest":{"endpoint":"/ping","method":"GET"}}]}`
	appSpec, err := spec.NewAppSpec([]byte(doc))
	require.NoError(t, err)
	app, err := clic.NewAppFromSpec(appSpec)
	require.NoError(t, err)

	ctx := provider.WithOptions(context.Background(), &provider.Options{Server: srv.URL})
	require.NoError(t, app.RunContext(ctx, []string{"ping"}))
	assert.Equal(t, "/ping", gotPath)
}
