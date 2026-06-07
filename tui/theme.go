package tui

import (
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

// palette is clic's studio color scheme: a modern, high-contrast set tuned to
// look good on dark terminals while staying legible on light ones via adaptive
// colors.
type palette struct {
	accent  lipgloss.AdaptiveColor // focus / primary actions
	accent2 lipgloss.AdaptiveColor // secondary highlights
	success lipgloss.AdaptiveColor // 2xx, ok
	warn    lipgloss.AdaptiveColor // 3xx/4xx
	danger  lipgloss.AdaptiveColor // 5xx / errors
	text    lipgloss.AdaptiveColor // primary text
	muted   lipgloss.AdaptiveColor // secondary text, hints
	faint   lipgloss.AdaptiveColor // borders, separators
	bgSel   lipgloss.AdaptiveColor // selected-row background
}

func defaultPalette() palette {
	return palette{
		accent:  lipgloss.AdaptiveColor{Light: "#7C3AED", Dark: "#C792EA"},
		accent2: lipgloss.AdaptiveColor{Light: "#0891B2", Dark: "#89DDFF"},
		success: lipgloss.AdaptiveColor{Light: "#16A34A", Dark: "#9ECE6A"},
		warn:    lipgloss.AdaptiveColor{Light: "#CA8A04", Dark: "#E0AF68"},
		danger:  lipgloss.AdaptiveColor{Light: "#DC2626", Dark: "#F7768E"},
		text:    lipgloss.AdaptiveColor{Light: "#1E293B", Dark: "#C0CAF5"},
		muted:   lipgloss.AdaptiveColor{Light: "#64748B", Dark: "#7A88CF"},
		faint:   lipgloss.AdaptiveColor{Light: "#CBD5E1", Dark: "#3B4261"},
		bgSel:   lipgloss.AdaptiveColor{Light: "#EDE9FE", Dark: "#2A2F45"},
	}
}

// theme bundles the lipgloss styles the studio renders with, derived from a
// palette so the whole UI re-themes from one place.
type theme struct {
	p palette

	// chrome
	title    lipgloss.Style // app name in the top bar
	subtitle lipgloss.Style // app description
	server   lipgloss.Style // server pill in the top bar
	help     lipgloss.Style // bottom hint bar
	helpKey  lipgloss.Style // a key glyph within the hint bar

	// panes
	paneTitle    lipgloss.Style // a pane header ("GROUPS", "REQUEST", …)
	paneTitleHot lipgloss.Style // header of the focused pane
	border       lipgloss.Style // unfocused pane border
	borderHot    lipgloss.Style // focused pane border

	// list rows
	group    lipgloss.Style // a group (has children)
	leaf     lipgloss.Style // a runnable command
	selected lipgloss.Style // the highlighted row in the focused pane
	selDim   lipgloss.Style // the highlighted row in an unfocused pane
	desc     lipgloss.Style // a row's description / secondary text

	// response
	latency     lipgloss.Style
	size        lipgloss.Style
	hdrKey      lipgloss.Style
	hdrVal      lipgloss.Style
	match       lipgloss.Style // a search hit within the response body
	contractOK  lipgloss.Style // "conforms" contract chip
	contractBad lipgloss.Style // "violations" contract chip

	// json
	json jsonStyles
}

func newTheme() theme {
	p := defaultPalette()
	base := lipgloss.NewStyle()

	return theme{
		p:        p,
		title:    base.Bold(true).Foreground(p.accent),
		subtitle: base.Foreground(p.muted),
		server:   base.Foreground(p.accent2).Bold(true),
		help:     base.Foreground(p.muted),
		helpKey:  base.Foreground(p.accent).Bold(true),

		paneTitle:    base.Foreground(p.muted).Bold(true),
		paneTitleHot: base.Foreground(p.accent).Bold(true),
		border:       base.Border(lipgloss.RoundedBorder()).BorderForeground(p.faint),
		borderHot:    base.Border(lipgloss.RoundedBorder()).BorderForeground(p.accent),

		group:    base.Foreground(p.text),
		leaf:     base.Foreground(p.text),
		selected: base.Bold(true).Foreground(p.accent).Background(p.bgSel),
		selDim:   base.Foreground(p.text).Background(p.bgSel),
		desc:     base.Foreground(p.muted),

		latency:     base.Foreground(p.accent2),
		size:        base.Foreground(p.muted),
		hdrKey:      base.Foreground(p.accent2),
		hdrVal:      base.Foreground(p.text),
		match:       base.Bold(true).Foreground(lipgloss.Color("#0B1020")).Background(p.accent2),
		contractOK:  base.Bold(true).Padding(0, 1).Foreground(lipgloss.Color("#0B1020")).Background(p.success),
		contractBad: base.Bold(true).Padding(0, 1).Foreground(lipgloss.Color("#0B1020")).Background(p.warn),

		json: jsonStyles{
			key:     base.Foreground(p.accent2),
			str:     base.Foreground(p.success),
			num:     base.Foreground(p.warn),
			boolean: base.Foreground(p.accent),
			null:    base.Foreground(p.danger),
			punct:   base.Foreground(p.muted),
		},
	}
}

// statusStyle colors an HTTP status code by class: 2xx success, 3xx/4xx
// warning, 5xx and transport errors danger.
func (t theme) statusStyle(code int) lipgloss.Style {
	base := lipgloss.NewStyle().Bold(true).Padding(0, 1)
	switch {
	case code >= 200 && code < 300:
		return base.Foreground(lipgloss.Color("#0B1020")).Background(t.p.success)
	case code >= 300 && code < 500:
		return base.Foreground(lipgloss.Color("#0B1020")).Background(t.p.warn)
	default:
		return base.Foreground(lipgloss.Color("#0B1020")).Background(t.p.danger)
	}
}

// huhTheme adapts the studio palette into a huh form theme so the embedded
// request form matches the surrounding chrome.
func (t theme) huhTheme() *huh.Theme {
	ht := huh.ThemeBase()
	ht.Focused.Base = ht.Focused.Base.BorderForeground(t.p.faint)
	ht.Focused.Title = ht.Focused.Title.Foreground(t.p.accent).Bold(true)
	ht.Focused.Description = ht.Focused.Description.Foreground(t.p.muted)
	ht.Focused.SelectedOption = ht.Focused.SelectedOption.Foreground(t.p.accent)
	ht.Focused.SelectSelector = ht.Focused.SelectSelector.Foreground(t.p.accent)
	ht.Focused.TextInput.Cursor = ht.Focused.TextInput.Cursor.Foreground(t.p.accent)
	ht.Focused.TextInput.Prompt = ht.Focused.TextInput.Prompt.Foreground(t.p.accent2)
	ht.Focused.TextInput.Placeholder = ht.Focused.TextInput.Placeholder.Foreground(t.p.faint)
	ht.Focused.FocusedButton = ht.Focused.FocusedButton.Background(t.p.accent)
	ht.Blurred = ht.Focused
	ht.Blurred.Base = ht.Blurred.Base.BorderStyle(lipgloss.HiddenBorder())
	return ht
}
