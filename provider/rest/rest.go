package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/jefflinse/clic/ioutil"
	"github.com/jefflinse/clic/provider"
	"github.com/spf13/cobra"
)

// Spec describes the provider.
type Spec struct {
	Endpoint    string                `json:"endpoint"               yaml:"endpoint"`
	Method      string                `json:"method"                 yaml:"method"`
	Headers     map[string]string     `json:"headers,omitempty"      yaml:"headers,omitempty"`
	QueryParams provider.ParameterSet `json:"query_params,omitempty" yaml:"query_params,omitempty"`
	BodyParams  provider.ParameterSet `json:"body_params,omitempty"  yaml:"body_params,omitempty"`
	PrintStatus bool                  `json:"print_status,omitempty" yaml:"print_status"`
}

// New creates a new provider.
func New(v any) (provider.Provider, error) {
	s := Spec{}
	return &s, ioutil.Intermarshal(v, &s)
}

// Configure wires up the command's positional arguments, flags, and run behavior.
func (s *Spec) Configure(cmd *cobra.Command) {
	if usage := s.allParams().ArgsUsage(); usage != "" {
		cmd.Use += " " + usage
	}

	s.allParams().RegisterFlags(cmd.Flags())

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
	} else if err := s.allParams().Validate(); err != nil {
		return err
	}

	return nil
}

func (s *Spec) allParams() provider.ParameterSet {
	return append(append(provider.ParameterSet{}, s.QueryParams...), s.BodyParams...)
}

func (s *Spec) parameterizedRequest(ctx context.Context, cmd *cobra.Command, args []string) (*http.Request, error) {
	if err := s.allParams().ResolveValues(cmd, args); err != nil {
		return nil, err
	}

	body := map[string]any{}
	for _, param := range s.BodyParams {
		body[param.Name] = param.Value()
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, s.Method, s.Endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("invalid endpoint '%s': %w", s.Endpoint, err)
	}

	req.Header.Set("Content-Type", "application/json")
	for name, value := range s.Headers {
		req.Header.Set(name, value)
	}

	if len(s.QueryParams) > 0 {
		query := req.URL.Query()
		for _, param := range s.QueryParams {
			query.Add(param.Name, fmt.Sprintf("%v", param.Value()))
		}

		req.URL.RawQuery = query.Encode()
	}

	return req, nil
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
