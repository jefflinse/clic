package tui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/itchyny/gojq"
)

// editKind selects which response transform the inline input bar is editing.
type editKind int

const (
	editSearch editKind = iota // '/' incremental search
	editFilter                 // 'f' jq filter
)

// editState backs the inline input bar at the bottom of the screen used to type
// a search query or a jq program. It captures all keystrokes while open.
type editState struct {
	kind editKind
	buf  string
}

// startSearch opens the search input, seeded with any current query so it can be
// refined rather than retyped.
func (s *studio) startSearch() {
	if s.resp.result == nil {
		s.flash = "no response to search"
		return
	}
	s.editing = &editState{kind: editSearch, buf: s.resp.search.query}
}

// startFilter opens the jq filter input, seeded with the current program.
func (s *studio) startFilter() {
	if s.resp.result == nil {
		s.flash = "no response to filter"
		return
	}
	s.editing = &editState{kind: editFilter, buf: s.resp.filter.program}
}

// handleEditKey drives the inline input bar. Search applies live on every
// keystroke (cheap substring scan); the jq filter applies on enter, since
// evaluating a half-typed program every keystroke is wasteful and noisy.
func (s *studio) handleEditKey(msg tea.KeyMsg) tea.Cmd {
	e := s.editing
	switch msg.String() {
	case "ctrl+c":
		return tea.Quit
	case "esc":
		if e.kind == editSearch {
			s.resp.clearSearch() // drop the live highlight applied while typing
		}
		s.editing = nil
	case "enter":
		s.commitEdit(e)
		s.editing = nil
	case "backspace":
		if n := len(e.buf); n > 0 {
			e.buf = e.buf[:n-1]
			s.liveEdit(e)
		}
	case "ctrl+u":
		e.buf = ""
		s.liveEdit(e)
	default:
		if len(msg.Runes) >= 1 {
			e.buf += string(msg.Runes)
			s.liveEdit(e)
		}
	}
	return nil
}

// liveEdit reflects an in-progress search immediately; the jq filter waits for
// enter.
func (s *studio) liveEdit(e *editState) {
	if e.kind == editSearch {
		s.resp.setSearch(e.buf)
	}
}

// commitEdit finalizes the input: a search jumps to its first hit, a filter runs
// and (on error) flashes why.
func (s *studio) commitEdit(e *editState) {
	switch e.kind {
	case editSearch:
		s.resp.setSearch(e.buf)
		if e.buf != "" && len(s.resp.search.lines) == 0 {
			s.flash = "no matches"
		}
	case editFilter:
		s.resp.applyFilter(e.buf)
		if s.resp.filter.err != nil {
			s.flash = "jq: " + oneLine(s.resp.filter.err.Error())
		}
	}
}

// runJQ evaluates a jq program against a JSON body and returns the results as
// indented JSON. Multiple outputs are emitted one per line, mirroring jq.
func runJQ(program string, input []byte) ([]byte, error) {
	query, err := gojq.Parse(program)
	if err != nil {
		return nil, err
	}

	var data any
	if err := json.Unmarshal(input, &data); err != nil {
		return nil, fmt.Errorf("response is not JSON")
	}

	var out bytes.Buffer
	iter := query.Run(data)
	count := 0
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, ok := v.(error); ok {
			return nil, err
		}
		chunk, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			return nil, err
		}
		if count > 0 {
			out.WriteByte('\n')
		}
		out.Write(chunk)
		count++
	}
	return out.Bytes(), nil
}

// prettyPlainJSON indents JSON without styling, for the plain text the search
// runs over. It reports false when the input is not valid JSON.
func prettyPlainJSON(raw []byte) (string, bool) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || !json.Valid(trimmed) {
		return "", false
	}
	var out bytes.Buffer
	if err := json.Indent(&out, trimmed, "", "  "); err != nil {
		return "", false
	}
	return out.String(), true
}

// highlightMatches wraps every case-insensitive occurrence of query in line with
// the given style, reporting whether any hit was found. Matching is byte-based,
// which is exact for ASCII queries (the common case for JSON keys and ids).
func highlightMatches(line, query string, style lipgloss.Style) (string, bool) {
	if query == "" {
		return line, false
	}
	lowLine, lowQuery := strings.ToLower(line), strings.ToLower(query)
	if len(lowLine) != len(line) || len(lowQuery) != len(query) {
		// lowercasing changed byte length (non-ASCII); fall back to a plain,
		// non-highlighted line rather than risk slicing mid-rune.
		return line, strings.Contains(lowLine, lowQuery)
	}

	var b strings.Builder
	found := false
	for i := 0; i < len(line); {
		j := strings.Index(lowLine[i:], lowQuery)
		if j < 0 {
			b.WriteString(line[i:])
			break
		}
		j += i
		b.WriteString(line[i:j])
		b.WriteString(style.Render(line[j : j+len(query)]))
		i = j + len(query)
		found = true
	}
	return b.String(), found
}
