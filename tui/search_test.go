package tui

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/jefflinse/clic/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func jsonResult(body string) *provider.Result {
	return &provider.Result{
		Kind:        provider.ResultHTTP,
		Status:      http.StatusOK,
		ContentType: "application/json",
		Body:        []byte(body),
	}
}

func TestSearch_FindsHitsAndJumps(t *testing.T) {
	r := newResponsePane(newTheme())
	r.setSize(80, 20)
	r.setResult(jsonResult(`{"name":"rex","friends":["rex","fido","rex"]}`))

	r.setSearch("rex")
	// pretty-printed, each "rex" lands on its own line: the name and two array
	// elements => three distinct match lines.
	require.Len(t, r.search.lines, 3)
	assert.Equal(t, 0, r.search.cur)

	r.nextMatch(1)
	assert.Equal(t, 1, r.search.cur)
	r.nextMatch(-1)
	assert.Equal(t, 0, r.search.cur)
	r.nextMatch(-1) // wraps to the last hit
	assert.Equal(t, 2, r.search.cur)

	// the search view still shows the matched text
	assert.Contains(t, r.body(), "rex")
}

func TestSearch_NoMatchesAndClear(t *testing.T) {
	r := newResponsePane(newTheme())
	r.setSize(80, 20)
	r.setResult(jsonResult(`{"name":"rex"}`))

	r.setSearch("zzz")
	assert.Empty(t, r.search.lines)

	r.clearSearch()
	assert.Equal(t, "", r.search.query)
}

func TestFilter_AppliesJQInPlace(t *testing.T) {
	r := newResponsePane(newTheme())
	r.setSize(80, 20)
	r.setResult(jsonResult(`{"name":"rex","age":3}`))

	r.applyFilter(".name")
	require.NoError(t, r.filter.err)
	assert.Equal(t, `"rex"`, string(r.sourceBytes()))
	assert.Contains(t, r.body(), "rex")
}

func TestFilter_BadProgramKeepsRawBody(t *testing.T) {
	r := newResponsePane(newTheme())
	r.setSize(80, 20)
	body := `{"name":"rex"}`
	r.setResult(jsonResult(body))

	r.applyFilter("this is not jq")
	require.Error(t, r.filter.err)
	// with an invalid filter the body falls back to the raw response
	assert.Equal(t, body, string(r.sourceBytes()))
	// and the summary surfaces the error
	assert.Contains(t, r.summary(), "jq")
}

func TestFilter_NonJSONResponse(t *testing.T) {
	r := newResponsePane(newTheme())
	r.setSize(80, 20)
	r.setResult(&provider.Result{Kind: provider.ResultText, Body: []byte("plain text, not json")})

	r.applyFilter(".foo")
	require.Error(t, r.filter.err)
	assert.Contains(t, r.filter.err.Error(), "not JSON")
}

func TestRunJQ_MultipleOutputsOnePerLine(t *testing.T) {
	out, err := runJQ(".items[]", []byte(`{"items":[1,2,3]}`))
	require.NoError(t, err)
	assert.Equal(t, "1\n2\n3", string(out))
}

func TestClearActive_PeelsSearchThenFilter(t *testing.T) {
	r := newResponsePane(newTheme())
	r.setSize(80, 20)
	r.setResult(jsonResult(`{"name":"rex"}`))

	r.applyFilter(".name")
	r.setSearch("rex")

	assert.True(t, r.clearActive()) // clears search first
	assert.Equal(t, "", r.search.query)
	assert.Equal(t, ".name", r.filter.program)

	assert.True(t, r.clearActive()) // then the filter
	assert.Equal(t, "", r.filter.program)

	assert.False(t, r.clearActive()) // nothing left
}

func TestHighlightMatches_CaseInsensitive(t *testing.T) {
	out, ok := highlightMatches("Name: Rex", "rex", newTheme().match)
	require.True(t, ok)
	assert.Contains(t, out, "Rex") // original case preserved despite case-insensitive match

	_, ok = highlightMatches("nothing here", "xyz", newTheme().match)
	assert.False(t, ok)
}

func TestStudio_FilterAndSearchViaKeys(t *testing.T) {
	s := newStudio(context.Background(), testApp())
	sized(s, 120, 40)
	s.Update(resultMsg{res: jsonResult(`{"name":"rex","id":7}`)})
	require.Equal(t, focusResponse, s.focus)

	// 'f' opens the jq input; type a program and apply it
	s.Update(key("f"))
	require.NotNil(t, s.editing)
	for _, r := range ".name" {
		s.Update(key(string(r)))
	}
	s.Update(key("enter"))
	assert.Nil(t, s.editing)
	assert.Equal(t, ".name", s.resp.filter.program)
	assert.Equal(t, `"rex"`, string(s.resp.sourceBytes()))

	// esc clears the filter rather than leaving the response pane
	s.Update(key("esc"))
	assert.Equal(t, "", s.resp.filter.program)
	assert.Equal(t, focusResponse, s.focus)
}

func TestStudio_SearchViaSlashKey(t *testing.T) {
	s := newStudio(context.Background(), testApp())
	sized(s, 120, 40)
	s.Update(resultMsg{res: jsonResult(`{"name":"rex","id":7}`)})

	s.Update(key("/")) // in the response pane, '/' is search (not the palette)
	require.NotNil(t, s.editing)
	require.Nil(t, s.pal)
	for _, r := range "name" {
		s.Update(key(string(r)))
	}
	s.Update(key("enter"))
	assert.Equal(t, "name", s.resp.search.query)
	assert.NotEmpty(t, s.resp.search.lines)

	view := s.View()
	assert.Contains(t, strings.ToLower(view), "name")
}
