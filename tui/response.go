package tui

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/jefflinse/clic/provider"
)

// respTab selects which view of a result the response pane shows.
type respTab int

const (
	tabPretty respTab = iota
	tabHeaders
	tabRaw
)

// responsePane renders the outcome of the last execution: a summary line (status
// badge, latency, size) above a scrollable body shown as pretty/highlighted
// JSON, raw text, or response headers.
type responsePane struct {
	vp     viewport.Model
	th     theme
	result *provider.Result
	err    error
	tab    respTab
	width  int
	height int
}

func newResponsePane(th theme) responsePane {
	return responsePane{vp: viewport.New(0, 0), th: th}
}

func (r *responsePane) setSize(w, h int) {
	r.width, r.height = w, h
	r.vp.Width = w
	r.vp.Height = h - 1 // reserve a row for the summary line
	r.reflow()
}

func (r *responsePane) setResult(res *provider.Result) {
	r.result, r.err = res, nil
	if res != nil && res.Kind == provider.ResultHTTP {
		r.tab = tabPretty
	}
	r.vp.GotoTop()
	r.reflow()
}

func (r *responsePane) setError(err error) {
	r.result, r.err = nil, err
	r.reflow()
}

func (r *responsePane) cycleTab() {
	if r.result == nil {
		return
	}
	r.tab = (r.tab + 1) % 3
	r.vp.GotoTop()
	r.reflow()
}

func (r *responsePane) update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	r.vp, cmd = r.vp.Update(msg)
	return cmd
}

// reflow regenerates the viewport content for the current result and tab.
func (r *responsePane) reflow() {
	r.vp.SetContent(r.body())
}

func (r *responsePane) body() string {
	switch {
	case r.err != nil:
		return r.th.statusStyle(0).Render(" ERR ") + " " + r.th.json.str.Render(r.err.Error())
	case r.result == nil:
		return r.th.desc.Render("Fill in the request and press ctrl+s to send.")
	}

	switch r.tab {
	case tabHeaders:
		return r.renderHeaders()
	case tabRaw:
		return string(r.result.Body)
	default:
		return r.renderBody()
	}
}

func (r *responsePane) renderBody() string {
	body := r.result.Body
	if len(body) == 0 {
		return r.th.desc.Render("(empty response)")
	}
	// only attempt syntax highlighting for reasonably-sized JSON
	if len(body) <= 512*1024 {
		if pretty, ok := highlightJSON(body, r.th.json); ok {
			return pretty
		}
	}
	return string(body)
}

func (r *responsePane) renderHeaders() string {
	if r.result.Headers == nil {
		return r.th.desc.Render("(no headers)")
	}
	names := make([]string, 0, len(r.result.Headers))
	for name := range r.result.Headers {
		names = append(names, name)
	}
	sort.Strings(names)

	var b strings.Builder
	for _, name := range names {
		value := strings.Join(r.result.Headers[name], ", ")
		b.WriteString(r.th.hdrKey.Render(name))
		b.WriteString(r.th.json.punct.Render(": "))
		b.WriteString(r.th.hdrVal.Render(value))
		b.WriteByte('\n')
	}
	return strings.TrimRight(b.String(), "\n")
}

// summary renders the one-line status/latency/size/tab strip above the body.
func (r *responsePane) summary() string {
	if r.err != nil {
		return r.th.statusStyle(0).Render(" FAILED ")
	}
	if r.result == nil {
		return r.th.desc.Render("response")
	}

	parts := []string{statusBadge(r.th, r.result)}
	if r.result.Latency > 0 {
		parts = append(parts, r.th.latency.Render(humanizeDuration(r.result.Latency)))
	}
	parts = append(parts, r.th.size.Render(humanizeBytes(len(r.result.Body))))
	if r.result.ContentType != "" {
		parts = append(parts, r.th.size.Render(shortContentType(r.result.ContentType)))
	}
	parts = append(parts, r.tabsStrip())
	return strings.Join(parts, r.th.json.punct.Render(" · "))
}

func (r *responsePane) tabsStrip() string {
	labels := []string{"pretty", "headers", "raw"}
	rendered := make([]string, len(labels))
	for i, label := range labels {
		if respTab(i) == r.tab {
			rendered[i] = r.th.helpKey.Render(label)
		} else {
			rendered[i] = r.th.desc.Render(label)
		}
	}
	return strings.Join(rendered, r.th.desc.Render("/"))
}

func statusBadge(th theme, res *provider.Result) string {
	if res.Kind == provider.ResultHTTP {
		text := fmt.Sprintf(" %d %s ", res.Status, http.StatusText(res.Status))
		return th.statusStyle(res.Status).Render(strings.TrimRight(text, " ") + " ")
	}
	// text results: show an exit-code badge
	label := " OK "
	code := 0
	if res.Status != 0 {
		label = fmt.Sprintf(" EXIT %d ", res.Status)
		code = 500
	}
	return th.statusStyle(code).Render(label)
}

func humanizeDuration(d time.Duration) string {
	switch {
	case d < time.Microsecond:
		return fmt.Sprintf("%dns", d.Nanoseconds())
	case d < time.Millisecond:
		return fmt.Sprintf("%.1fµs", float64(d.Nanoseconds())/1e3)
	case d < time.Second:
		return fmt.Sprintf("%dms", d.Milliseconds())
	default:
		return fmt.Sprintf("%.2fs", d.Seconds())
	}
}

func humanizeBytes(n int) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%dB", n)
	}
	div, exp := int64(unit), 0
	for size := int64(n) / unit; size >= unit; size /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%cB", float64(n)/float64(div), "KMGT"[exp])
}

func shortContentType(ct string) string {
	if i := strings.IndexByte(ct, ';'); i >= 0 {
		ct = ct[:i]
	}
	return strings.TrimSpace(ct)
}
