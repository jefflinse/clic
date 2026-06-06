package tui

import (
	"strings"

	"github.com/jefflinse/clic/provider"
)

// variable is a value captured from a response and reused in later requests by
// referencing {{name}} in any field. It is the heart of request chaining.
type variable struct {
	name  string
	value string
}

// setVar adds or replaces a captured variable, keeping insertion order stable.
func (s *studio) setVar(name, value string) {
	for i := range s.vars {
		if s.vars[i].name == name {
			s.vars[i].value = value
			return
		}
	}
	s.vars = append(s.vars, variable{name: name, value: value})
}

// applyVars substitutes {{name}} references in every string the user entered
// with the captured variable values, across scalars, the body, and the raw body.
func (s *studio) applyVars(in provider.Inputs) provider.Inputs {
	if len(s.vars) == 0 {
		return in
	}
	for sec := range in.Scalars {
		for name, v := range in.Scalars[sec] {
			in.Scalars[sec][name] = s.substituteValue(v)
		}
	}
	for name, v := range in.Body {
		in.Body[name] = s.substituteValue(v)
	}
	in.RawBody = s.substituteString(in.RawBody)
	return in
}

// substituteValue substitutes variable references within a value, descending
// into nested body maps and arrays.
func (s *studio) substituteValue(v any) any {
	switch t := v.(type) {
	case string:
		return s.substituteString(t)
	case map[string]any:
		for k, val := range t {
			t[k] = s.substituteValue(val)
		}
		return t
	case []any:
		for i, val := range t {
			t[i] = s.substituteValue(val)
		}
		return t
	default:
		return v
	}
}

// substituteString replaces every {{name}} occurrence with its variable value.
func (s *studio) substituteString(str string) string {
	if !strings.Contains(str, "{{") {
		return str
	}
	for _, v := range s.vars {
		str = strings.ReplaceAll(str, "{{"+v.name+"}}", v.value)
	}
	return str
}
