package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// leafRef points at a runnable command anywhere in the tree, with the indices
// needed to jump the studio's selection there.
type leafRef struct {
	path     string
	groupIdx int
	entryIdx int
	cmd      *Command
}

// paletteState backs the command palette: a fuzzy finder over every runnable
// command in the app, opened with '/' or ctrl+p.
type paletteState struct {
	query   string
	idx     int
	all     []leafRef
	matches []leafRef
}

// filter recomputes the visible matches for the current query (a case-
// insensitive subsequence match against each command's full path).
func (p *paletteState) filter() {
	if p.query == "" {
		p.matches = p.all
	} else {
		// fresh slice: never reuse p.matches, which may alias p.all
		matches := make([]leafRef, 0, len(p.all))
		needle := strings.ToLower(p.query)
		for _, ref := range p.all {
			if subsequence(needle, strings.ToLower(ref.path)) {
				matches = append(matches, ref)
			}
		}
		p.matches = matches
	}
	if p.idx >= len(p.matches) {
		p.idx = len(p.matches) - 1
	}
	if p.idx < 0 {
		p.idx = 0
	}
}

// subsequence reports whether needle's runes appear in order within haystack.
func subsequence(needle, haystack string) bool {
	if needle == "" {
		return true
	}
	n := []rune(needle)
	i := 0
	for _, r := range haystack {
		if r == n[i] {
			if i++; i == len(n) {
				return true
			}
		}
	}
	return false
}

func (s *studio) openPalette() {
	p := &paletteState{all: s.buildLeafIndex()}
	p.filter()
	s.pal = p
}

// buildLeafIndex collects every runnable command across all groups for the
// palette, tagged with where to jump to select it.
func (s *studio) buildLeafIndex() []leafRef {
	var refs []leafRef
	for gi := range s.groups {
		entries := entriesFor(&s.groups[gi])
		for ei := range entries {
			if entries[ei].cmd != nil {
				refs = append(refs, leafRef{
					path:     entries[ei].path,
					groupIdx: gi,
					entryIdx: ei,
					cmd:      entries[ei].cmd,
				})
			}
		}
	}
	return refs
}

func (s *studio) handlePaletteKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc", "ctrl+p":
		s.pal = nil
	case "enter":
		if s.pal.idx < len(s.pal.matches) {
			ref := s.pal.matches[s.pal.idx]
			s.pal = nil
			return s.jumpTo(ref)
		}
		s.pal = nil
	case "up", "ctrl+k":
		if s.pal.idx > 0 {
			s.pal.idx--
		}
	case "down", "ctrl+j":
		if s.pal.idx < len(s.pal.matches)-1 {
			s.pal.idx++
		}
	case "backspace":
		if n := len(s.pal.query); n > 0 {
			s.pal.query = s.pal.query[:n-1]
			s.pal.filter()
		}
	case "ctrl+u":
		s.pal.query = ""
		s.pal.filter()
	default:
		if len(msg.Runes) == 1 {
			s.pal.query += string(msg.Runes)
			s.pal.filter()
		}
	}
	return nil
}

// jumpTo points the studio's selection at a command anywhere in the tree and
// focuses its request form (or the commands column when it takes no input).
func (s *studio) jumpTo(ref leafRef) tea.Cmd {
	s.groupIdx = ref.groupIdx
	s.syncEntries()
	s.commandIdx = ref.entryIdx
	cmd := s.selectLeafFromEntries()
	if s.req != nil && s.req.hasInputs() {
		s.setFocus(focusRequest)
	} else {
		s.setFocus(focusCommands)
	}
	return cmd
}

// ---- rendering -------------------------------------------------------------

// renderHelp draws the keybinding reference as a centered modal.
func (s *studio) renderHelp() string {
	type row struct{ key, desc string }
	rows := []row{
		{"tab ⁄ shift+tab", "cycle panes (move between form fields)"},
		{"↑ ↓ / j k", "select in a list · scroll the response"},
		{"← → / h l", "move between columns · switch response view"},
		{"enter", "drill in / edit the request / run"},
		{"esc", "step back toward the command tree"},
		{"ctrl+s", "send the request"},
		{"ctrl+p", "command palette (jump anywhere)"},
		{"c", "copy request as curl / clic / url, or response"},
		{"x", "capture a response value as a {{variable}}"},
		{"v", "list captured variables"},
		{"y", "copy response body to clipboard"},
		{"/  ·  n ⁄ N", "search the response · jump between hits"},
		{"f", "filter the response with a jq program"},
		{"o", "open the response in $EDITOR"},
		{"A", "sign in (OAuth2 apps)"},
		{"?", "toggle this help"},
		{"ctrl+c", "quit"},
	}

	var b strings.Builder
	b.WriteString(s.th.title.Render("clic studio") + s.th.subtitle.Render("  ·  keys"))
	b.WriteString("\n\n")
	for _, r := range rows {
		b.WriteString(s.th.helpKey.Render(padRight(r.key, 14)))
		b.WriteString(s.th.help.Render(r.desc))
		b.WriteByte('\n')
	}
	b.WriteString("\n")
	b.WriteString(s.th.desc.Render("press any key to close"))

	return s.modal(b.String())
}

// renderPalette draws the command palette as a centered modal.
func (s *studio) renderPalette() string {
	var b strings.Builder
	b.WriteString(s.th.paneTitleHot.Render("⌕ ") + s.th.title.Render(s.pal.query) + s.th.json.boolean.Render("▌"))
	b.WriteString("\n")
	b.WriteString(s.th.desc.Render(strings.Repeat("─", 52)))
	b.WriteString("\n")

	if len(s.pal.matches) == 0 {
		b.WriteString(s.th.desc.Render("no matching commands"))
	}

	const maxRows = 12
	start := 0
	if s.pal.idx >= maxRows {
		start = s.pal.idx - maxRows + 1
	}
	for i := start; i < len(s.pal.matches) && i < start+maxRows; i++ {
		ref := s.pal.matches[i]
		line := "  " + ref.path
		if i == s.pal.idx {
			b.WriteString(s.th.selected.Width(52).Render("▸ " + ref.path))
		} else {
			b.WriteString(s.th.group.Render(truncate(line, 52)))
		}
		b.WriteByte('\n')
	}
	b.WriteString("\n")
	b.WriteString(s.th.help.Render("↑↓ select · ⏎ jump · esc cancel"))

	return s.modal(b.String())
}

// modal centers content in a bordered box over the full screen.
func (s *studio) modal(content string) string {
	box := s.th.borderHot.Padding(1, 3).Render(content)
	return lipgloss.Place(s.width, s.height, lipgloss.Center, lipgloss.Center, box)
}

func padRight(s string, w int) string {
	if len(s) >= w {
		return s
	}
	return s + strings.Repeat(" ", w-len(s))
}
