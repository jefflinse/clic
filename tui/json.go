package tui

import (
	"bytes"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// jsonStyles holds the per-token styles used to syntax-highlight JSON.
type jsonStyles struct {
	key     lipgloss.Style // object keys
	str     lipgloss.Style // string values
	num     lipgloss.Style // numbers
	boolean lipgloss.Style // true / false
	null    lipgloss.Style // null
	punct   lipgloss.Style // braces, brackets, commas, colons
}

// highlightJSON pretty-prints and syntax-highlights raw JSON with two-space
// indentation, preserving object key order. It reports false when the input is
// not valid JSON, so callers can fall back to displaying it verbatim.
func highlightJSON(raw []byte, s jsonStyles) (string, bool) {
	dec := json.NewDecoder(bytes.NewReader(bytes.TrimSpace(raw)))
	dec.UseNumber()

	var b strings.Builder
	if err := writeValue(&b, dec, s, 0); err != nil {
		return "", false
	}

	// reject trailing junk so "{}garbage" doesn't render as a clean object
	if dec.More() {
		return "", false
	}

	return b.String(), true
}

func writeValue(b *strings.Builder, dec *json.Decoder, s jsonStyles, indent int) error {
	tok, err := dec.Token()
	if err != nil {
		return err
	}

	if delim, ok := tok.(json.Delim); ok {
		switch delim {
		case '{':
			return writeObject(b, dec, s, indent)
		case '[':
			return writeArray(b, dec, s, indent)
		}
	}

	b.WriteString(scalarToken(tok, s))
	return nil
}

func writeObject(b *strings.Builder, dec *json.Decoder, s jsonStyles, indent int) error {
	b.WriteString(s.punct.Render("{"))
	first := true
	for dec.More() {
		if !first {
			b.WriteString(s.punct.Render(","))
		}
		first = false
		b.WriteByte('\n')
		writeIndent(b, indent+1)

		keyTok, err := dec.Token()
		if err != nil {
			return err
		}
		key, _ := keyTok.(string)
		b.WriteString(s.key.Render(quote(key)))
		b.WriteString(s.punct.Render(": "))

		if err := writeValue(b, dec, s, indent+1); err != nil {
			return err
		}
	}
	if _, err := dec.Token(); err != nil { // consume '}'
		return err
	}
	if !first {
		b.WriteByte('\n')
		writeIndent(b, indent)
	}
	b.WriteString(s.punct.Render("}"))
	return nil
}

func writeArray(b *strings.Builder, dec *json.Decoder, s jsonStyles, indent int) error {
	b.WriteString(s.punct.Render("["))
	first := true
	for dec.More() {
		if !first {
			b.WriteString(s.punct.Render(","))
		}
		first = false
		b.WriteByte('\n')
		writeIndent(b, indent+1)
		if err := writeValue(b, dec, s, indent+1); err != nil {
			return err
		}
	}
	if _, err := dec.Token(); err != nil { // consume ']'
		return err
	}
	if !first {
		b.WriteByte('\n')
		writeIndent(b, indent)
	}
	b.WriteString(s.punct.Render("]"))
	return nil
}

func scalarToken(tok json.Token, s jsonStyles) string {
	switch t := tok.(type) {
	case string:
		return s.str.Render(quote(t))
	case json.Number:
		return s.num.Render(t.String())
	case bool:
		return s.boolean.Render(strconv.FormatBool(t))
	case nil:
		return s.null.Render("null")
	default:
		return ""
	}
}

func writeIndent(b *strings.Builder, depth int) {
	for range depth {
		b.WriteString("  ")
	}
}

// quote renders a string as a properly-escaped JSON string literal.
func quote(s string) string {
	b, err := json.Marshal(s)
	if err != nil {
		return strconv.Quote(s)
	}
	return string(b)
}
