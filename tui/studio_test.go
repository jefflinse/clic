package tui

import (
	"context"
	"net/http"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jefflinse/clic/form"
	"github.com/jefflinse/clic/provider"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeProvider is a provider.Interactive + Describer for exercising the studio
// without importing a real provider (which would create an import cycle, since
// rest imports tui).
type fakeProvider struct {
	sections []provider.Section
	result   *provider.Result
	preview  *provider.RequestPreview
	summary  string
	gotInput provider.Inputs
}

func (f *fakeProvider) Configure(*cobra.Command)     {}
func (f *fakeProvider) Type() string                 { return "fake" }
func (f *fakeProvider) Validate() error              { return nil }
func (f *fakeProvider) Sections() []provider.Section { return f.sections }
func (f *fakeProvider) Summary() string              { return f.summary }
func (f *fakeProvider) Execute(_ context.Context, in provider.Inputs) (*provider.Result, error) {
	f.gotInput = in
	return f.result, nil
}

// Preview echoes the collected inputs back so tests can assert variable
// substitution and live preview wiring. It reports a simple HTTP request.
func (f *fakeProvider) Preview(_ context.Context, in provider.Inputs) (*provider.RequestPreview, error) {
	f.gotInput = in
	if f.preview != nil {
		return f.preview, nil
	}
	url := "https://api.petstore.io/pets"
	if id := in.Scalar("path", "id"); id != nil {
		url += "/" + id.(string)
	}
	return &provider.RequestPreview{
		Kind:    provider.ResultHTTP,
		Method:  "GET",
		URL:     url,
		Headers: http.Header{"Content-Type": []string{"application/json"}},
	}, nil
}

func testApp() StudioApp {
	return StudioApp{
		Name:        "petstore",
		Description: "manage pets",
		Server:      "https://api.petstore.io",
		Commands: []Command{
			{
				Name: "pets",
				Subcommands: []Command{
					{Name: "getById", Provider: &fakeProvider{
						summary:  "GET /pets/{id}",
						sections: []provider.Section{{Key: "path", Title: "Path", Fields: []form.Field{{Name: "id", Type: form.StringField, Required: true}}}},
					}},
					{Name: "list", Provider: &fakeProvider{summary: "GET /pets"}},
				},
			},
			{Name: "ping", Provider: &fakeProvider{summary: "GET /ping"}},
		},
	}
}

func sized(s *studio, w, h int) {
	s.Update(tea.WindowSizeMsg{Width: w, Height: h})
}

func key(s string) tea.KeyMsg {
	if len(s) == 1 {
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
	}
	switch s {
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "right":
		return tea.KeyMsg{Type: tea.KeyRight}
	case "left":
		return tea.KeyMsg{Type: tea.KeyLeft}
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func TestStudio_RendersPanes(t *testing.T) {
	s := newStudio(context.Background(), testApp())
	sized(s, 120, 40)

	view := s.View()
	for _, want := range []string{"petstore", "manage pets", "api.petstore.io", "GROUPS", "COMMANDS", "REQUEST", "pets", "ping"} {
		assert.Contains(t, view, want)
	}
}

func TestStudio_FlattensGroupChildrenAndSnapsToLeaf(t *testing.T) {
	s := newStudio(context.Background(), testApp())
	sized(s, 120, 40)

	// the first group "pets" has two leaves; the highlighted command should be a
	// selectable leaf, and its provider should drive the request form.
	require.NotNil(t, s.leaf)
	assert.Equal(t, "getById", s.leaf.Name)
	assert.True(t, s.req.hasInputs())
}

func TestStudio_FocusFlow(t *testing.T) {
	s := newStudio(context.Background(), testApp())
	sized(s, 120, 40)
	assert.Equal(t, focusGroups, s.focus)

	s.Update(key("right")) // groups -> commands
	assert.Equal(t, focusCommands, s.focus)

	s.Update(key("right")) // commands -> request (getById has inputs)
	assert.Equal(t, focusRequest, s.focus)

	s.Update(tea.KeyMsg{Type: tea.KeyEsc}) // back to commands
	assert.Equal(t, focusCommands, s.focus)

	s.Update(key("h")) // commands -> groups
	assert.Equal(t, focusGroups, s.focus)
}

func TestStudio_SelectingDifferentLeafRebuildsForm(t *testing.T) {
	s := newStudio(context.Background(), testApp())
	sized(s, 120, 40)
	s.Update(key("right")) // into commands

	s.Update(key("down")) // getById -> list
	require.NotNil(t, s.leaf)
	assert.Equal(t, "list", s.leaf.Name)
	assert.False(t, s.req.hasInputs(), "list has no sections")
}

func TestStudio_Preselect(t *testing.T) {
	s := newStudio(context.Background(), testApp())
	s.preselect([]string{"pets", "getById"})
	sized(s, 120, 40)

	require.NotNil(t, s.leaf)
	assert.Equal(t, "getById", s.leaf.Name)
	assert.Equal(t, focusRequest, s.focus)
}

func TestStudio_ResultMsgRendersStatus(t *testing.T) {
	s := newStudio(context.Background(), testApp())
	sized(s, 120, 40)

	s.Update(resultMsg{res: &provider.Result{
		Kind:    provider.ResultHTTP,
		Status:  http.StatusOK,
		Headers: http.Header{"Content-Type": []string{"application/json"}},
		Body:    []byte(`{"ok":true}`),
	}})

	assert.Equal(t, focusResponse, s.focus)
	view := s.View()
	assert.Contains(t, view, "200")
	assert.Contains(t, view, "OK")
}

func TestStudio_PreviewsRequestBeforeSending(t *testing.T) {
	s := newStudio(context.Background(), testApp())
	s.preselect([]string{"pets", "getById"})
	sized(s, 120, 40)

	// type an id into the request form; the preview pane should reflect it before
	// anything is sent.
	s.Update(key("4"))
	s.Update(key("2"))

	view := s.View()
	assert.Contains(t, view, "REQUEST PREVIEW")
	assert.Contains(t, view, "https://api.petstore.io/pets/42")
}

func TestStudio_CopyMenuOffersCurlClicAndURL(t *testing.T) {
	app := testApp()
	app.Invocation = "clic ./petstore.yaml"
	s := newStudio(context.Background(), app)
	s.preselect([]string{"pets", "getById"})
	sized(s, 120, 40)
	s.Update(key("7")) // id = 7

	s.openCopyMenu()
	require.NotNil(t, s.copy)

	var labels []string
	for _, item := range s.copy.items {
		labels = append(labels, item.label)
	}
	assert.Contains(t, labels, "request as curl")
	assert.Contains(t, labels, "clic command")
	assert.Contains(t, labels, "request URL")

	// the clic command reproduces the request headlessly
	for _, item := range s.copy.items {
		if item.label == "clic command" {
			assert.Equal(t, "clic ./petstore.yaml pets getById", item.value)
		}
		if item.label == "request as curl" {
			assert.Contains(t, item.value, "curl ")
			assert.Contains(t, item.value, "https://api.petstore.io/pets/7")
		}
	}

	view := s.View()
	assert.Contains(t, view, "to clipboard")
}

func TestStudio_TopLevelLeafIsSelectable(t *testing.T) {
	s := newStudio(context.Background(), testApp())
	sized(s, 120, 40)

	// move to the top-level "ping" leaf group; its commands column should hold
	// the leaf itself.
	s.Update(key("down")) // groups: pets -> ping
	require.NotNil(t, s.leaf)
	assert.Equal(t, "ping", s.leaf.Name)
}

func TestStudio_HelpBarReflectsFocus(t *testing.T) {
	s := newStudio(context.Background(), testApp())
	sized(s, 120, 40)
	assert.Contains(t, s.helpBar(), "groups")

	s.Update(key("right"))
	s.Update(key("right")) // into request
	assert.Contains(t, strings.ToLower(s.helpBar()), "send")
}
