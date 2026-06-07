package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/itchyny/gojq"
	"github.com/jefflinse/clic/ioutil"
)

// testSuite is a clic contract-test file: a reference to the spec under test
// plus a list of cases to run against it.
type testSuite struct {
	Spec   string     `yaml:"spec"   json:"spec"`
	Server string     `yaml:"server" json:"server"`
	Cases  []testCase `yaml:"cases"  json:"cases"`
}

// testCase is a single named request and the expectations its response must
// satisfy. Cmd is the command and arguments exactly as they would be typed
// after the spec on the clic CLI (e.g. "users get 42 --verbose=true").
type testCase struct {
	Name   string      `yaml:"name"   json:"name"`
	Cmd    string      `yaml:"cmd"    json:"cmd"`
	Expect expectation `yaml:"expect" json:"expect"`
}

// expectation describes what a case's response must satisfy. Status is an int or
// a list of ints. Contract, when unset, validates the response against the
// OpenAPI schema only if one is declared; when explicitly true it also requires
// that a schema exists; when false it is skipped. Assert holds optional
// gojq-based body assertions.
type expectation struct {
	Status   any         `yaml:"status"   json:"status"`
	Contract *bool       `yaml:"contract" json:"contract"`
	Assert   []assertion `yaml:"assert"   json:"assert"`
}

// assertion checks a gojq expression's first output against a comparator. With
// no comparator the expression's result must be truthy.
type assertion struct {
	JQ       string  `yaml:"jq"       json:"jq"`
	Equals   *string `yaml:"equals"   json:"equals"`
	Contains *string `yaml:"contains" json:"contains"`
	Exists   *bool   `yaml:"exists"   json:"exists"`
}

// loadTestSuite reads and parses a contract-test suite file.
func loadTestSuite(path string) (*testSuite, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read test suite: %w", err)
	}

	var suite testSuite
	if err := ioutil.Unmarshal(data, &suite); err != nil {
		return nil, fmt.Errorf("failed to parse test suite: %w", err)
	}
	if len(suite.Cases) == 0 {
		return nil, fmt.Errorf("test suite has no cases")
	}
	for i, c := range suite.Cases {
		if c.Cmd == "" {
			return nil, fmt.Errorf("case %d (%q) has no cmd", i+1, c.Name)
		}
	}
	return &suite, nil
}

// wantStatuses normalizes the expect.status field (an int, a list of ints, or
// unset) into a slice of acceptable status codes. An empty result means "any".
func wantStatuses(v any) []int {
	switch t := v.(type) {
	case nil:
		return nil
	case []any:
		var out []int
		for _, e := range t {
			if n, ok := toInt(e); ok {
				out = append(out, n)
			}
		}
		return out
	default:
		if n, ok := toInt(t); ok {
			return []int{n}
		}
		return nil
	}
}

func toInt(v any) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case int64:
		return int(n), true
	case uint64:
		return int(n), true
	case float64:
		return int(n), true
	default:
		return 0, false
	}
}

// evalAssertion runs an assertion's gojq expression against the response body
// and checks the first output against the comparator, returning a descriptive
// error when it does not hold.
func evalAssertion(a assertion, body []byte) error {
	val, ok, err := firstJQ(a.JQ, body)
	if err != nil {
		return err
	}

	switch {
	case a.Exists != nil:
		present := ok && val != nil
		if present != *a.Exists {
			return fmt.Errorf("exists=%v, but value is %spresent", *a.Exists, absentPrefix(present))
		}
	case a.Equals != nil:
		if got := renderJQ(val); got != *a.Equals {
			return fmt.Errorf("got %q, want %q", got, *a.Equals)
		}
	case a.Contains != nil:
		if got := renderJQ(val); !strings.Contains(got, *a.Contains) {
			return fmt.Errorf("%q does not contain %q", got, *a.Contains)
		}
	default:
		if !ok || val == nil || val == false {
			return fmt.Errorf("expression is not truthy")
		}
	}
	return nil
}

func absentPrefix(present bool) string {
	if present {
		return ""
	}
	return "not "
}

// firstJQ parses and runs a gojq program over a JSON body, returning its first
// output value (ok=false when the program yields nothing).
func firstJQ(program string, body []byte) (any, bool, error) {
	query, err := gojq.Parse(program)
	if err != nil {
		return nil, false, err
	}

	var data any
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, false, fmt.Errorf("response is not JSON")
	}

	iter := query.Run(data)
	v, ok := iter.Next()
	if !ok {
		return nil, false, nil
	}
	if e, isErr := v.(error); isErr {
		return nil, false, e
	}
	return v, true, nil
}

// renderJQ renders a gojq output value for comparison: strings verbatim, other
// values as compact JSON.
func renderJQ(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	return string(b)
}

// shellwords splits a command string into arguments, honoring single and double
// quotes so values containing spaces can be passed.
func shellwords(s string) ([]string, error) {
	var args []string
	var cur strings.Builder
	inWord := false
	quote := rune(0)

	for _, r := range s {
		switch {
		case quote != 0:
			if r == quote {
				quote = 0
			} else {
				cur.WriteRune(r)
			}
		case r == '\'' || r == '"':
			quote = r
			inWord = true
		case r == ' ' || r == '\t' || r == '\n':
			if inWord {
				args = append(args, cur.String())
				cur.Reset()
				inWord = false
			}
		default:
			cur.WriteRune(r)
			inWord = true
		}
	}
	if quote != 0 {
		return nil, fmt.Errorf("unterminated quote")
	}
	if inWord {
		args = append(args, cur.String())
	}
	return args, nil
}
