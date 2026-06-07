package mock

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

const doc = `
openapi: 3.0.0
info: {title: Users, version: "1.0"}
paths:
  /users/{id}:
    get:
      parameters:
        - {name: id, in: path, required: true, schema: {type: string}}
      responses:
        "200":
          description: a user
          content:
            application/json:
              schema:
                type: object
                required: [id, name]
                properties:
                  id: {type: integer}
                  name: {type: string}
        "404":
          description: not found
          content:
            application/json:
              schema: {type: object, properties: {message: {type: string}}}
  /users:
    post:
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required: [name]
              properties:
                name: {type: string}
                age: {type: integer}
      responses:
        "201":
          description: created
          content:
            application/json:
              schema:
                type: object
                properties: {id: {type: integer}}
`

func newServer(t *testing.T, opts Options) *httptest.Server {
	t.Helper()
	d, err := openapi3.NewLoader().LoadFromData([]byte(doc))
	if err != nil {
		t.Fatalf("load doc: %v", err)
	}
	h, routes, err := Handler(d, opts)
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	if len(routes) != 2 {
		t.Fatalf("expected 2 routes, got %d: %+v", len(routes), routes)
	}
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return srv
}

func TestMock_SynthesizesMatchedResponse(t *testing.T) {
	srv := newServer(t, Options{ValidateRequests: true})

	resp, err := http.Get(srv.URL + "/users/42")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var body map[string]any
	dec(t, resp, &body)
	if _, ok := body["id"]; !ok {
		t.Fatalf("expected an id in the synthesized body, got %v", body)
	}
	if _, ok := body["name"]; !ok {
		t.Fatalf("expected a name in the synthesized body, got %v", body)
	}
}

func TestMock_ValidatesRequestBody(t *testing.T) {
	srv := newServer(t, Options{ValidateRequests: true})

	// missing the required "name"
	resp, err := http.Post(srv.URL+"/users", "application/json", strings.NewReader(`{"age": 30}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", resp.StatusCode)
	}
	var body struct {
		Errors []string `json:"errors"`
	}
	dec(t, resp, &body)
	if len(body.Errors) == 0 {
		t.Fatal("expected validation errors in the 422 body")
	}
}

func TestMock_ValidRequestBodySucceeds(t *testing.T) {
	srv := newServer(t, Options{ValidateRequests: true})

	resp, err := http.Post(srv.URL+"/users", "application/json", strings.NewReader(`{"name": "Ada"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want 201", resp.StatusCode)
	}
}

func TestMock_UnknownPathIs404(t *testing.T) {
	srv := newServer(t, Options{ValidateRequests: true})

	resp, err := http.Get(srv.URL + "/nope")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
}

func TestMock_PreferHeaderSelectsStatus(t *testing.T) {
	srv := newServer(t, Options{ValidateRequests: true})

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/users/42", nil)
	req.Header.Set("Prefer", "code=404")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 (from Prefer header)", resp.StatusCode)
	}
}

func dec(t *testing.T, resp *http.Response, v any) {
	t.Helper()
	b, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(b, v); err != nil {
		t.Fatalf("decode body %q: %v", string(b), err)
	}
}
