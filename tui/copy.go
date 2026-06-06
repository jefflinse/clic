package tui

import (
	"strings"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
)

// copyItem is one entry in the copy menu: a label and the text it puts on the
// clipboard.
type copyItem struct {
	label   string
	preview string // a short, one-line hint of what will be copied
	value   string
}

// copyMenu is the overlay opened with 'c' offering several copy-to-clipboard
// renderings of the current request and response.
type copyMenu struct {
	items []copyItem
	idx   int
}

// openCopyMenu assembles the applicable copy actions for the current selection
// and opens the menu. It does nothing when there is nothing worth copying.
func (s *studio) openCopyMenu() {
	items := s.copyItems()
	if len(items) == 0 {
		s.flash = "nothing to copy yet"
		return
	}
	s.copy = &copyMenu{items: items}
}

// copyItems builds the copy actions available right now: curl / clic / URL from
// the live request preview, and the response body once a result exists.
func (s *studio) copyItems() []copyItem {
	var items []copyItem

	if pv := s.resp.preview; pv != nil {
		if curl := curlCommand(pv); curl != "" {
			items = append(items, copyItem{label: "request as curl", preview: oneLine(curl), value: curl})
		}
		if clic := clicCommand(s.app.Invocation, s.selectedPath(), pv.CLIArgs); clic != "" {
			items = append(items, copyItem{label: "clic command", preview: clic, value: clic})
		}
		if pv.URL != "" {
			items = append(items, copyItem{label: "request URL", preview: pv.URL, value: pv.URL})
		}
	}

	if s.resp.result != nil && len(s.resp.result.Body) > 0 {
		body := string(s.resp.result.Body)
		items = append(items, copyItem{label: "response body", preview: oneLine(body), value: body})
	}

	return items
}

// selectedPath returns the command path of the highlighted leaf (e.g.
// ["pets","getById"]), recovered from its display path.
func (s *studio) selectedPath() []string {
	if s.commandIdx < 0 || s.commandIdx >= len(s.entries) {
		return nil
	}
	parts := strings.Split(s.entries[s.commandIdx].path, " / ")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

func (s *studio) handleCopyKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc", "c", "q":
		s.copy = nil
	case "up", "k", "ctrl+k":
		if s.copy.idx > 0 {
			s.copy.idx--
		}
	case "down", "j", "ctrl+j":
		if s.copy.idx < len(s.copy.items)-1 {
			s.copy.idx++
		}
	case "enter":
		item := s.copy.items[s.copy.idx]
		s.copy = nil
		if err := clipboard.WriteAll(item.value); err != nil {
			s.flash = "clipboard unavailable"
			return nil
		}
		s.flash = "copied " + item.label
	}
	return nil
}

// renderCopyMenu draws the copy menu as a centered modal.
func (s *studio) renderCopyMenu() string {
	var b strings.Builder
	b.WriteString(s.th.title.Render("copy") + s.th.subtitle.Render("  ·  to clipboard"))
	b.WriteString("\n\n")

	for i, item := range s.copy.items {
		marker := "  "
		label := s.th.group.Render(padRight(item.label, 18))
		if i == s.copy.idx {
			marker = s.th.helpKey.Render("▸ ")
			label = s.th.selected.Render(padRight(item.label, 18))
		}
		b.WriteString(marker + label + s.th.desc.Render(truncate(item.preview, 46)))
		b.WriteByte('\n')
	}

	b.WriteString("\n")
	b.WriteString(s.th.help.Render("↑↓ select · ⏎ copy · esc cancel"))
	return s.modal(b.String())
}

// oneLine collapses whitespace runs so multi-line content previews on one row.
func oneLine(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
