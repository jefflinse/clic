package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/jefflinse/clic/form"
	"github.com/jefflinse/clic/ioutil"
	"github.com/jefflinse/clic/oas"
	"github.com/jefflinse/clic/provider"
	"github.com/jefflinse/clic/tui"
	"github.com/spf13/cobra"
)

// Spec describes the provider.
type Spec struct {
	BaseURL      string                `json:"base_url,omitempty"      yaml:"base_url,omitempty"`
	Endpoint     string                `json:"endpoint"                yaml:"endpoint"`
	Method       string                `json:"method"                  yaml:"method"`
	Headers      map[string]string     `json:"headers,omitempty"       yaml:"headers,omitempty"`
	PathParams   provider.ParameterSet `json:"path_params,omitempty"   yaml:"path_params,omitempty"`
	QueryParams  provider.ParameterSet `json:"query_params,omitempty"  yaml:"query_params,omitempty"`
	HeaderParams provider.ParameterSet `json:"header_params,omitempty" yaml:"header_params,omitempty"`
	BodyParams   provider.ParameterSet `json:"body_params,omitempty"   yaml:"body_params,omitempty"`
	RawBody      bool                  `json:"raw_body,omitempty"      yaml:"raw_body,omitempty"`
	Body         []form.Field          `json:"body,omitempty"          yaml:"body,omitempty"`
	PrintStatus  bool                  `json:"print_status,omitempty"  yaml:"print_status,omitempty"`

	// Responses holds the OpenAPI application/json response schemas for this
	// operation, keyed by status ("200", "default", …), used for contract
	// validation. It is populated only when compiled from an OpenAPI document
	// and is excluded from serialization, so it is unavailable in converted or
	// built native specs.
	Responses oas.ResponseSchemas `json:"-" yaml:"-"`
}

const bodyFlagName = "body"

// New creates a new provider.
func New(v any) (provider.Provider, error) {
	s := Spec{}
	return &s, ioutil.Intermarshal(v, &s)
}

// Configure wires up the command's positional arguments, flags, and run behavior.
//
// Path parameters are positional (and substituted into the endpoint); query,
// header, and body-field parameters are flags. When RawBody is set, the request
// body comes from a --body flag (inline JSON or @file) instead of body fields.
func (s *Spec) Configure(cmd *cobra.Command) {
	if usage := s.PathParams.ArgsUsage(); usage != "" {
		cmd.Use += " " + usage
	}

	s.QueryParams.RegisterAsFlags(cmd)
	s.HeaderParams.RegisterAsFlags(cmd)
	if s.RawBody {
		cmd.Flags().String(bodyFlagName, "", "request body as inline JSON or @file")
	} else {
		s.BodyParams.RegisterAsFlags(cmd)
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		// path parameters are positional and substituted into the endpoint
		if err := s.PathParams.ResolveValues(cmd, args); err != nil {
			return err
		}
		s.QueryParams.ResolveFromFlags(cmd)
		s.HeaderParams.ResolveFromFlags(cmd)

		body, err := s.requestBody(cmd)
		if err != nil {
			return err
		}

		req, err := s.buildRequest(cmd.Context(), body)
		if err != nil {
			return err
		}

		code, _, respBody, err := doRequest(req)
		if err != nil {
			return err
		}

		if s.PrintStatus {
			fmt.Println(code)
		}

		fmt.Println(string(respBody))
		return nil
	}
}

// Type returns the type.
func (s *Spec) Type() string {
	return "rest"
}

