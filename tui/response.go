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
	tabRequest
)

// respTabCount is the number of cyclable response views.
const respTabCount = 4

// searchState backs the in-pane incremental search ('/'): the query, the line
// indices that contain a hit, and which hit is currently centered.
type searchState struct {
	query string
	lines []int
	cur   int
}

// filterState backs the jq filter ('f'): the program, its evaluated output, and
// any parse/run error. An empty program means no filter is active.
type filterState struct {
	program string
	out     []byte
	err     error
}

// responsePane renders the outcome of the last execution: a summary line (status
// badge, latency, size) above a scrollable body shown as pretty/highlighted
// JSON, raw text, or response headers. A jq filter can transform the body in
// place, and an incremental search highlights and jumps between hits.
type responsePane struct {
	vp      viewport.Model
	th      theme
	result  *provider.Result
	preview *provider.RequestPreview
	err     error
	tab     respTab
	search  searchState
	filter  filterState
	width   int
	height  int
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
	// a fresh result invalidates any search/filter from the previous one
	r.search = searchState{}
	r.filter = filterState{}
	r.vp.GotoTop()
	r.reflow()
}

func (r *responsePane) setError(err error) {
	r.result, r.err = nil, err
	r.reflow()
}

// setPreview stores the live request preview. It is shown whenever no result is
// displayed yet, and via the "request" tab once a result is present.
func (r *responsePane) setPreview(pv *provider.RequestPreview) {
	r.preview = pv
	if r.result == nil && r.err == nil {
		r.reflow()
	}
}

// cycleTab moves the response view forward (dir +1) or backward (dir -1)
// through the pretty/headers/raw/request tabs.
func (r *responsePane) cycleTab(dir int) {
	if r.result == nil {
		return
	}
	r.tab = respTab((int(r.tab) + dir + respTabCount) % respTabCount)
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
		// before the first send, the pane previews what ctrl+s will send
		return r.renderPreview()
	}

	// an active search renders the body as plain text with hits highlighted, so
	// matches are visible regardless of JSON coloring and align with the line
	// indices used for jumping.
	if r.search.query != "" {
		return r.renderSearch()
	}

	switch r.tab {
	case tabHeaders:
		return r.renderHeaders()
	case tabRaw:
		return string(r.sourceBytes())
	case tabRequest:
		return r.renderPreview()
	default:
		return r.renderBody()
	}
}

// sourceBytes is the body the views operate on: the jq-filtered output when a
// filter is active and valid, otherwise the raw response body.
func (r *responsePane) sourceBytes() []byte {
	if r.filter.program != "" && r.filter.err == nil {
		return r.filter.out
	}
	return r.result.Body
}

func (r *responsePane) renderBody() string {
	body := r.sourceBytes()
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

// searchText is the plain-text the incremental search runs over: pretty-printed
// JSON when the body parses, otherwise the body verbatim. It is independent of
// the active tab so '/' always searches the response payload.
func (r *responsePane) searchText() string {
	src := r.sourceBytes()
	if pretty, ok := prettyPlainJSON(src); ok {
		return pretty
	}
	return string(src)
}

// renderSearch returns the search text with every hit wrapped in the match
// style.
func (r *responsePane) renderSearch() string {
	lines := strings.Split(r.searchText(), "\n")
	for i, line := range lines {
		if hl, ok := highlightMatches(line, r.search.query, r.th.match); ok {
			lines[i] = hl
		}
	}
	return strings.Join(lines, "\n")
}

// setSearch applies a query, recording the lines that contain a hit and jumping
// the viewport to the first one. An empty query clears the search.
func (r *responsePane) setSearch(query string) {
	r.search.query = query
	r.search.lines = nil
	r.search.cur = 0
	if query != "" {
		needle := strings.ToLower(query)
		for i, line := range strings.Split(r.searchText(), "\n") {
			if strings.Contains(strings.ToLower(line), needle) {
				r.search.lines = append(r.search.lines, i)
			}
		}
	}
	r.reflow()
	r.scrollToMatch()
}

// nextMatch moves to the next (dir +1) or previous (dir -1) hit, wrapping.
func (r *responsePane) nextMatch(dir int) {
	if len(r.search.lines) == 0 {
		return
	}
	r.search.cur = (r.search.cur + dir + len(r.search.lines)) % len(r.search.lines)
	r.scrollToMatch()
}

func (r *responsePane) scrollToMatch() {
	if len(r.search.lines) == 0 {
		return
	}
	r.vp.SetYOffset(max(0, r.search.lines[r.search.cur]-1))
}

func (r *responsePane) clearSearch() {
	r.search = searchState{}
	r.reflow()
}

// applyFilter runs a jq program over the response body and shows the result in
// place. An empty program clears the filter; a parse/run error is retained and
// surfaced in the summary line while the body keeps showing the raw response.
func (r *responsePane) applyFilter(program string) {
	r.filter = filterState{program: program}
	if program != "" {
		out, err := runJQ(program, r.result.Body)
		if err != nil {
			r.filter.err = err
		} else {
			r.filter.out = out
		}
	}
	r.search = searchState{} // line indices no longer align with the new body
	r.vp.GotoTop()
	r.reflow()
}

func (r *responsePane) clearFilter() {
	r.filter = filterState{}
	r.vp.GotoTop()
	r.reflow()
}

// clearActive dismisses an active search or filter (search first), reporting
// whether anything was cleared. It backs esc's "peel one layer" behavior.
func (r *responsePane) clearActive() bool {
	switch {
	case r.search.query != "":
		r.clearSearch()
		return true
	case r.filter.program != "":
		r.clearFilter()
		return true
	}
	return false
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
		if r.preview == nil {
			return r.th.desc.Render("response")
		}
		return r.th.paneTitle.Render("▲ REQUEST PREVIEW") +
			r.th.json.punct.Render(" · ") +
			r.th.desc.Render("ctrl+s sends")
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
	if note := r.activeNote(); note != "" {
		parts = append(parts, note)
	}
	return strings.Join(parts, r.th.json.punct.Render(" · "))
}

func (r *responsePane) tabsStrip() string {
	labels := []string{"pretty", "headers", "raw", "request"}
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

// activeNote summarizes any active jq filter or search for the summary line: the
// jq program (or its error) and the current hit position.
func (r *responsePane) activeNote() string {
	var notes []string
	if r.filter.program != "" {
		if r.filter.err != nil {
			notes = append(notes, r.th.statusStyle(0).Render(" jq ")+" "+r.th.desc.Render(oneLine(r.filter.err.Error())))
		} else {
			notes = append(notes, r.th.helpKey.Render("jq ▸ ")+r.th.hdrVal.Render(r.filter.program))
		}
	}
	if r.search.query != "" {
		pos := "0/0"
		if n := len(r.search.lines); n > 0 {
			pos = fmt.Sprintf("%d/%d", r.search.cur+1, n)
		}
		notes = append(notes, r.th.helpKey.Render("/"+r.search.query)+" "+r.th.desc.Render(pos))
	}
	return strings.Join(notes, r.th.json.punct.Render(" · "))
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
