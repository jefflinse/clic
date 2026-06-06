package tui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// jsonLeaf is one scalar value in a response, addressed by its JSON path, that
// the user can capture into a variable for request chaining.
type jsonLeaf struct {
	path  string
	value string
}

// captureState backs the capture overlay ('x'): a two-phase flow that first
// picks a scalar from the last response, then names it as a {{variable}}.
type captureState struct {
	pairs  []jsonLeaf
	idx    int
	naming bool
	name   string
	value  string
}

// openCapture flattens the current response into its scalar values and opens the
// capture overlay, or flashes why it cannot.
func (s *studio) openCapture() {
	if s.resp.result == nil {
		s.flash = "no response to capture from"
		return
	}
	leaves := flattenJSONLeaves(s.resp.result.Body)
	if len(leaves) == 0 {
		s.flash = "response has no JSON values to capture"
		return
	}
	s.cap = &captureState{pairs: leaves}
}

func (s *studio) handleCaptureKey(msg tea.KeyMsg) tea.Cmd {
	c := s.cap
	if c.naming {
		return s.handleCaptureNaming(msg)
	}
	switch msg.String() {
	case "esc", "x", "q":
		s.cap = nil
	case "up", "k", "ctrl+k":
		if c.idx > 0 {
			c.idx--
		}
	case "down", "j", "ctrl+j":
		if c.idx < len(c.pairs)-1 {
			c.idx++
		}
	case "enter":
		leaf := c.pairs[c.idx]
		c.value = leaf.value
		c.name = suggestVarName(leaf.path)
		c.naming = true
	}
	return nil
}

func (s *studio) handleCaptureNaming(msg tea.KeyMsg) tea.Cmd {
	c := s.cap
	switch msg.String() {
	case "esc":
		c.naming = false
	case "enter":
		name := strings.TrimSpace(c.name)
		if name == "" {
			return nil
		}
		s.setVar(name, c.value)
		s.cap = nil
		s.flash = "captured {{" + name + "}}"
		s.refreshPreview()
	case "backspace":
		if n := len(c.name); n > 0 {
			c.name = c.name[:n-1]
		}
	case "ctrl+u":
		c.name = ""
	default:
		if len(msg.Runes) == 1 {
			c.name += string(msg.Runes)
		}
	}
	return nil
}

// renderCapture draws the capture overlay for whichever phase is active.
func (s *studio) renderCapture() string {
	c := s.cap
	var b strings.Builder
	if c.naming {
		b.WriteString(s.th.title.Render("capture variable"))
		b.WriteString("\n\n")
		b.WriteString(s.th.desc.Render("value  ") + s.th.json.str.Render(truncate(c.value, 44)))
		b.WriteString("\n\n")
		b.WriteString(s.th.group.Render("name   {{") + s.th.title.Render(c.name) +
			s.th.json.boolean.Render("▌") + s.th.group.Render("}}"))
		b.WriteString("\n\n")
		b.WriteString(s.th.help.Render("⏎ save · esc back"))
		return s.modal(b.String())
	}

	b.WriteString(s.th.title.Render("capture") + s.th.subtitle.Render("  ·  pick a value to reuse"))
	b.WriteString("\n\n")

	const maxRows = 12
	start := 0
	if c.idx >= maxRows {
		start = c.idx - maxRows + 1
	}
	for i := start; i < len(c.pairs) && i < start+maxRows; i++ {
		leaf := c.pairs[i]
		path := leaf.path
		if path == "" {
			path = "(value)"
		}
		if i == c.idx {
			b.WriteString(s.th.helpKey.Render("▸ ") + s.th.selected.Render(padRight(path, 24)) +
				" " + s.th.json.str.Render(truncate(leaf.value, 26)))
		} else {
			b.WriteString("  " + s.th.group.Render(padRight(path, 24)) +
				" " + s.th.desc.Render(truncate(leaf.value, 26)))
		}
		b.WriteByte('\n')
	}
	b.WriteString("\n")
	b.WriteString(s.th.help.Render("↑↓ select · ⏎ name it · esc cancel"))
	return s.modal(b.String())
}

// renderVars lists the captured variables as a reference card.
func (s *studio) renderVars() string {
	var b strings.Builder
	b.WriteString(s.th.title.Render("variables") + s.th.subtitle.Render("  ·  reference as {{name}}"))
	b.WriteString("\n\n")
	if len(s.vars) == 0 {
		b.WriteString(s.th.desc.Render("none captured yet — press x on a response to capture one"))
	}
	for _, v := range s.vars {
		b.WriteString(s.th.json.key.Render(padRight("{{"+v.name+"}}", 18)))
		b.WriteString(s.th.json.str.Render(truncate(v.value, 40)))
		b.WriteByte('\n')
	}
	b.WriteString("\n")
	b.WriteString(s.th.desc.Render("press any key to close"))
	return s.modal(b.String())
}

// flattenJSONLeaves walks a JSON document in document order, returning every
// scalar value addressed by its path (e.g. ".data[0].id"). Non-JSON input
// yields no leaves.
func flattenJSONLeaves(raw []byte) []jsonLeaf {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()

	var leaves []jsonLeaf
	var parse func(prefix string) error
	parse = func(prefix string) error {
		tok, err := dec.Token()
		if err != nil {
			return err
		}
		delim, ok := tok.(json.Delim)
		if !ok {
			leaves = append(leaves, jsonLeaf{path: prefix, value: scalarString(tok)})
			return nil
		}
		switch delim {
		case '{':
			for dec.More() {
				keyTok, err := dec.Token()
				if err != nil {
					return err
				}
				key, _ := keyTok.(string)
				if err := parse(prefix + "." + key); err != nil {
					return err
				}
			}
		case '[':
			for i := 0; dec.More(); i++ {
				if err := parse(fmt.Sprintf("%s[%d]", prefix, i)); err != nil {
					return err
				}
			}
		}
		_, err = dec.Token() // consume the matching closing delim
		return err
	}

	if err := parse(""); err != nil {
		return leaves // a partial walk is still useful
	}
	return leaves
}

func scalarString(v any) string {
	switch t := v.(type) {
	case nil:
		return "null"
	case bool:
		if t {
			return "true"
		}
		return "false"
	case json.Number:
		return t.String()
	case string:
		return t
	default:
		return fmt.Sprintf("%v", t)
	}
}

// suggestVarName derives a tidy variable name from a JSON path's last segment
// (e.g. ".data[0].petId" -> "petId").
func suggestVarName(path string) string {
	seg := path
	if i := strings.LastIndex(seg, "."); i >= 0 {
		seg = seg[i+1:]
	}
	if j := strings.IndexByte(seg, '['); j >= 0 {
		seg = seg[:j]
	}
	var b strings.Builder
	for _, r := range seg {
		switch {
		case r == '_',
			r >= 'a' && r <= 'z',
			r >= 'A' && r <= 'Z',
			r >= '0' && r <= '9':
			b.WriteRune(r)
		}
	}
	if b.Len() == 0 {
		return "var"
	}
	return b.String()
}
