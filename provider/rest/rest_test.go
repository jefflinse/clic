package rest

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jefflinse/clic/form"
	"github.com/jefflinse/clic/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSections_OrdersAndLabelsInputs(t *testing.T) {
	s := &Spec{
		Method:       "POST",
		Endpoint:     "/pets/{id}",
		PathParams:   provider.ParameterSet{{Name: "id", Type: provider.StringParamType, Required: true}},
		QueryParams:  provider.ParameterSet{{Name: "verbose", Type: provider.BoolParamType}},
		HeaderParams: provider.ParameterSet{{Name: "X-Trace", Type: provider.StringParamType}},
		Body:         []form.Field{{Name: "name", Type: form.StringField, Required: true}},
	}

	secs := s.Sections()
	require.Len(t, secs, 4)
	assert.Equal(t, []string{"path", "query", "header", "body"}, []string{secs[0].Key, secs[1].Key, secs[2].Key, secs[3].Key})
	assert.Equal(t, "id", secs[0].Fields[0].Name)
	assert.Equal(t, form.BooleanField, secs[1].Fields[0].Type)
	assert.False(t, secs[3].Raw)
}

func TestSections_RawBody(t *testing.T) {
	s := &Spec{Method: "POST", Endpoint: "/x", RawBody: true}
	secs := s.Sections()
	require.Len(t, secs, 1)
	assert.True(t, secs[0].Raw)
}

func TestExecute_WiresPathQueryHeaderAndBody(t *testing.T) {
	var (
		gotPath   string
		gotQuery  string
		gotHeader string
		gotBody   map[string]any
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.Query().Get("verbose")
		gotHeader = r.Header.Get("X-Trace")
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &gotBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	s := &Spec{
		Method:       "POST",
		BaseURL:      srv.URL,
		Endpoint:     "/pets/{id}",
		PathParams:   provider.ParameterSet{{Name: "id", Type: provider.StringParamType, Required: true}},
		QueryParams:  provider.ParameterSet{{Name: "verbose", Type: provider.BoolParamType}},
		HeaderParams: provider.ParameterSet{{Name: "X-Trace", Type: provider.StringParamType}},
		Body:         []form.Field{{Name: "name", Type: form.StringField}},
	}

	result, err := s.Execute(context.Background(), provider.Inputs{
		Scalars: map[string]map[string]any{
			"path":   {"id": "42"},
			"query":  {"verbose": true},
			"header": {"X-Trace": "abc"},
		},
		Body: map[string]any{"name": "Rex"},
	})
	require.NoError(t, err)

	assert.Equal(t, "/pets/42", gotPath)
	assert.Equal(t, "true", gotQuery)
	assert.Equal(t, "abc", gotHeader)
	assert.Equal(t, "Rex", gotBody["name"])

	assert.Equal(t, provider.ResultHTTP, result.Kind)
	assert.Equal(t, http.StatusCreated, result.Status)
	assert.Equal(t, "application/json", result.ContentType)
	assert.JSONEq(t, `{"ok":true}`, string(result.Body))
	assert.Positive(t, result.Latency)
	assert.Contains(t, result.RequestLine, "POST ")
}

func TestExecute_RawBody(t *testing.T) {
	var got []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	s := &Spec{Method: "POST", BaseURL: srv.URL, Endpoint: "/x", RawBody: true}
	_, err := s.Execute(context.Background(), provider.Inputs{RawBody: `{"raw":1}`})
	require.NoError(t, err)
	assert.JSONEq(t, `{"raw":1}`, string(got))
}
