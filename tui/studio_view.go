package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/jefflinse/clic/provider"
)

// relayout recomputes pane dimensions for the current terminal size and resizes
// the embedded sub-models. It is called on resize and whenever the selected
// command changes (which may add or remove the request form).
func (s *studio) relayout() {
	if s.width <= 0 || s.height <= 0 {
		return
	}

	const topH, helpH = 1, 1
	bodyH := max(s.height-topH-helpH, 8)

	// the response pane takes roughly the bottom two-fifths, leaving the rest
	// to the three columns.
	s.respH = bodyH * 2 / 5
	s.respH = clamp(s.respH, 7, bodyH-6)
	s.colsH = bodyH - s.respH

	// outer column widths (each includes its 1-cell border on each side).
	s.gW = clamp(s.width/5, 16, 26)
	s.cW = clamp(s.width/4, 18, 32)
	s.rW = s.width - s.gW - s.cW
	if s.rW < 26 {
		s.cW = max(18, s.width-s.gW-26)
		s.rW = s.width - s.gW - s.cW
		if s.rW < 20 {
			s.gW = max(12, s.width-s.cW-20)
			s.rW = s.width - s.gW - s.cW
		}
	}

	if s.req != nil {
		// inner width/height: minus border (2) and the pane's title row (1).
		s.req.setSize(s.rW-2, s.colsH-3)
	}
	s.resp.setSize(s.width-2, s.respH-2)
}

func (s *studio) View() string {
	if s.width == 0 || s.height == 0 {
		return "loading…"
	}

	if s.helpOpen {
		return s.renderHelp()
	}
	if s.pal != nil {
		return s.renderPalette()
	}

	groups := s.pane("GROUPS", s.renderGroups(s.gW-2), s.gW, s.colsH, s.focus == focusGroups)
	commands := s.pane("COMMANDS", s.renderCommands(s.cW-2), s.cW, s.colsH, s.focus == focusCommands)
	request := s.pane(s.requestTitle(), s.renderRequest(), s.rW, s.colsH, s.focus == focusRequest)
	columns := lipgloss.JoinHorizontal(lipgloss.Top, groups, commands, request)

	respTitle := s.resp.summary()
	if s.sending {
		respTitle = s.spin.View() + s.th.subtitle.Render("sending…")
	}
	response := s.pane(respTitle, s.resp.vp.View(), s.width, s.respH, s.focus == focusResponse)

	return lipgloss.JoinVertical(lipgloss.Left, s.topBar(), columns, response, s.helpBar())
}

// pane renders a titled, bordered box of the given OUTER dimensions, clipping
// its title and body to fit.
func (s *studio) pane(title, body string, outerW, outerH int, hot bool) string {
	innerW := max(1, outerW-2)
	innerH := max(1, outerH-2)

	titleStyle := s.th.paneTitle
	box := s.th.border
	if hot {
		titleStyle = s.th.paneTitleHot
		box = s.th.borderHot
	}

	head := clip(titleStyle.Render(title), innerW, 1)
	content := head + "\n" + clip(body, innerW, max(1, innerH-1))
	return box.Width(innerW).Height(innerH).Render(content)
}

func (s *studio) topBar() string {
	left := s.th.title.Render(s.app.Name)
	if s.app.Description != "" {
		left += "  " + s.th.subtitle.Render(s.app.Description)
	}
	right := ""
	if s.app.Server != "" {
		right = s.th.server.Render("⇆ " + s.app.Server)
	}

	gap := max(s.width-lipgloss.Width(left)-lipgloss.Width(right), 1)
	return clip(left+strings.Repeat(" ", gap)+right, s.width, 1)
}

