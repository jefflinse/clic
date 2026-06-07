package tui

import (
	"context"
	"fmt"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jefflinse/clic/provider"
)

// Command is the studio's view of one node in an app's command tree: a runnable
// leaf (Provider set) or a group (Subcommands set). It deliberately mirrors only
// what the studio needs, so the tui package depends on provider but not on spec.
type Command struct {
	Name        string
	Description string
	Provider    provider.Provider
	Subcommands []Command
}

// StudioApp is the input to RunStudio: an app's identity plus its command tree.
type StudioApp struct {
	Name        string
	Description string
	Server      string
	// Invocation is the headless launch prefix (e.g. "clic ./petstore.yaml")
	// used to render "copy as clic command".
	Invocation string
	Commands   []Command
}

// RunStudio launches the full-screen studio for the given app. When selected is
// non-empty it names a command path (e.g. ["pets","getById"]) to pre-select and
// focus on launch. The context carries clic's options and auth for execution.
func RunStudio(ctx context.Context, app StudioApp, selected []string) error {
	m := newStudio(ctx, app)
	m.preselect(selected)
	_, err := tea.NewProgram(m, tea.WithAltScreen(), tea.WithContext(ctx)).Run()
	return err
}

// focusZone identifies which pane currently receives input.
type focusZone int

const (
	focusGroups focusZone = iota
	focusCommands
	focusRequest
	focusResponse
)

// entry is a row in the commands column: a runnable leaf (cmd set) or a
// non-selectable group header (cmd nil), indented by depth. path is the full
// "group / sub / leaf" trail, used by the command palette.
type entry struct {
	label string
	path  string
	depth int
	cmd   *Command
}

type resultMsg struct {
	res *provider.Result
	err error
}

type studio struct {
	ctx context.Context
	app StudioApp
	th  theme

	groups   []Command
	groupIdx int

	entries    []entry
	commandIdx int
	leaf       *Command

	req  *requestForm
	resp responsePane
	spin spinner.Model

	focus   focusZone
	sending bool

	// captured variables for request chaining ({{name}} references)
	vars []variable

	// overlays and transient UI
	helpOpen bool
	varsOpen bool
	pal      *paletteState
	copy     *copyMenu
	cap      *captureState
	editing  *editState // inline search / jq-filter input bar
	flash    string

	width, height int

	// layout, recomputed on resize
	gW, cW, rW int
	colsH      int // outer height of the column panes
	respH      int // outer height of the response pane
}

func newStudio(ctx context.Context, app StudioApp) *studio {
	th := newTheme()
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(th.p.accent)

	s := &studio{
		ctx:    ctx,
		app:    app,
		th:     th,
		groups: app.Commands,
		resp:   newResponsePane(th),
		spin:   sp,
	}
	s.syncEntries()
	s.selectLeafFromEntries()
	return s
}

func (s *studio) Init() tea.Cmd {
	return s.spin.Tick
}

// ---- tree navigation -------------------------------------------------------

// syncEntries rebuilds the commands column for the highlighted group.
func (s *studio) syncEntries() {
	s.entries = nil
	if len(s.groups) == 0 {
		return
	}
	s.entries = entriesFor(&s.groups[s.groupIdx])
	if s.commandIdx >= len(s.entries) {
		s.commandIdx = len(s.entries) - 1
	}
	s.snapToSelectable(1)
}

// entriesFor flattens a group into the rows shown in the commands column. A
// group with subcommands expands to its descendants (headers + leaves); a
// runnable top-level command is its own single selectable row.
func entriesFor(g *Command) []entry {
	if len(g.Subcommands) > 0 {
		return flattenChildren(g.Subcommands, 0, g.Name)
	}
	return []entry{{label: g.Name, path: g.Name, depth: 0, cmd: g}}
}

func flattenChildren(cmds []Command, depth int, parentPath string) []entry {
	var out []entry
	for i := range cmds {
		c := &cmds[i]
		path := c.Name
		if parentPath != "" {
			path = parentPath + " / " + c.Name
		}
		if len(c.Subcommands) > 0 {
			out = append(out, entry{label: c.Name, path: path, depth: depth})
			out = append(out, flattenChildren(c.Subcommands, depth+1, path)...)
		} else {
			out = append(out, entry{label: c.Name, path: path, depth: depth, cmd: c})
		}
	}
	return out
}

