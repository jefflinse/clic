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
	Commands    []Command
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

	// overlays and transient UI
	helpOpen bool
	pal      *paletteState
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
	s.relayout()
	if s.req.form != nil {
		return s.req.form.Init()
	}
	return nil
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
			s.focus = focusRequest
		} else {
			s.focus = focusCommands
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

	switch msg.String() {
	case "ctrl+c":
		return s, tea.Quit
	case "ctrl+s":
		return s, s.send()
	case "ctrl+p":
		s.openPalette()
		return s, nil
	}

	// '?' and '/' are reserved outside the request form (where they're text)
	if s.focus != focusRequest {
		switch msg.String() {
		case "?":
			s.helpOpen = true
			return s, nil
		case "/":
			s.openPalette()
			return s, nil
		}
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
	case "right", "l", "enter", "tab":
		s.focus = focusCommands
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
	case "left", "h", "shift+tab":
		s.focus = focusGroups
	case "right", "l", "enter", "tab":
		if s.req != nil && s.req.hasInputs() {
			s.focus = focusRequest
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
	switch msg.String() {
	case "esc":
		s.focus = focusCommands
		return nil
	}
	if s.req == nil || s.req.form == nil {
		return nil
	}
	form, cmd := s.req.form.Update(msg)
	s.req.form = asHuhForm(form, s.req.form)
	return cmd
}

func (s *studio) handleResponseKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc", "left", "h":
		if s.req != nil && s.req.hasInputs() {
			s.focus = focusRequest
		} else {
			s.focus = focusCommands
		}
		return nil
	case "tab":
		s.resp.cycleTab()
		return nil
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

// forward routes a non-key message to whichever sub-model is active.
func (s *studio) forward(msg tea.Msg) tea.Cmd {
	switch s.focus {
	case focusRequest:
		if s.req != nil && s.req.form != nil {
			form, cmd := s.req.form.Update(msg)
			s.req.form = asHuhForm(form, s.req.form)
			return cmd
		}
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
