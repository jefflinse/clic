package tui

import (
	"context"
	"net/http"
	"testing"

	"github.com/jefflinse/clic/form"
	"github.com/jefflinse/clic/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFlattenJSONLeaves_DocumentOrderAndPaths(t *testing.T) {
	raw := []byte(`{"id":42,"name":"Rex","tags":[{"k":"a"},{"k":"b"}],"ok":true,"meta":null}`)
	leaves := flattenJSONLeaves(raw)

	got := map[string]string{}
	var order []string
	for _, l := range leaves {
		got[l.path] = l.value
		order = append(order, l.path)
	}

	assert.Equal(t, "42", got[".id"])
	assert.Equal(t, "Rex", got[".name"])
	assert.Equal(t, "a", got[".tags[0].k"])
	assert.Equal(t, "b", got[".tags[1].k"])
	assert.Equal(t, "true", got[".ok"])
	assert.Equal(t, "null", got[".meta"])
	// numbers keep their literal form (no float formatting)
	assert.Equal(t, []string{".id", ".name", ".tags[0].k", ".tags[1].k", ".ok", ".meta"}, order)
}

func TestFlattenJSONLeaves_NonJSON(t *testing.T) {
	assert.Empty(t, flattenJSONLeaves([]byte("not json")))
	assert.Empty(t, flattenJSONLeaves(nil))
}

func TestSuggestVarName(t *testing.T) {
	assert.Equal(t, "id", suggestVarName(".id"))
	assert.Equal(t, "petId", suggestVarName(".data[0].petId"))
	assert.Equal(t, "tags", suggestVarName(".tags[3]"))
	assert.Equal(t, "var", suggestVarName(""))
}

func TestApplyVars_SubstitutesAcrossInputs(t *testing.T) {
	s := &studio{}
	s.setVar("petId", "42")

	in := provider.Inputs{
		Scalars: map[string]map[string]any{
			"path":  {"id": "{{petId}}"},
			"query": {"n": 5},
		},
		Body:    map[string]any{"ref": "pet-{{petId}}", "nested": map[string]any{"x": "{{petId}}"}},
		RawBody: `{"id":"{{petId}}"}`,
	}
	out := s.applyVars(in)

	assert.Equal(t, "42", out.Scalars["path"]["id"])
	assert.Equal(t, 5, out.Scalars["query"]["n"]) // non-strings untouched
	assert.Equal(t, "pet-42", out.Body["ref"])
	assert.Equal(t, "42", out.Body["nested"].(map[string]any)["x"])
	assert.Equal(t, `{"id":"42"}`, out.RawBody)
}

func TestStudio_CaptureAndVarsOverlaysRender(t *testing.T) {
	prov := &fakeProvider{result: &provider.Result{Kind: provider.ResultHTTP, Body: []byte(`{"id":7,"name":"Rex"}`)}}
	app := StudioApp{Commands: []Command{{Name: "pets", Provider: prov}}}
	s := newStudio(context.Background(), app)
	sized(s, 120, 40)
	s.Update(resultMsg{res: prov.result})

	s.openCapture()
	capView := s.View()
	assert.Contains(t, capView, "capture")
	assert.Contains(t, capView, ".name")

	s.Update(key("enter")) // into the naming phase
	assert.Contains(t, s.View(), "capture variable")

	s.Update(key("enter")) // save with suggested name
	s.varsOpen = true
	assert.Contains(t, s.View(), "{{id}}")
}

// end-to-end: capture an id from a response, reference it in another command's
// field, and confirm the value reaches the provider on send.
func TestStudio_CaptureThenChainIntoNextRequest(t *testing.T) {
	getById := &fakeProvider{
		summary:  "GET /pets/{id}",
		sections: []provider.Section{{Key: "path", Title: "Path", Fields: []form.Field{{Name: "id", Type: form.StringField, Required: true}}}},
		result:   &provider.Result{Kind: provider.ResultHTTP, Status: http.StatusOK, Body: []byte(`{"id":42}`)},
	}
	app := StudioApp{Commands: []Command{{Name: "pets", Subcommands: []Command{{Name: "getById", Provider: getById}}}}}

	s := newStudio(context.Background(), app)
	s.preselect([]string{"pets", "getById"})
	sized(s, 120, 40)

	// a response arrives, then capture .id as {{petId}}
	s.Update(resultMsg{res: getById.result})
	s.openCapture()
	require.NotNil(t, s.cap)
	s.Update(key("enter")) // pick the highlighted leaf (.id)
	assert.True(t, s.cap.naming)
	s.Update(key("enter")) // accept suggested name "id"
	require.Len(t, s.vars, 1)
	assert.Equal(t, "id", s.vars[0].name)
	assert.Equal(t, "42", s.vars[0].value)

	// reference the variable in the form and send; the provider should see "42"
	s.focus = focusRequest
	for _, r := range "{{id}}" {
		s.Update(key(string(r)))
	}
	cmd := s.send()
	require.NotNil(t, cmd)
	cmd() // execute the send thunk
	assert.Equal(t, "42", getById.gotInput.Scalar("path", "id"))
}
