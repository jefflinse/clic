package handyman_test

import (
	"testing"

	"github.com/jefflinse/handyman"
	"github.com/stretchr/testify/assert"
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
			app, err := handyman.NewApp([]byte(test.json))
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
	app, err := handyman.NewApp([]byte(`{"name":"app","description":"the app"}`))
	assert.NoError(t, err)
	assert.NoError(t, app.Run([]string{}))
}
