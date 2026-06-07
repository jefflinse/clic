package rest

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/jefflinse/clic/form"
	"github.com/jefflinse/clic/oas"
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

func TestPreview_HTTPRequestAndCLIArgs(t *testing.T) {
	s := &Spec{
		Method:       "post",
		BaseURL:      "https://api.example.com",
		Endpoint:     "/pets/{id}",
		PathParams:   provider.ParameterSet{{Name: "id", Type: provider.StringParamType, Required: true}},
		QueryParams:  provider.ParameterSet{{Name: "verbose", Type: provider.BoolParamType}},
		HeaderParams: provider.ParameterSet{{Name: "X-Trace", Type: provider.StringParamType}},
		BodyParams:   provider.ParameterSet{{Name: "name", Type: provider.StringParamType}},
	}

	pv, err := s.Preview(context.Background(), provider.Inputs{
		Scalars: map[string]map[string]any{
			"path":   {"id": "42"},
			"query":  {"verbose": true},
			"header": {"X-Trace": "abc"},
		},
		Body: map[string]any{"name": "Rex"},
	})
	require.NoError(t, err)

	assert.Equal(t, provider.ResultHTTP, pv.Kind)
	assert.Equal(t, "POST", pv.Method)
	assert.Equal(t, "https://api.example.com/pets/42?verbose=true", pv.URL)
	assert.Equal(t, "abc", pv.Headers.Get("X-Trace"))
	assert.Equal(t, "application/json", pv.Headers.Get("Content-Type"))
	assert.JSONEq(t, `{"name":"Rex"}`, string(pv.Body))

	// path is positional; query/header/body become flags
	assert.Equal(t, []string{"42", "--verbose=true", "--x-trace=abc", "--name=Rex"}, pv.CLIArgs)
}

func TestPreview_RawBodyOmitsEmptyAndCarriesBodyFlag(t *testing.T) {
	s := &Spec{Method: "GET", BaseURL: "https://api.example.com", Endpoint: "/x"}

	empty, err := s.Preview(context.Background(), provider.Inputs{})
	require.NoError(t, err)
	assert.Nil(t, empty.Body)
	assert.Empty(t, empty.CLIArgs)

	s.RawBody = true
	withBody, err := s.Preview(context.Background(), provider.Inputs{RawBody: `{"raw":1}`})
	require.NoError(t, err)
	assert.Equal(t, []byte(`{"raw":1}`), withBody.Body)
	assert.Equal(t, []string{`--body={"raw":1}`}, withBody.CLIArgs)
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

// contractSchema returns the response schemas for a GET /x whose 200 body is an
// object with a required integer id.
func contractSchema(t *testing.T) oas.ResponseSchemas {
	t.Helper()
	data := []byte(`
openapi: 3.0.0
info: {title: T, version: "1.0"}
paths:
  /x:
    get:
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                type: object
                required: [id]
                properties: {id: {type: integer}}
`)
	doc, err := openapi3.NewLoader().LoadFromData(data)
	require.NoError(t, err)
	return oas.Extract(doc.Paths.Value("/x").Get)
}

func contractServer(t *testing.T, body string) *Spec {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return &Spec{Method: "GET", BaseURL: srv.URL, Endpoint: "/x", Responses: contractSchema(t)}
}

func TestExecute_AttachesContract_Conforms(t *testing.T) {
	s := contractServer(t, `{"id": 1}`)
	res, err := s.Execute(context.Background(), provider.Inputs{})
	require.NoError(t, err)
	require.NotNil(t, res.Contract)
	assert.True(t, res.Contract.Checked)
	assert.Equal(t, "200", res.Contract.Status)
	assert.Empty(t, res.Contract.Violations)
}

func TestExecute_AttachesContract_Violates(t *testing.T) {
	s := contractServer(t, `{"id": "not-an-integer"}`)
	res, err := s.Execute(context.Background(), provider.Inputs{})
	require.NoError(t, err)
	require.NotNil(t, res.Contract)
	assert.True(t, res.Contract.Checked)
	assert.NotEmpty(t, res.Contract.Violations)
}

func TestExecute_NoResponseSchemas_NoContract(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"id":1}`))
	}))
	defer srv.Close()

	s := &Spec{Method: "GET", BaseURL: srv.URL, Endpoint: "/x"}
	res, err := s.Execute(context.Background(), provider.Inputs{})
	require.NoError(t, err)
	assert.Nil(t, res.Contract, "a spec without response schemas should not produce a contract result")
}
