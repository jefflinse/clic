package tui

import (
	"context"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSubsequence(t *testing.T) {
	// subsequence is case-sensitive; the palette lowercases both sides first
	assert.True(t, subsequence("gbi", "pets / getbyid"))
	assert.True(t, subsequence("", "anything"))
	assert.False(t, subsequence("zzz", "pets / getbyid"))
	assert.False(t, subsequence("idg", "getbyid")) // order matters
}

func TestPalette_IndexesAllLeaves(t *testing.T) {
	s := newStudio(context.Background(), testApp())
	sized(s, 120, 40)
	s.openPalette()
	require.NotNil(t, s.pal)
	// getById, list, ping
	assert.Len(t, s.pal.all, 3)
	assert.Len(t, s.pal.matches, 3)
}

func TestPalette_FilterAndJump(t *testing.T) {
	s := newStudio(context.Background(), testApp())
	sized(s, 120, 40)
	s.openPalette()

	for _, r := range "ping" {
		s.handlePaletteKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	require.Len(t, s.pal.matches, 1)
	assert.Equal(t, "ping", s.pal.matches[0].path)

	s.handlePaletteKey(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Nil(t, s.pal, "palette closes on jump")
	require.NotNil(t, s.leaf)
	assert.Equal(t, "ping", s.leaf.Name)
}

func TestPalette_OpensViaSlashOutsideRequest(t *testing.T) {
	s := newStudio(context.Background(), testApp())
	sized(s, 120, 40)
	s.Update(key("/"))
	assert.NotNil(t, s.pal, "/ opens palette from a list pane")
}

func TestPalette_SlashIsLiteralInRequestForm(t *testing.T) {
	s := newStudio(context.Background(), testApp())
	sized(s, 120, 40)
	s.focus = focusRequest
	s.Update(key("/"))
	assert.Nil(t, s.pal, "/ is text while editing the request form")
}

func TestHelpOverlay_Toggles(t *testing.T) {
	s := newStudio(context.Background(), testApp())
	sized(s, 120, 40)
	s.Update(key("?"))
	assert.True(t, s.helpOpen)
	assert.Contains(t, s.View(), "command palette")

	s.Update(key("x")) // any key closes
	assert.False(t, s.helpOpen)
}