// Validate validates the provider.
func (s *Spec) Validate() error {
	if s.Method == "" {
		return fmt.Errorf("invalid %s command spec: missing method", s.Type())
	} else if s.Endpoint == "" {
		return fmt.Errorf("invalid %s command spec: missing endpoint", s.Type())
	}

	for _, set := range []provider.ParameterSet{s.PathParams, s.QueryParams, s.HeaderParams, s.BodyParams} {
		if err := set.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// Summary describes the request in one line, e.g. "GET /pets/{id}".
func (s *Spec) Summary() string {
	return strings.ToUpper(s.Method) + " " + s.Endpoint
}

// Sections describes the request's inputs for interactive entry: path, query,
// and header parameters, plus a body section (discrete fields, flat body
// params, or a single raw-text block depending on how the command is defined).
func (s *Spec) Sections() []provider.Section {
	var secs []provider.Section
	if len(s.PathParams) > 0 {
		secs = append(secs, provider.Section{Key: "path", Title: "Path", Fields: s.PathParams.Fields()})
	}
	if len(s.QueryParams) > 0 {
		secs = append(secs, provider.Section{Key: "query", Title: "Query", Fields: s.QueryParams.Fields()})
	}
	if len(s.HeaderParams) > 0 {
		secs = append(secs, provider.Section{Key: "header", Title: "Headers", Fields: s.HeaderParams.Fields()})
	}

	switch {
	case s.RawBody:
		secs = append(secs, provider.Section{Key: "body", Title: "Body", Raw: true})
	case len(s.Body) > 0:
		secs = append(secs, provider.Section{Key: "body", Title: "Body", Fields: s.Body})
	case len(s.BodyParams) > 0:
		secs = append(secs, provider.Section{Key: "body", Title: "Body", Fields: s.BodyParams.Fields()})
	}

	return secs
}

// Execute assigns the interactively-collected values, performs the request, and
// returns a structured result for display.
func (s *Spec) Execute(ctx context.Context, in provider.Inputs) (*provider.Result, error) {
	s.PathParams.Assign(in.Scalars["path"])
	s.QueryParams.Assign(in.Scalars["query"])
	s.HeaderParams.Assign(in.Scalars["header"])

	body, err := s.interactiveBodyBytes(in)
	if err != nil {
		return nil, err
	}

	return s.do(ctx, bytes.NewReader(body))
}

// do builds and performs the request and returns a structured result, including
// any contract-validation outcome. It is shared by the interactive Execute path
// and the headless run path.
func (s *Spec) do(ctx context.Context, body io.Reader) (*provider.Result, error) {
	req, err := s.buildRequest(ctx, body)
	if err != nil {
		return nil, err
	}

	start := time.Now()
	code, headers, respBody, err := doRequest(req)
	if err != nil {
		return nil, err
	}

	return &provider.Result{
		Kind:        provider.ResultHTTP,
		RequestLine: s.Method + " " + req.URL.String(),
		Status:      code,
		Latency:     time.Since(start),
		Headers:     headers,
		ContentType: headers.Get("Content-Type"),
		Body:        respBody,
		Contract:    s.validateContract(code, respBody),
	}, nil
}

// validateContract checks the response body against the OpenAPI response schema
// for its status. It returns nil when the operation declares no response
// schemas at all (e.g. a native spec). When a schema exists for the status it
// reports conformance; when none matches the status (or the body is empty),
// the result is marked unchecked.
func (s *Spec) validateContract(status int, body []byte) *provider.ContractResult {
	if len(s.Responses) == 0 {
		return nil
	}
	key, ms, ok := oas.PickResponse(s.Responses, status)
	if !ok || len(body) == 0 {
		return &provider.ContractResult{Checked: false}
	}
	return &provider.ContractResult{
		Checked:    true,
		Status:     key,
		Violations: oas.ValidateBody(ms.Schema, body),
	}
}

// Preview describes exactly what the request will send, without performing it.
// It assigns the collected inputs the same way Execute does, then reports the
// resolved method, URL, headers, and body, plus the headless CLI arguments that
// reproduce the request.
func (s *Spec) Preview(ctx context.Context, in provider.Inputs) (*provider.RequestPreview, error) {
	s.PathParams.Assign(in.Scalars["path"])
	s.QueryParams.Assign(in.Scalars["query"])
	s.HeaderParams.Assign(in.Scalars["header"])

	body, err := s.interactiveBodyBytes(in)
	if err != nil {
		return nil, err
	}

	req, err := s.buildRequest(ctx, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	return &provider.RequestPreview{
		Kind:    provider.ResultHTTP,
		Method:  strings.ToUpper(s.Method),
		URL:     req.URL.String(),
		Headers: req.Header,
		Body:    displayBody(body),
		CLIArgs: s.cliArgs(in),
	}, nil
}

// interactiveBodyBytes builds the raw request body from collected studio inputs.
// A nil result means the request carries no body.
func (s *Spec) interactiveBodyBytes(in provider.Inputs) ([]byte, error) {
	switch {
	case s.RawBody:
		if in.RawBody == "" {
			return nil, nil
		}
		if path, ok := strings.CutPrefix(in.RawBody, "@"); ok {
			content, err := os.ReadFile(path)
			if err != nil {
				return nil, fmt.Errorf("failed to read body file: %w", err)
			}
			return content, nil
		}
		return []byte(in.RawBody), nil

	case len(s.Body) > 0:
		return json.Marshal(in.Body)

	default:
		s.BodyParams.Assign(in.Body)
		body := map[string]any{}
		for _, param := range s.BodyParams {
			body[param.Name] = param.Value()
		}
		return json.Marshal(body)
	}
}

// cliArgs returns the positional arguments and flags that reproduce this request
// from the headless CLI: path parameters are positional (in declared order),
// query/header/body parameters are flags, and a raw or structured body becomes a
// --body flag.
func (s *Spec) cliArgs(in provider.Inputs) []string {
	var args []string
	for _, p := range s.PathParams {
		args = append(args, fmt.Sprintf("%v", p.Value()))
	}
	for _, set := range []provider.ParameterSet{s.QueryParams, s.HeaderParams} {
		for _, p := range set {
			if v := fmt.Sprintf("%v", p.Value()); v != "" {
				args = append(args, "--"+p.CLIFlagName()+"="+v)
			}
		}
	}

	switch {
	case s.RawBody:
		if in.RawBody != "" {
			args = append(args, "--"+bodyFlagName+"="+in.RawBody)
		}
	case len(s.Body) > 0:
		if b := displayBody(mustJSON(in.Body)); b != nil {
			args = append(args, "--"+bodyFlagName+"="+string(b))
		}
	default:
		for _, p := range s.BodyParams {
			if v := fmt.Sprintf("%v", p.Value()); v != "" {
				args = append(args, "--"+p.CLIFlagName()+"="+v)
			}
		}
	}
	return args
}

// displayBody returns b unless it is empty or an empty JSON object, in which
// case it returns nil so callers can omit a meaningless body.
func displayBody(b []byte) []byte {
	switch strings.TrimSpace(string(b)) {
	case "", "{}", "null":
		return nil
	}
	return b
}

// mustJSON marshals v, returning nil on error (used only for display).
func mustJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	return b
}

// buildRequest assembles the HTTP request from parameters that already hold
// their values (assigned from either cobra flags or interactive inputs) and the
// given body reader. It substitutes path parameters, applies headers and query
// parameters, and attaches auth from the context.
func (s *Spec) buildRequest(ctx context.Context, body io.Reader) (*http.Request, error) {
	endpoint := s.PathParams.InjectPathValues(s.effectiveEndpoint(ctx))

	req, err := http.NewRequestWithContext(ctx, s.Method, endpoint, body)
	if err != nil {
		return nil, fmt.Errorf("invalid endpoint '%s': %w", endpoint, err)
	}

	req.Header.Set("Content-Type", "application/json")
	for name, value := range s.Headers {
		req.Header.Set(name, value)
	}
	for _, param := range s.HeaderParams {
		if value := fmt.Sprintf("%v", param.Value()); value != "" {
			req.Header.Set(param.Name, value)
		}
	}

	if len(s.QueryParams) > 0 {
		query := req.URL.Query()
		for _, param := range s.QueryParams {
			if value := fmt.Sprintf("%v", param.Value()); value != "" {
				query.Add(param.Name, value)
			}
		}
		req.URL.RawQuery = query.Encode()
	}

	if auth := provider.AuthFromContext(ctx); auth != nil {
		auth.Apply(req, provider.OptionsFromContext(ctx))
	}

	return req, nil
}

// effectiveEndpoint joins the base URL (overridable via the global --server
// flag, threaded through the context options) with the endpoint path. When no
// base is configured, the endpoint is used as-is.
func (s *Spec) effectiveEndpoint(ctx context.Context) string {
	base := s.BaseURL
	if override := provider.OptionsFromContext(ctx).Server; override != "" {
		base = override
	}

	if base == "" {
		return s.Endpoint
	}

	return strings.TrimRight(base, "/") + "/" + strings.TrimLeft(s.Endpoint, "/")
}

// requestBody returns the request body for the headless CLI path, either from
// the --body flag (RawBody mode) or assembled from the body-field parameters.
func (s *Spec) requestBody(cmd *cobra.Command) (io.Reader, error) {
	if s.RawBody {
		raw, _ := cmd.Flags().GetString(bodyFlagName)
		if raw != "" {
			if path, ok := strings.CutPrefix(raw, "@"); ok {
				content, err := os.ReadFile(path)
				if err != nil {
					return nil, fmt.Errorf("failed to read body file: %w", err)
				}
				return bytes.NewReader(content), nil
			}
			return strings.NewReader(raw), nil
		}

		// no raw body supplied: offer an interactive form when the user opted
		// in and we have a schema to drive it
		if provider.OptionsFromContext(cmd.Context()).Interactive && len(s.Body) > 0 {
			values, err := tui.PromptBody(s.Body)
			if err != nil {
				return nil, err
			}
			bodyBytes, err := json.Marshal(values)
			if err != nil {
				return nil, err
			}
			return bytes.NewReader(bodyBytes), nil
		}

		return http.NoBody, nil
	}

	s.BodyParams.ResolveFromFlags(cmd)
	body := map[string]any{}
	for _, param := range s.BodyParams {
		body[param.Name] = param.Value()
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(bodyBytes), nil
}

// doRequest performs an HTTP request, returning the status code, response
// headers, and body.
func doRequest(req *http.Request) (int, http.Header, []byte, error) {
	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, nil, err
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, resp.Header, nil, err
	}

	return resp.StatusCode, resp.Header, body, nil
}
