package oas

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

// ValidateBody checks a response body against a schema, returning a list of
// human-readable violation messages. It returns nil when the body conforms (or
// when there is no schema to validate against). A body that is not valid JSON
// is itself reported as a single violation.
//
// Validation runs in response mode with format validation enabled and collects
// every violation rather than stopping at the first.
func ValidateBody(schema *openapi3.SchemaRef, body []byte) []string {
	if schema == nil || schema.Value == nil {
		return nil
	}

	var v any
	if err := json.Unmarshal(body, &v); err != nil {
		return []string{fmt.Sprintf("response body is not valid JSON: %v", err)}
	}

	err := schema.Value.VisitJSON(v,
		openapi3.VisitAsResponse(),
		openapi3.EnableFormatValidation(),
		openapi3.MultiErrors(),
	)
	if err == nil {
		return nil
	}

	return flatten(err)
}

// flatten renders a (possibly multi-) validation error into readable
// "path: reason" messages.
func flatten(err error) []string {
	if multi, ok := errors.AsType[openapi3.MultiError](err); ok {
		var msgs []string
		for _, e := range multi {
			msgs = append(msgs, flatten(e)...)
		}
		if len(msgs) == 0 {
			return []string{err.Error()}
		}
		return msgs
	}

	if se, ok := errors.AsType[*openapi3.SchemaError](err); ok {
		path := strings.Join(se.JSONPointer(), ".")
		if path == "" {
			return []string{se.Reason}
		}
		return []string{path + ": " + se.Reason}
	}

	return []string{err.Error()}
}
