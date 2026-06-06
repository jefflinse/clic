package tui

import (
	"sort"
	"strings"

	"github.com/jefflinse/clic/provider"
)

// renderPreview formats a RequestPreview as colored, scrollable text for the
// response pane's "request" view: the request line, headers, and body for HTTP,
// or the resolved invocation for text providers.
func (r *responsePane) renderPreview() string {
	pv := r.preview
	switch {
	case pv == nil:
		return r.th.desc.Render("Select a command to preview its request.")
	case pv.Kind != provider.ResultHTTP:
		if pv.Display == "" {
			return r.th.desc.Render("(nothing to send)")
		}
		return r.th.hdrVal.Render(pv.Display)
	}

	var b strings.Builder
	b.WriteString(r.th.helpKey.Render(pv.Method))
	b.WriteByte(' ')
	b.WriteString(r.th.hdrVal.Render(pv.URL))
	b.WriteByte('\n')

	for _, name := range sortedHeaderNames(pv.Headers) {
		value := strings.Join(pv.Headers[name], ", ")
		b.WriteString(r.th.hdrKey.Render(name))
		b.WriteString(r.th.json.punct.Render(": "))
		b.WriteString(r.th.hdrVal.Render(value))
		b.WriteByte('\n')
	}

	if len(pv.Body) > 0 {
		b.WriteByte('\n')
		if pretty, ok := highlightJSON(pv.Body, r.th.json); ok {
			b.WriteString(pretty)
		} else {
			b.WriteString(string(pv.Body))
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

// curlCommand renders an HTTP preview as an equivalent curl invocation.
func curlCommand(pv *provider.RequestPreview) string {
	if pv == nil || pv.Kind != provider.ResultHTTP {
		return ""
	}
	var b strings.Builder
	b.WriteString("curl")
	if pv.Method != "" && pv.Method != "GET" {
		b.WriteString(" -X " + pv.Method)
	}
	b.WriteString(" " + shellQuote(pv.URL))
	for _, name := range sortedHeaderNames(pv.Headers) {
		for _, v := range pv.Headers[name] {
			b.WriteString(" \\\n  -H " + shellQuote(name+": "+v))
		}
	}
	if len(pv.Body) > 0 {
		b.WriteString(" \\\n  -d " + shellQuote(string(pv.Body)))
	}
	return b.String()
}

// clicCommand renders the headless clic invocation that reproduces a request:
// the launch prefix (e.g. "clic ./petstore.yaml"), the command path, and the
// preview's CLI arguments.
func clicCommand(invocation string, cmdPath, args []string) string {
	parts := make([]string, 0, len(cmdPath)+len(args)+1)
	if invocation != "" {
		parts = append(parts, invocation)
	}
	parts = append(parts, cmdPath...)
	for _, a := range args {
		parts = append(parts, shellQuote(a))
	}
	return strings.Join(parts, " ")
}

func sortedHeaderNames(h map[string][]string) []string {
	names := make([]string, 0, len(h))
	for name := range h {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// shellQuote single-quotes a string for safe pasting into a POSIX shell, but
// leaves already-safe tokens unquoted for readability.
func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	if !strings.ContainsAny(s, " \t\n\r\"'\\$`&|;<>()*?[]{}#~!") {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
