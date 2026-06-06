package provider

import (
	"context"
	"net/http"
	"time"

	"github.com/jefflinse/clic/form"
)

// ResultKind distinguishes how a command's result should be displayed.
type ResultKind string

const (
	// ResultHTTP is the result of an HTTP request: it carries a status code,
	// response headers, and a body, and is rendered with a status badge.
	ResultHTTP ResultKind = "http"

	// ResultText is a plain textual result (command output, a lambda payload):
	// it carries only a body and an optional exit code.
	ResultText ResultKind = "text"
)

// A Result is the structured outcome of executing a command interactively. It
// is the data the studio renders in its response pane; the headless CLI paths
// continue to print to stdout and do not produce a Result.
type Result struct {
	// Kind selects how the result is displayed.
	Kind ResultKind

	// RequestLine summarizes what was sent (e.g. "GET https://api/pets/42").
	RequestLine string

	// Status is the HTTP status code (ResultHTTP) or process exit code
	// (ResultText). Zero means "not applicable".
	Status int

	// Latency is the wall-clock time the execution took.
	Latency time.Duration

	// Headers are the HTTP response headers (ResultHTTP only).
	Headers http.Header

	// ContentType is the response's content type, when known.
	ContentType string

	// Body is the raw response body or captured output.
	Body []byte
}

// A Section is a named group of input fields within a command's interactive
// form. The Key is a stable identifier ("path", "query", "header", "body")
// that a provider uses to route collected values back to the right place.
type Section struct {
	// Key routes collected values back to the provider ("path", "query",
	// "header", "body").
	Key string

	// Title is the human-facing heading shown above the section.
	Title string

	// Fields are the inputs in the section, in display order.
	Fields []form.Field

	// Raw marks a body section whose value is entered as a single block of
	// freeform text (e.g. raw JSON) rather than as discrete fields.
	Raw bool
}

// Inputs carries the values collected from an interactive form back to a
// provider for execution.
type Inputs struct {
	// Scalars maps a section key ("path", "query", "header") to that section's
	// field name/value pairs.
	Scalars map[string]map[string]any

	// Body is the assembled request body, when the command builds one from
	// discrete fields.
	Body map[string]any

	// RawBody, when non-empty, is used verbatim as the request body instead of
	// Body (raw-body commands and the studio's raw editing mode).
	RawBody string
}

// Scalar returns the value of a named field within a section, or nil.
func (in Inputs) Scalar(section, name string) any {
	if in.Scalars == nil {
		return nil
	}
	return in.Scalars[section][name]
}

// Describer is implemented by providers that can summarize what a command does
// in one line (e.g. "GET /pets/{id}"), shown as the studio's request header.
type Describer interface {
	Summary() string
}

// Interactive is implemented by providers that can be driven from the studio.
// They describe their inputs as ordered sections and execute with the values
// collected from those sections, returning a structured Result for display.
type Interactive interface {
	// Sections returns the input sections to render, in display order. A nil or
	// empty result means the command takes no interactive input and can be run
	// immediately.
	Sections() []Section

	// Execute runs the command with the collected inputs and returns a result.
	Execute(ctx context.Context, in Inputs) (*Result, error)
}