// snapToSelectable moves commandIdx onto the nearest selectable (leaf) entry in
// the given direction (+1 down, -1 up), so group headers are skipped.
func (s *studio) snapToSelectable(dir int) {
	if len(s.entries) == 0 {
		s.commandIdx = 0
		return
	}
	for i := 0; i < len(s.entries); i++ {
		idx := s.commandIdx + i*dir
		if idx >= 0 && idx < len(s.entries) && s.entries[idx].cmd != nil {
			s.commandIdx = idx
			return
		}
	}
	// nothing in that direction; try the other way
	for i := range s.entries {
		if s.entries[i].cmd != nil {
			s.commandIdx = i
			return
		}
	}
}

// selectLeafFromEntries points leaf at the highlighted entry and (re)builds its
// request form, returning a cmd to initialize the form.
func (s *studio) selectLeafFromEntries() tea.Cmd {
	var leaf *Command
	if s.commandIdx >= 0 && s.commandIdx < len(s.entries) {
		leaf = s.entries[s.commandIdx].cmd
	}
	if leaf == s.leaf && s.req != nil {
		return nil
	}
	s.leaf = leaf
	s.req = newRequestForm(providerSections(leaf), s.th)
	s.resp.setResult(nil)
	s.refreshPreview()
	s.relayout()
	if s.req.form != nil {
		return s.req.form.Init()
	}
	return nil
}

// currentInputs collects everything entered in the request form, with chained
// variables substituted, ready for preview or execution.
func (s *studio) currentInputs() provider.Inputs {
	if s.req == nil {
		return provider.Inputs{}
	}
	return s.applyVars(s.req.collect())
}

// refreshPreview recomputes the live request preview for the selected command,
// so the response pane can show exactly what ctrl+s will send.
func (s *studio) refreshPreview() {
	pv, ok := s.previewer()
	if !ok {
		s.resp.setPreview(nil)
		return
	}
	preview, err := pv.Preview(s.ctx, s.currentInputs())
	if err != nil {
		s.resp.setPreview(nil)
		return
	}
	s.resp.setPreview(preview)
}

// previewer returns the selected leaf's Previewer, if it implements one.
func (s *studio) previewer() (provider.Previewer, bool) {
	if s.leaf == nil {
		return nil, false
	}
	pv, ok := s.leaf.Provider.(provider.Previewer)
	return pv, ok
}

func providerSections(leaf *Command) []provider.Section {
	if leaf == nil {
		return nil
	}
	if iv, ok := leaf.Provider.(provider.Interactive); ok {
		return iv.Sections()
	}
	return nil
}

// preselect points the studio at a command named by path and focuses its form.
func (s *studio) preselect(path []string) {
	if len(path) == 0 {
		return
	}
	for i := range s.groups {
		if s.groups[i].Name != path[0] {
			continue
		}
		s.groupIdx = i
		s.syncEntries()
		if len(path) > 1 {
			for j := range s.entries {
				if s.entries[j].cmd != nil && s.entries[j].label == path[len(path)-1] {
					s.commandIdx = j
					break
				}
			}
		}
		s.selectLeafFromEntries()
		if s.req != nil && s.req.hasInputs() {
			s.setFocus(focusRequest)
		} else {
			s.setFocus(focusCommands)
		}
		return
	}
}

// ---- update ----------------------------------------------------------------

func (s *studio) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		s.width, s.height = msg.Width, msg.Height
		s.relayout()
		return s, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		s.spin, cmd = s.spin.Update(msg)
		return s, cmd

	case resultMsg:
		s.sending = false
		if msg.err != nil {
			s.resp.setError(msg.err)
		} else {
			s.resp.setResult(msg.res)
		}
		s.focus = focusResponse
		return s, nil

	case editorFinishedMsg:
		if msg.err != nil {
			s.flash = "editor: " + oneLine(msg.err.Error())
		}
		return s, nil

	case tea.KeyMsg:
		return s.handleKey(msg)
	}

	// forward other messages (cursor blink, etc.) to the active sub-model
	return s, s.forward(msg)
}

