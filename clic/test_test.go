package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/jefflinse/clic"
	"github.com/jefflinse/clic/provider"
	"github.com/jefflinse/clic/spec"
)

func TestShellwords(t *testing.T) {
	cases := map[string][]string{
		"users get 42":                  {"users", "get", "42"},
		`users create --body='{"a":1}'`: {"users", "create", `--body={"a":1}`},
		`a "b c" d`:                     {"a", "b c", "d"},
		"":                              nil,
	}
	for in, want := range cases {
		got, err := shellwords(in)
		if err != nil {
			t.Fatalf("shellwords(%q): %v", in, err)
		}
		if len(got) != len(want) {
			t.Fatalf("shellwords(%q) = %v, want %v", in, got, want)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("shellwords(%q)[%d] = %q, want %q", in, i, got[i], want[i])
			}
		}
	}

	if _, err := shellwords(`a "unterminated`); err == nil {
		t.Fatal("expected an unterminated-quote error")
	}
}

func TestWantStatuses(t *testing.T) {
	if got := wantStatuses(200); len(got) != 1 || got[0] != 200 {
		t.Fatalf("scalar: got %v", got)
	}
	if got := wantStatuses([]any{200, 201}); len(got) != 2 {
		t.Fatalf("list: got %v", got)
	}
	if got := wantStatuses(nil); got != nil {
		t.Fatalf("nil: got %v", got)
	}
}

func TestEvalAssertion(t *testing.T) {
	body := []byte(`{"email":"ada@example.com","roles":["admin","user"],"active":true}`)
	str := func(s string) *string { return &s }
	yes := true

	mustPass := func(a assertion) {
		if err := evalAssertion(a, body); err != nil {
			t.Fatalf("expected %+v to pass: %v", a, err)
		}
	}
	mustFail := func(a assertion) {
		if err := evalAssertion(a, body); err == nil {
			t.Fatalf("expected %+v to fail", a)
		}
	}

	mustPass(assertion{JQ: ".email", Equals: str("ada@example.com")})
	mustFail(assertion{JQ: ".email", Equals: str("nope@example.com")})
	mustPass(assertion{JQ: ".roles | length", Equals: str("2")})
	gt1 := 1.0
	mustPass(assertion{JQ: ".roles | length", GT: &gt1})
	gt5 := 5.0
	mustFail(assertion{JQ: ".roles | length", GT: &gt5})
	mustPass(assertion{JQ: ".roles[0]", Contains: str("dmin")})
	no := false
	mustPass(assertion{JQ: ".email", Exists: &yes})
	mustFail(assertion{JQ: ".missing", Exists: &yes})
	mustPass(assertion{JQ: ".missing", Exists: &no})
	mustPass(assertion{JQ: ".active"})
	mustFail(assertion{JQ: ".missing"})
}

// usersSpec writes an OpenAPI spec with GET /users/{id} to a temp file.
func usersSpec(t *testing.T) string {
	t.Helper()
	doc := `
openapi: 3.0.0
info: {title: Users, version: "1.0"}
paths:
  /users/{id}:
    get:
      summary: get a user
      parameters:
        - {name: id, in: path, required: true, schema: {type: string}}
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
                  email: {type: string}
        "404":
          description: not found
          content:
            application/json:
              schema: {type: object, properties: {message: {type: string}}}
`
	path := filepath.Join(t.TempDir(), "users.yaml")
	if err := os.WriteFile(path, []byte(doc), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestRunCase_EndToEnd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/users/42":
			_, _ = w.Write([]byte(`{"id":42,"email":"ada@example.com"}`))
		case "/users/7":
			// contract violation: id should be an integer
			_, _ = w.Write([]byte(`{"id":"seven","email":"x@y.z"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"message":"not found"}`))
		}
	}))
	defer srv.Close()

	appSpec, err := clic.LoadSpec(usersSpec(t), spec.FormatUnknown)
	if err != nil {
		t.Fatalf("load spec: %v", err)
	}
	opts := &provider.Options{Server: srv.URL}
	str := func(s string) *string { return &s }
	yes := true

	t.Run("passing case", func(t *testing.T) {
		cr := runCase(context.Background(), appSpec, opts, testCase{
			Name: "get user",
			Cmd:  "users get 42",
			Expect: expectation{
				Status:   200,
				Contract: &yes,
				Assert:   []assertion{{JQ: ".email", Equals: str("ada@example.com")}},
			},
		})
		if !cr.passed() {
			t.Fatalf("expected pass, got failures: %v", cr.Failures)
		}
		if cr.Status != 200 {
			t.Fatalf("status = %d", cr.Status)
		}
	})

	t.Run("wrong status fails", func(t *testing.T) {
		cr := runCase(context.Background(), appSpec, opts, testCase{
			Name:   "missing user",
			Cmd:    "users get 99",
			Expect: expectation{Status: 200},
		})
		if cr.passed() {
			t.Fatal("expected failure for 404 vs expected 200")
		}
	})

	t.Run("contract violation fails", func(t *testing.T) {
		cr := runCase(context.Background(), appSpec, opts, testCase{
			Name:   "bad shape",
			Cmd:    "users get 7",
			Expect: expectation{Status: 200, Contract: &yes},
		})
		if cr.passed() {
			t.Fatal("expected a contract failure for the malformed id")
		}
		found := false
		for _, f := range cr.Failures {
			if len(f) >= 8 && f[:8] == "contract" {
				found = true
			}
		}
		if !found {
			t.Fatalf("expected a contract failure, got %v", cr.Failures)
		}
	})
}
