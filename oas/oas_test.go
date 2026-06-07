package oas

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

func mustMarshal(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return b
}

// userDoc returns a small parsed OpenAPI document with a GET /users/{id}
// operation whose 200 response is a user object.
func userDoc(t *testing.T) *openapi3.T {
	t.Helper()
	data := []byte(`
openapi: 3.0.0
info: {title: Users, version: "1.0"}
paths:
  /users/{id}:
    get:
      responses:
        "200":
          description: a user
          content:
            application/json:
              schema:
                type: object
                required: [id, email]
                properties:
                  id: {type: integer}
                  email: {type: string, format: email}
                  role: {type: string, enum: [admin, user]}
        "404":
          description: not found
          content:
            application/json:
              schema: {type: object, properties: {message: {type: string}}}
`)
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromData(data)
	if err != nil {
		t.Fatalf("load doc: %v", err)
	}
	return doc
}

func userOp(t *testing.T) *openapi3.Operation {
	t.Helper()
	return userDoc(t).Paths.Value("/users/{id}").Get
}

func TestExtract(t *testing.T) {
	rs := Extract(userOp(t))
	if _, ok := rs["200"]; !ok {
		t.Fatalf("expected a 200 response schema, got keys %v", keys(rs))
	}
	if _, ok := rs["404"]; !ok {
		t.Fatalf("expected a 404 response schema, got keys %v", keys(rs))
	}
	if rs["200"].Schema == nil || rs["200"].Schema.Value == nil {
		t.Fatal("expected a non-nil 200 schema")
	}
}

func TestExtractNoJSON(t *testing.T) {
	data := []byte(`
openapi: 3.0.0
info: {title: T, version: "1.0"}
paths:
  /x:
    get:
      responses:
        "204": {description: no content}
`)
	doc, err := openapi3.NewLoader().LoadFromData(data)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if rs := Extract(doc.Paths.Value("/x").Get); rs != nil {
		t.Fatalf("expected nil for a JSON-less operation, got %v", rs)
	}
}

func TestValidateBodyConforms(t *testing.T) {
	schema := Extract(userOp(t))["200"].Schema
	body := []byte(`{"id": 1, "email": "ada@example.com", "role": "admin"}`)
	if v := ValidateBody(schema, body); v != nil {
		t.Fatalf("expected conforming body, got violations: %v", v)
	}
}

func TestValidateBodyViolations(t *testing.T) {
	schema := Extract(userOp(t))["200"].Schema

	cases := map[string]string{
		"wrong type":       `{"id": "nope", "email": "ada@example.com"}`,
		"missing required": `{"id": 1}`,
		"bad enum":         `{"id": 1, "email": "ada@example.com", "role": "wizard"}`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			v := ValidateBody(schema, []byte(body))
			if len(v) == 0 {
				t.Fatalf("expected at least one violation for %q", body)
			}
		})
	}
}

// TestValidateBodyFormat covers format validation for the formats kin-openapi
// registers by default (e.g. date-time); email and similar are not validated
// unless a validator is registered.
func TestValidateBodyFormat(t *testing.T) {
	data := []byte(`
openapi: 3.0.0
info: {title: T, version: "1.0"}
paths:
  /x:
    get:
      responses:
        "200":
          description: x
          content:
            application/json:
              schema:
                type: object
                properties:
                  at: {type: string, format: date-time}
`)
	doc, _ := openapi3.NewLoader().LoadFromData(data)
	schema := Extract(doc.Paths.Value("/x").Get)["200"].Schema

	if v := ValidateBody(schema, []byte(`{"at": "2026-01-01T00:00:00Z"}`)); v != nil {
		t.Fatalf("expected a valid date-time to conform, got %v", v)
	}
	if v := ValidateBody(schema, []byte(`{"at": "yesterday"}`)); len(v) == 0 {
		t.Fatal("expected a malformed date-time to violate")
	}
}

func TestValidateBodyInvalidJSON(t *testing.T) {
	schema := Extract(userOp(t))["200"].Schema
	v := ValidateBody(schema, []byte(`{not json`))
	if len(v) != 1 || !strings.Contains(v[0], "not valid JSON") {
		t.Fatalf("expected a JSON-parse violation, got %v", v)
	}
}

func TestValidateBodyNilSchema(t *testing.T) {
	if v := ValidateBody(nil, []byte(`{}`)); v != nil {
		t.Fatalf("expected nil for a nil schema, got %v", v)
	}
}

func TestSynthesizeObject(t *testing.T) {
	schema := Extract(userOp(t))["200"].Schema
	v := Synthesize(schema)
	obj, ok := v.(map[string]any)
	if !ok {
		t.Fatalf("expected an object, got %T", v)
	}
	if _, ok := obj["id"]; !ok {
		t.Fatalf("expected an id property, got %v", obj)
	}
	// the synthesized body should itself conform to the schema
	if viols := ValidateBody(schema, mustMarshal(t, obj)); viols != nil {
		t.Fatalf("synthesized body should conform, got %v", viols)
	}
}

func TestSynthesizeArrayAndScalars(t *testing.T) {
	data := []byte(`
openapi: 3.0.0
info: {title: T, version: "1.0"}
paths:
  /x:
    get:
      responses:
        "200":
          description: list
          content:
            application/json:
              schema:
                type: array
                items: {type: integer}
`)
	doc, _ := openapi3.NewLoader().LoadFromData(data)
	v := Synthesize(Extract(doc.Paths.Value("/x").Get)["200"].Schema)
	arr, ok := v.([]any)
	if !ok || len(arr) != 1 {
		t.Fatalf("expected a one-element array, got %#v", v)
	}
}

func TestSynthesizePrefersExample(t *testing.T) {
	data := []byte(`
openapi: 3.0.0
info: {title: T, version: "1.0"}
paths:
  /x:
    get:
      responses:
        "200":
          description: x
          content:
            application/json:
              schema:
                type: string
                example: "hello"
`)
	doc, _ := openapi3.NewLoader().LoadFromData(data)
	if v := Synthesize(Extract(doc.Paths.Value("/x").Get)["200"].Schema); v != "hello" {
		t.Fatalf("expected the schema example, got %#v", v)
	}
}

func TestPickResponse(t *testing.T) {
	rs := Extract(userOp(t))

	if status, _, ok := PickResponse(rs, 404); !ok || status != "404" {
		t.Fatalf("expected exact 404 match, got %q ok=%v", status, ok)
	}
	if status, _, ok := PickResponse(rs, 0); !ok || status != "200" {
		t.Fatalf("expected lowest-2xx default of 200, got %q ok=%v", status, ok)
	}
	if _, _, ok := PickResponse(nil, 200); ok {
		t.Fatal("expected no match for empty schemas")
	}
}

func keys(rs ResponseSchemas) []string {
	out := make([]string, 0, len(rs))
	for k := range rs {
		out = append(out, k)
	}
	return out
}
