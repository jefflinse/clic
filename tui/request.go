package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/jefflinse/clic/provider"
)

// requestForm wraps an embedded huh form built from a command's input sections.
// Bindings are created once (so typed values survive a rebuild) while the huh
// form itself is rebuilt on demand — after a send, or on resize — which resets
// its completion state without discarding what the user entered.
type requestForm struct {
	sections []provider.Section
	binds    map[string][]*binding // section key -> field bindings
	raw      map[string]*string    // section key -> raw-body holder
	th       theme
	form     *huh.Form // nil when the command takes no input
}

// newRequestForm builds the bindings for a command's sections and an initial
// huh form over them.
func newRequestForm(sections []provider.Section, th theme) *requestForm {
	rf := &requestForm{
		sections: sections,
		binds:    map[string][]*binding{},
		raw:      map[string]*string{},
		th:       th,
	}
	for _, sec := range sections {
		if sec.Raw {
			rf.raw[sec.Key] = new(string)
			continue
		}
		rf.binds[sec.Key] = newBindings(sec.Fields)
	}
	rf.rebuild()
	return rf
}

// hasInputs reports whether the command exposes any fields to fill in.
func (rf *requestForm) hasInputs() bool {
	return rf.form != nil
}

// setSize resizes the embedded form to fit its pane.
func (rf *requestForm) setSize(w, h int) {
	if rf.form != nil && w > 0 && h > 0 {
		rf.form = rf.form.WithWidth(w).WithHeight(h)
	}
}

// asHuhForm recovers the *huh.Form from the tea.Model returned by Form.Update,
// falling back to the previous form if the assertion ever fails.
func asHuhForm(m tea.Model, fallback *huh.Form) *huh.Form {
	if f, ok := m.(*huh.Form); ok {
		return f
	}
	return fallback
}

// rebuild constructs a fresh huh form over the existing bindings and raw
// holders, preserving entered values.
func (rf *requestForm) rebuild() {
	var fields []huh.Field
	for _, sec := range rf.sections {
		fields = append(fields, huh.NewNote().Title(sectionHeading(sec)))
		if sec.Raw {
			fields = append(fields, huh.NewText().
				Title("raw body").
				Description("inline JSON or @file").
				Value(rf.raw[sec.Key]))
			continue
		}
		for _, b := range rf.binds[sec.Key] {
			fields = append(fields, b.studioInputs("")...)
		}
	}

	if len(fields) == 0 {
		rf.form = nil
		return
	}

	rf.form = huh.NewForm(huh.NewGroup(fields...)).
		WithTheme(rf.th.huhTheme()).
		WithShowHelp(false).
		WithShowErrors(true)
	rf.form.Init()
}

// collect assembles everything the user entered into provider.Inputs ready for
// execution.
func (rf *requestForm) collect() provider.Inputs {
	in := provider.Inputs{Scalars: map[string]map[string]any{}}

	if holder, ok := rf.raw["body"]; ok {
		in.RawBody = strings.TrimSpace(*holder)
	}

	for key, binds := range rf.binds {
		hydrateComplexArrays(binds)
		assembled := assemble(binds)
		if key == "body" {
			in.Body = assembled
		} else {
			in.Scalars[key] = assembled
		}
	}

	return in
}

// sectionHeading renders a section's title as an uppercase heading for the form.
func sectionHeading(sec provider.Section) string {
	return strings.ToUpper(sec.Title)
}