func (s *studio) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	s.flash = "" // any keypress clears a transient flash message

	// overlays capture input while open
	if s.helpOpen {
		s.helpOpen = false
		return s, nil
	}
	if s.pal != nil {
		return s, s.handlePaletteKey(msg)
	}
	if s.copy != nil {
		return s, s.handleCopyKey(msg)
	}
	if s.cap != nil {
		return s, s.handleCaptureKey(msg)
	}
	if s.varsOpen {
		s.varsOpen = false
		return s, nil
	}
	// the inline search / jq-filter bar captures all input while open
	if s.editing != nil {
		return s, s.handleEditKey(msg)
	}

	switch msg.String() {
	case "ctrl+c":
		return s, tea.Quit
	case "ctrl+s":
		return s, s.send()
	case "ctrl+p":
		s.openPalette()
		return s, nil
	}

	// these keys are reserved outside the request form, where they're text input
	if s.focus != focusRequest {
		switch msg.String() {
		case "?":
			s.helpOpen = true
			return s, nil
		case "/":
			// in the response pane, '/' searches the body; elsewhere it opens the
			// command palette (ctrl+p remains a palette alias everywhere).
			if s.focus == focusResponse {
				s.startSearch()
			} else {
				s.openPalette()
			}
			return s, nil
		case "c":
			s.openCopyMenu()
			return s, nil
		case "x":
			s.openCapture()
			return s, nil
		case "v":
			s.varsOpen = true
			return s, nil
		}
	}

	// tab/shift+tab cycle the focus ring consistently from every pane; esc steps
	// back toward the command tree. In the request form, tab/shift+tab navigate
	// fields first and only cross pane boundaries at the form's edges, so they
	// fall through to handleRequestKey.
	switch msg.String() {
	case "tab":
		if s.focus != focusRequest {
			return s, s.cycleFocus(1)
		}
	case "shift+tab":
		if s.focus != focusRequest {
			return s, s.cycleFocus(-1)
		}
	case "esc":
		// in the response pane, esc peels off an active search/filter before
		// stepping back toward the command tree.
		if s.focus == focusResponse && s.resp.clearActive() {
			return s, nil
		}
		return s, s.stepBack()
	}

	switch s.focus {
	case focusGroups:
		return s, s.handleGroupsKey(msg)
	case focusCommands:
		return s, s.handleCommandsKey(msg)
	case focusRequest:
		return s, s.handleRequestKey(msg)
	case focusResponse:
		return s, s.handleResponseKey(msg)
	}
	return s, nil
}

// focusRing is the order tab/shift+tab cycle through and esc steps back along.
var focusRing = []focusZone{focusGroups, focusCommands, focusRequest, focusResponse}

// setFocus is the single mutator for the focused pane. The embedded form keeps
// its own field position (like any form remembers your cursor), so re-entering
// the request pane lands where you left it; tab/shift+tab carry you out at the
// form's edges.
func (s *studio) setFocus(z focusZone) {
	s.focus = z
}

// cycleFocus moves focus forward (dir +1) or backward (dir -1) through the ring,
// wrapping at the ends.
func (s *studio) cycleFocus(dir int) tea.Cmd {
	cur := 0
	for i, z := range focusRing {
		if z == s.focus {
			cur = i
			break
		}
	}
	s.setFocus(focusRing[(cur+dir+len(focusRing))%len(focusRing)])
	return nil
}

// stepBack moves focus one pane toward the command tree, stopping at Groups (no
// wrap). It is the universal "back out" gesture, bound to esc.
func (s *studio) stepBack() tea.Cmd {
	switch s.focus {
	case focusCommands:
		s.setFocus(focusGroups)
	case focusRequest:
		s.setFocus(focusCommands)
	case focusResponse:
		s.setFocus(focusRequest)
	}
	return nil
}

func (s *studio) handleGroupsKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "up", "k":
		if s.groupIdx > 0 {
			s.groupIdx--
			s.syncEntries()
			return s.selectLeafFromEntries()
		}
	case "down", "j":
		if s.groupIdx < len(s.groups)-1 {
			s.groupIdx++
			s.syncEntries()
			return s.selectLeafFromEntries()
		}
	case "right", "l", "enter":
		s.setFocus(focusCommands)
	}
	return nil
}

func (s *studio) handleCommandsKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "up", "k":
		s.moveCommand(-1)
		return s.selectLeafFromEntries()
	case "down", "j":
		s.moveCommand(1)
		return s.selectLeafFromEntries()
	case "left", "h":
		s.setFocus(focusGroups)
	case "right", "l":
		s.setFocus(focusRequest)
	case "enter":
		// enter is "do the thing": edit a command that has inputs, run one that
		// doesn't.
		if s.req != nil && s.req.hasInputs() {
			s.setFocus(focusRequest)
		} else {
			return s.send()
		}
	}
	return nil
}

