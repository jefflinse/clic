package tui

import (
	"testing"

	"github.com/jefflinse/clic/form"
	"github.com/jefflinse/clic/provider"
	"github.com/stretchr/testify/assert"
)

func TestRequestForm_CollectRoutesSections(t *testing.T) {
	sections := []provider.Section{
		{Key: "path", Title: "Path", Fields: []form.Field{{Name: "id", Type: form.StringField, Required: true}}},
		{Key: "query", Title: "Query", Fields: []form.Field{{Name: "verbose", Type: form.BooleanField}}},
		{Key: "header", Title: "Headers", Fields: []form.Field{{Name: "X-Trace", Type: form.StringField}}},
		{Key: "body", Title: "Body", Fields: []form.Field{{Name: "name", Type: form.StringField}}},
	}
	rf := newRequestForm(sections, newTheme())

	// simulate user input as huh would write it
	rf.binds["path"][0].str = "42"
	rf.binds["query"][0].boolean = true
	rf.binds["header"][0].str = "abc"
	rf.binds["body"][0].str = "Rex"

	in := rf.collect()
	assert.Equal(t, "42", in.Scalars["path"]["id"])
	assert.Equal(t, true, in.Scalars["query"]["verbose"])
	assert.Equal(t, "abc", in.Scalars["header"]["X-Trace"])
	assert.Equal(t, "Rex", in.Body["name"])
	assert.Empty(t, in.RawBody)
}

func TestRequestForm_CollectRawBody(t *testing.T) {
	rf := newRequestForm([]provider.Section{{Key: "body", Title: "Body", Raw: true}}, newTheme())
	*rf.raw["body"] = "  {\"a\":1}  "

	in := rf.collect()
	assert.Equal(t, `{"a":1}`, in.RawBody)
	assert.Nil(t, in.Body)
}

func TestRequestForm_NoSectionsHasNoInputs(t *testing.T) {
	rf := newRequestForm(nil, newTheme())
	assert.False(t, rf.hasInputs())
	assert.Empty(t, rf.collect().Scalars)
}

func TestRequestForm_ComplexArrayHydratesFromJSON(t *testing.T) {
	sections := []provider.Section{{
		Key:   "body",
		Title: "Body",
		Fields: []form.Field{{
			Name: "tags",
			Type: form.ArrayField,
			Item: &form.Field{Type: form.ObjectField, Fields: []form.Field{{Name: "id", Type: form.IntegerField}}},
		}},
	}}
	rf := newRequestForm(sections, newTheme())
	rf.binds["body"][0].str = `[{"id":1},{"id":2}]`

	in := rf.collect()
	tags, ok := in.Body["tags"].([]any)
	assert.True(t, ok)
	assert.Len(t, tags, 2)
}
