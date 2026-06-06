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

	"github.com/jefflinse/clic/form"
	"github.com/jefflinse/clic/ioutil"
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
		req, err := s.parameterizedRequest(cmd.Context(), cmd, args)
		if err != nil {
			return err
		}

		code, body, err := doRequest(req)
		if err != nil {
			return err
		}

		if s.PrintStatus {
			fmt.Println(code)
		}

		fmt.Println(string(body))
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

func (s *Spec) parameterizedRequest(ctx context.Context, cmd *cobra.Command, args []string) (*http.Request, error) {
	// path parameters are positional and substituted into the endpoint
	if err := s.PathParams.ResolveValues(cmd, args); err != nil {
		return nil, err
	}
	endpoint := s.PathParams.InjectPathValues(s.effectiveEndpoint(cmd))

	s.QueryParams.ResolveFromFlags(cmd)
	s.HeaderParams.ResolveFromFlags(cmd)

	bodyReader, err := s.requestBody(cmd)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, s.Method, endpoint, bodyReader)
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
// flag) with the endpoint path. When no base is configured, the endpoint is
// used as-is.
func (s *Spec) effectiveEndpoint(cmd *cobra.Command) string {
	base := s.BaseURL
	if override := provider.OptionsFromContext(cmd.Context()).Server; override != "" {
		base = override
	}

	if base == "" {
		return s.Endpoint
	}

	return strings.TrimRight(base, "/") + "/" + strings.TrimLeft(s.Endpoint, "/")
}

// requestBody returns the request body, either from the --body flag (RawBody
// mode) or assembled from the body-field parameters.
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

// Performs an HTTP request.
func doRequest(req *http.Request) (int, []byte, error) {
	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, err
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, err
	}

	return resp.StatusCode, body, nil
}
