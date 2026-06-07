// Package mock serves a stateless mock API from a parsed OpenAPI document:
// every operation responds with a synthesized example, and incoming requests
// can be validated against the spec. It is provider-free, depending only on
// kin-openapi, the oas helpers, and the standard library.
package mock

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers/gorillamux"
	"github.com/jefflinse/clic/oas"
)

// Options configures the mock server.
type Options struct {
	// Status forces a specific response status. Zero selects automatically
	// (a Prefer: code=NNN header, then the lowest 2xx, then "default").
	Status int

	// ValidateRequests validates each incoming request against the spec and
	// responds 422 with the violations when it does not conform.
	ValidateRequests bool
}

// Route is a single operation the mock serves.
type Route struct {
	Method string
	Path   string
}

// Handler builds an http.Handler that serves synthesized responses for the
// given OpenAPI document, returning the routes it serves. Routing is by path
// only: the document's servers are normalized so requests to the local mock
// match the spec's raw paths regardless of the declared server host. The
// document is not strictly validated, so specs with minor schema issues still
// serve.
func Handler(doc *openapi3.T, opts Options) (http.Handler, []Route, error) {
	doc.Servers = openapi3.Servers{{URL: "/"}}

	router, err := gorillamux.NewRouter(doc)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to build router: %w", err)
	}

	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		route, pathParams, err := router.FindRoute(r)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]any{
				"error": fmt.Sprintf("no matching operation for %s %s", r.Method, r.URL.Path),
			})
			return
		}

		if opts.ValidateRequests {
			input := &openapi3filter.RequestValidationInput{
				Request:    r,
				PathParams: pathParams,
				Route:      route,
				Options: &openapi3filter.Options{
					MultiError:         true,
					AuthenticationFunc: openapi3filter.NoopAuthenticationFunc,
				},
			}
			if err := openapi3filter.ValidateRequest(r.Context(), input); err != nil {
				writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
					"errors": requestErrors(err),
				})
				return
			}
		}

		status, body := synthesize(route.Operation, opts.Status, r.Header.Get("Prefer"))
		writeJSON(w, status, body)
	})

	return h, routesOf(doc), nil
}

// synthesize chooses a response status and body for an operation, preferring an
// explicit example over a schema-derived one.
func synthesize(op *openapi3.Operation, forced int, prefer string) (int, any) {
	rs := oas.Extract(op)

	preferred := forced
	if p := parsePrefer(prefer); p > 0 {
		preferred = p
	}

	key, ms, ok := oas.PickResponse(rs, preferred)
	if !ok {
		return fallbackStatus(preferred), nil
	}

	status := statusFromKey(key, preferred)
	if ms.Example != nil {
		return status, ms.Example
	}
	return status, oas.Synthesize(ms.Schema)
}

// statusFromKey resolves a numeric status from the matched response key,
// preferring an explicitly requested status and falling back to 200.
func statusFromKey(key string, preferred int) int {
	if preferred > 0 {
		return preferred
	}
	if n, err := strconv.Atoi(key); err == nil {
		return n
	}
	return http.StatusOK
}

func fallbackStatus(preferred int) int {
	if preferred > 0 {
		return preferred
	}
	return http.StatusOK
}

// parsePrefer reads the response status from a Prefer header value of the form
// "code=NNN" (RFC-7240 style, as used by Prism), returning 0 when absent.
func parsePrefer(header string) int {
	for part := range strings.SplitSeq(header, ",") {
		part = strings.TrimSpace(part)
		if code, ok := strings.CutPrefix(part, "code="); ok {
			if n, err := strconv.Atoi(strings.TrimSpace(code)); err == nil {
				return n
			}
		}
	}
	return 0
}

// requestErrors flattens a request-validation error into readable messages.
func requestErrors(err error) []string {
	if multi, ok := errAsMulti(err); ok {
		var msgs []string
		for _, e := range multi {
			msgs = append(msgs, firstLine(e.Error()))
		}
		return msgs
	}
	return []string{firstLine(err.Error())}
}

func errAsMulti(err error) (openapi3.MultiError, bool) {
	multi, ok := err.(openapi3.MultiError) //nolint:errorlint // top-level type is sufficient here
	return multi, ok
}

func firstLine(s string) string {
	first, _, _ := strings.Cut(s, "\n")
	return strings.TrimSpace(first)
}

// routesOf lists the operations declared in the document, sorted for a stable
// startup banner.
func routesOf(doc *openapi3.T) []Route {
	var routes []Route
	for path, item := range doc.Paths.Map() {
		for method := range item.Operations() {
			routes = append(routes, Route{Method: method, Path: path})
		}
	}
	sort.Slice(routes, func(i, j int) bool {
		if routes[i].Path != routes[j].Path {
			return routes[i].Path < routes[j].Path
		}
		return routes[i].Method < routes[j].Method
	})
	return routes
}

// writeJSON writes a JSON response with the given status. A nil body writes no
// payload (e.g. for a 204).
func writeJSON(w http.ResponseWriter, status int, body any) {
	if body == nil {
		w.WriteHeader(status)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
