// Package oas provides OpenAPI-schema utilities that operate directly on a
// parsed kin-openapi document: extracting an operation's response schemas,
// validating a response body against a schema, and synthesizing an example
// value from a schema.
//
// It is deliberately provider-free (it imports only kin-openapi and the
// standard library) so that both the openapi compiler and the rest provider can
// depend on it without an import cycle, mirroring the provider-free oauth
// package.
package oas

import (
	"sort"
	"strconv"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

// jsonMediaType is the response content type clic validates and synthesizes.
const jsonMediaType = "application/json"

// MediaSchema is the application/json schema (and any example) declared for a
// single response status.
type MediaSchema struct {
	Schema  *openapi3.SchemaRef
	Example any
}

// ResponseSchemas maps an operation's response status keys ("200", "404",
// "default") to their application/json schema. It is empty for operations that
// declare no JSON responses.
type ResponseSchemas map[string]MediaSchema

// Extract pulls the application/json response schemas (and examples) from an
// operation, keyed by status code ("200", "default", …). It returns nil when
// the operation declares no JSON responses.
func Extract(op *openapi3.Operation) ResponseSchemas {
	if op == nil || op.Responses == nil {
		return nil
	}

	out := ResponseSchemas{}
	for status, ref := range op.Responses.Map() {
		if ref == nil || ref.Value == nil {
			continue
		}
		mt := ref.Value.Content.Get(jsonMediaType)
		if mt == nil {
			continue
		}
		out[status] = MediaSchema{Schema: mt.Schema, Example: mt.Example}
	}

	if len(out) == 0 {
		return nil
	}
	return out
}

// PickResponse selects the response schema to use for a given status. It prefers
// an exact match, then the lowest 2xx response, then the "default" response.
// The returned status is the matched key.
func PickResponse(rs ResponseSchemas, prefer int) (status string, ms MediaSchema, ok bool) {
	if len(rs) == 0 {
		return "", MediaSchema{}, false
	}

	if prefer > 0 {
		if ms, found := rs[strconv.Itoa(prefer)]; found {
			return strconv.Itoa(prefer), ms, true
		}
	}

	// lowest 2xx, for stable selection
	var successKeys []string
	for k := range rs {
		if strings.HasPrefix(k, "2") && len(k) == 3 {
			successKeys = append(successKeys, k)
		}
	}
	if len(successKeys) > 0 {
		sort.Strings(successKeys)
		return successKeys[0], rs[successKeys[0]], true
	}

	if ms, found := rs["default"]; found {
		return "default", ms, true
	}

	return "", MediaSchema{}, false
}
