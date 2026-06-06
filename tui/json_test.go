package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// plainStyles renders tokens without color so tests can assert on structure.
func plainStyles() jsonStyles {
	s := lipgloss.NewStyle()
	return jsonStyles{key: s, str: s, num: s, boolean: s, null: s, punct: s}
}

func TestHighlightJSON_PrettyPrintsAndPreservesOrder(t *testing.T) {
	out, ok := highlightJSON([]byte(`{"id":42,"name":"Rex","tags":["a","b"],"ok":true,"x":null}`), plainStyles())
	require.True(t, ok)

	want := `{
  "id": 42,
  "name": "Rex",
  "tags": [
    "a",
    "b"
  ],
  "ok": true,
  "x": null
}`
	assert.Equal(t, want, out)
}

func TestHighlightJSON_Nested(t *testing.T) {
	out, ok := highlightJSON([]byte(`{"a":{"b":[1,2]}}`), plainStyles())
	require.True(t, ok)
	assert.Contains(t, out, "    \"b\": [")
	// "1" and "2" indented four spaces under the array
	assert.True(t, strings.Contains(out, "\n      1,"))
}

func TestHighlightJSON_RejectsInvalid(t *testing.T) {
	for _, in := range []string{`not json`, `{"a":}`, `{}garbage`, ``} {
		_, ok := highlightJSON([]byte(in), plainStyles())
		assert.False(t, ok, "expected %q to be rejected", in)
	}
}

func TestHighlightJSON_TopLevelArray(t *testing.T) {
	out, ok := highlightJSON([]byte(`[1,2,3]`), plainStyles())
	require.True(t, ok)
	assert.Equal(t, "[\n  1,\n  2,\n  3\n]", out)
}