// moveCommand advances the highlighted leaf in the given direction, skipping
// group headers.
func (s *studio) moveCommand(dir int) {
	for idx := s.commandIdx + dir; idx >= 0 && idx < len(s.entries); idx += dir {
		if s.entries[idx].cmd != nil {
			s.commandIdx = idx
			return
		}
	}
}

func (s *studio) handleRequestKey(msg tea.KeyMsg) tea.Cmd {
	// a command with no inputs is still a focus stop: it participates in the ring
	// and offers a spatial path to the response.
	if s.req == nil || s.req.form == nil {
		switch msg.String() {
		case "tab":
			return s.cycleFocus(1)
		case "shift+tab":
			return s.cycleFocus(-1)
		case "left", "h":
			s.setFocus(focusCommands)
		case "right", "l":
			s.setFocus(focusResponse)
		case "enter":
			return s.send()
		}
		return nil
	}

	// Embedded huh disables tab on the last field and shift+tab on the first, so
	// the form never moves past its own edges — it returns no command there. We
	// use that nil command as the signal to cross the pane boundary: tab off the
	// last field flows forward to the response, shift+tab off the first flows
	// back to commands. Mid-form, navigation returns a command and we stay put.
	switch msg.String() {
	case "tab":
		// A non-nil command means huh moved between fields, so stay. A nil command
		// means tab did nothing: either a required field blocked navigation (huh
		// now shows its error — stay so the user can fix it) or we are genuinely on
		// the last field, where we carry focus forward into the response.
		if cmd := s.updateForm(msg); cmd != nil {
			return cmd
		}
		if len(s.req.form.Errors()) > 0 {
			return nil
		}
		s.setFocus(focusResponse)
		return nil
	case "shift+tab":
		// shift+tab never validates; a nil command means we are on the first field,
		// so step back to commands. Otherwise huh moved between fields.
		if cmd := s.updateForm(msg); cmd != nil {
			return cmd
		}
		s.setFocus(focusCommands)
		return nil
	}
	return s.updateForm(msg)
}

// updateForm feeds a message to the embedded huh form and refreshes the live
// preview on edits. It is the single chokepoint for form updates so async huh
// messages (field transitions, cursor blink) are routed consistently.
func (s *studio) updateForm(msg tea.Msg) tea.Cmd {
	if s.req == nil || s.req.form == nil {
		return nil
	}
	form, cmd := s.req.form.Update(msg)
	s.req.form = asHuhForm(form, s.req.form)
	if _, ok := msg.(tea.KeyMsg); ok {
		s.refreshPreview()
	}
	return cmd
}

func (s *studio) handleResponseKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "left", "h":
		s.resp.cycleTab(-1)
		return nil
	case "right", "l":
		s.resp.cycleTab(1)
		return nil
	case "f":
		s.startFilter()
		return nil
	case "n":
		s.resp.nextMatch(1)
		return nil
	case "N":
		s.resp.nextMatch(-1)
		return nil
	case "o":
		return s.openInEditor()
	case "y":
		s.yankResponse()
		return nil
	}
	return s.resp.update(msg)
}

// yankResponse copies the current response body to the system clipboard.
func (s *studio) yankResponse() {
	if s.resp.result == nil {
		return
	}
	if err := clipboard.WriteAll(string(s.resp.result.Body)); err != nil {
		s.flash = "clipboard unavailable"
		return
	}
	s.flash = "copied response to clipboard"
}

// forward routes a non-key message (cursor blink, huh's async group transitions,
// etc.) to whichever sub-model is active. Request updates go through updateForm
// so a completion delivered by one of those async messages still carries focus
// to the response.
func (s *studio) forward(msg tea.Msg) tea.Cmd {
	switch s.focus {
	case focusRequest:
		return s.updateForm(msg)
	case focusResponse:
		return s.resp.update(msg)
	}
	return nil
}

func (s *studio) send() tea.Cmd {
	if s.sending || s.leaf == nil {
		return nil
	}
	iv, ok := s.leaf.Provider.(provider.Interactive)
	if !ok {
		name := s.leaf.Name
		return func() tea.Msg {
			return resultMsg{err: fmt.Errorf("command %q cannot be run interactively", name)}
		}
	}

	s.sending = true
	in := provider.Inputs{}
	if s.req != nil {
		in = s.req.collect()
	}
	ctx := s.ctx
	return tea.Batch(
		s.spin.Tick,
		func() tea.Msg {
			res, err := iv.Execute(ctx, in)
			return resultMsg{res: res, err: err}
		},
	)
}
