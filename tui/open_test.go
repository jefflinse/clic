package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEditorCommand_SplitsBinaryAndFlags(t *testing.T) {
	name, args := editorCommand("code -w")
	assert.Equal(t, "code", name)
	assert.Equal(t, []string{"-w"}, args)

	name, args = editorCommand("vim")
	assert.Equal(t, "vim", name)
	assert.Empty(t, args)

	name, _ = editorCommand("")
	assert.Equal(t, "", name)

	name, _ = editorCommand("   ")
	assert.Equal(t, "", name)
}

func TestResolveEditor_PrefersVisualThenEditor(t *testing.T) {
	t.Setenv("VISUAL", "myvisual")
	t.Setenv("EDITOR", "myeditor")
	assert.Equal(t, "myvisual", resolveEditor())

	t.Setenv("VISUAL", "")
	assert.Equal(t, "myeditor", resolveEditor())

	t.Setenv("EDITOR", "")
	assert.NotEmpty(t, resolveEditor()) // platform fallback (vi / notepad)
}

func TestExtForContentType(t *testing.T) {
	cases := map[string]string{
		"application/json":            ".json",
		"application/json; charset=8": ".json",
		"text/html":                   ".html",
		"application/xml":             ".xml",
		"text/plain":                  ".txt",
		"":                            ".txt",
	}
	for ct, want := range cases {
		assert.Equal(t, want, extForContentType(ct), ct)
	}
}

func TestWriteTempResponse_UsesContentTypeExtension(t *testing.T) {
	path, err := writeTempResponse(jsonResult(`{"ok":true}`))
	require.NoError(t, err)
	assert.True(t, len(path) > 5 && path[len(path)-5:] == ".json", path)
}
