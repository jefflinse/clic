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
			name:        "success, no commands",
			json:        `{"name":"app","description":"the app"}`,
			expectError: false,
		},
		{
			name:        "success, with a valid command",
			json:        `{"name":"app","description":"the app","commands":[{"name":"cmd","description":"the cmd"}]}`,
			expectError: false,
		},
		{
			name:        "success, with a valid command containing valid subcommand",
			json:        `{"name":"app","description":"the app","commands":[{"name":"cmd","description":"the cmd","subcommands":[{"name":"subcmd","description":"the subcmd"}]}]}`,
			expectError: false,
		},
		{
			name:        "failure, bad JSON",
			json:        `{"name":"app","description}`,
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
