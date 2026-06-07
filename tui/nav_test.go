package tui

import (
	"context"
	"net/http"
	"testing"

	"github.com/jefflinse/clic/form"
	"github.com/jefflinse/clic/provider"
	"github.com/stretchr/testify/assert"
)

// tabbing forward cycles the pane ring consistently and wraps; shift+tab
// reverses it. Exercised on a no-input command so tab crosses panes directly
// (with a form, tab navigates fields first).
func TestNav_TabCyclesPaneRing(t *testing.T) {
	s := newStudio(context.Background(), testApp())
	s.preselect([]string{"ping"}) // top-level leaf, no inputs -> lands on commands
	sized(s, 120, 40)
	assert.Equal(t, focusCommands, s.focus)

	s.Update(key("tab"))
	assert.Equal(t, focusRequest, s.focus)
	s.Update(key("tab"))
	assert.Equal(t, focusResponse, s.focus)
	s.Update(key("tab")) // wrap
	assert.Equal(t, focusGroups, s.focus)
	s.Update(key("tab"))
	assert.Equal(t, focusCommands, s.focus)

	// shift+tab reverses
	s.Update(key("shift+tab"))
	assert.Equal(t, focusGroups, s.focus)
	s.Update(key("shift+tab")) // wrap backward
	assert.Equal(t, focusResponse, s.focus)
}

// esc steps back toward the command tree and stops at groups (no wrap).
func TestNav_EscStepsBack(t *testing.T) {
	s := newStudio(context.Background(), testApp())
	s.preselect([]string{"ping"})
	sized(s, 120, 40)

	s.focus = focusResponse
	s.Update(key("esc"))
	assert.Equal(t, focusRequest, s.focus)
	s.Update(key("esc"))
	assert.Equal(t, focusCommands, s.focus)
	s.Update(key("esc"))
	assert.Equal(t, focusGroups, s.focus)
	s.Update(key("esc")) // stays at groups
	assert.Equal(t, focusGroups, s.focus)
}

// the response pane is always reachable via the ring — the bug this sweep fixes
// was that once you left it, nothing brought you back.
func TestNav_ResponseReachableAfterLeaving(t *testing.T) {
	s := newStudio(context.Background(), testApp())
	s.preselect([]string{"pets", "getById"}) // form command, focus starts on request
	sized(s, 120, 40)

	// a response arrives, then we step away from it
	s.Update(resultMsg{res: &provider.Result{Kind: provider.ResultHTTP, Status: http.StatusOK, Body: []byte(`{}`)}})
	assert.Equal(t, focusResponse, s.focus)
	s.Update(key("esc"))
	assert.Equal(t, focusRequest, s.focus)

	// shift+tab from groups wraps straight back to the response
	s.focus = focusGroups
	s.Update(key("shift+tab"))
	assert.Equal(t, focusResponse, s.focus)
}

// shift+tab on the form's first field leaves the form backward into commands.
func TestNav_ShiftTabLeavesFormFromFirstField(t *testing.T) {
	s := newStudio(context.Background(), testApp())
	s.preselect([]string{"pets", "getById"})
	sized(s, 120, 40)
	assert.Equal(t, focusRequest, s.focus)

	s.Update(key("shift+tab"))
	assert.Equal(t, focusCommands, s.focus)
}

// tab off the form's last field crosses forward into the response pane.
func TestNav_TabOffLastFieldEntersResponse(t *testing.T) {
	s := newStudio(context.Background(), testApp())
	s.preselect([]string{"pets", "getById"}) // single-field form, focus on request
	sized(s, 120, 40)
	assert.Equal(t, focusRequest, s.focus)

	s.Update(key("tab")) // id is the only (last) field -> edge -> response
	assert.Equal(t, focusResponse, s.focus)
}

// tab between fields stays within the form (huh returns a nav command, so we
// don't treat it as an edge).
func TestNav_TabBetweenFieldsStaysInForm(t *testing.T) {
	prov := &fakeProvider{sections: []provider.Section{{Key: "q", Title: "Q", Fields: []form.Field{
		{Name: "a", Type: form.StringField},
		{Name: "b", Type: form.StringField},
	}}}}
	s := newStudio(context.Background(), StudioApp{Commands: []Command{{Name: "two", Provider: prov}}})
	s.preselect([]string{"two"})
	sized(s, 120, 40)
	assert.Equal(t, focusRequest, s.focus)

	s.Update(key("tab")) // moves from field a to field b, stays in the form
	assert.Equal(t, focusRequest, s.focus)
}

// in the response pane, left/right switch the view tab rather than moving panes.
func TestNav_ResponseArrowsSwitchView(t *testing.T) {
	s := newStudio(context.Background(), testApp())
	sized(s, 120, 40)
	s.Update(resultMsg{res: &provider.Result{
		Kind:    provider.ResultHTTP,
		Status:  http.StatusOK,
		Headers: http.Header{"Content-Type": []string{"application/json"}},
		Body:    []byte(`{"a":1}`),
	}})
	assert.Equal(t, focusResponse, s.focus)
	assert.Equal(t, tabPretty, s.resp.tab)

	s.Update(key("right"))
	assert.Equal(t, tabHeaders, s.resp.tab)
	s.Update(key("left"))
	assert.Equal(t, tabPretty, s.resp.tab)
	s.Update(key("left")) // wraps backward to the last tab
	assert.Equal(t, tabRequest, s.resp.tab)

	// focus stays on the response — arrows do not move panes here
	assert.Equal(t, focusResponse, s.focus)
}