func (s *studio) helpBar() string {
	var hints [][2]string
	switch s.focus {
	case focusGroups:
		hints = [][2]string{{"↑↓", "groups"}, {"→", "commands"}, {"^s", "run"}, {"^c", "quit"}}
	case focusCommands:
		hints = [][2]string{{"↑↓", "select"}, {"←", "groups"}, {"→/⏎", "edit"}, {"^s", "run"}, {"^c", "quit"}}
	case focusRequest:
		hints = [][2]string{{"tab", "next field"}, {"^s", "send"}, {"esc", "back"}, {"^c", "quit"}}
	case focusResponse:
		hints = [][2]string{{"↑↓", "scroll"}, {"tab", "view"}, {"esc", "back"}, {"^s", "resend"}, {"^c", "quit"}}
	}

	// always advertise the palette and help, the two non-obvious global keys
	hints = append(hints, [2]string{"/", "find"}, [2]string{"?", "help"})

	parts := make([]string, 0, len(hints))
	for _, h := range hints {
		parts = append(parts, s.th.helpKey.Render(h[0])+" "+s.th.help.Render(h[1]))
	}
	bar := strings.Join(parts, s.th.help.Render("  ·  "))

	if s.flash != "" {
		bar = s.th.server.Render("✓ "+s.flash) + s.th.help.Render("    ") + bar
	}
	return clip(bar, s.width, 1)
}

func (s *studio) requestTitle() string {
	if s.leaf == nil {
		return "REQUEST"
	}
	if d, ok := s.leaf.Provider.(provider.Describer); ok {
		if summary := d.Summary(); summary != "" {
			return "REQUEST " + s.th.subtitle.Render(summary)
		}
	}
	return "REQUEST " + s.th.subtitle.Render(s.leaf.Name)
}

func (s *studio) renderGroups(w int) string {
	lines := make([]string, 0, len(s.groups))
	for i := range s.groups {
		c := &s.groups[i]
		marker := "• "
		if len(c.Subcommands) > 0 {
			marker = "▸ "
		}
		st := s.rowStyle(focusGroups, i == s.groupIdx)
		lines = append(lines, st.Width(w).Render(truncate(marker+c.Name, w)))
	}
	return strings.Join(lines, "\n")
}

func (s *studio) renderCommands(w int) string {
	if len(s.entries) == 0 {
		return s.th.desc.Render("(no commands)")
	}
	lines := make([]string, 0, len(s.entries))
	for i, e := range s.entries {
		indent := strings.Repeat("  ", e.depth)
		if e.cmd == nil {
			lines = append(lines, s.th.paneTitle.Render(truncate(indent+"▸ "+e.label, w)))
			continue
		}
		st := s.rowStyle(focusCommands, i == s.commandIdx)
		lines = append(lines, st.Width(w).Render(truncate(indent+"• "+e.label, w)))
	}
	return strings.Join(lines, "\n")
}

func (s *studio) renderRequest() string {
	if s.leaf == nil {
		return s.th.desc.Render("Select a command.")
	}
	if s.req == nil || s.req.form == nil {
		return s.th.desc.Render("This command takes no input.\n\nPress ctrl+s to run it.")
	}
	return s.req.form.View()
}

// rowStyle picks the style for a list row: bright selection in the focused pane,
// dimmed selection elsewhere, plain otherwise.
func (s *studio) rowStyle(zone focusZone, selected bool) lipgloss.Style {
	switch {
	case selected && s.focus == zone:
		return s.th.selected
	case selected:
		return s.th.selDim
	default:
		return s.th.group
	}
}

// clip pads/truncates content to exactly w columns by h rows.
func clip(content string, w, h int) string {
	return lipgloss.NewStyle().
		Width(w).
		Height(h).
		MaxWidth(w).
		MaxHeight(h).
		Render(content)
}

// truncate shortens a plain (unstyled) string to w columns, adding an ellipsis.
func truncate(s string, w int) string {
	r := []rune(s)
	if len(r) <= w {
		return s
	}
	if w <= 1 {
		return string(r[:max(0, w)])
	}
	return string(r[:w-1]) + "…"
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
