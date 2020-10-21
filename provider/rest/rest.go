package rest

import (
	"bytes"
	"encoding/json"
	"fmt"
	goioutil "io/ioutil"
	"net/http"
	"net/url"

	"github.com/jefflinse/handyman/ioutil"
	"github.com/jefflinse/handyman/provider"
	"github.com/urfave/cli/v2"
)

// Spec describes the provider.
type Spec struct {
	Endpoint    string                `json:"endpoint"               yaml:"endpoint"`
	Method      string                `json:"method"                 yaml:"method"`
	Headers     map[string]string     `json:"headers,omitempty"      yaml:"headers,omitempty"`
	QueryParams provider.ParameterSet `json:"query_params,omitempty" yaml:"query_params,omitempty"`
	BodyParams  provider.ParameterSet `json:"body_params,omitempty"  yaml:"body_params,omitempty"`
}

// New creates a new provider.
func New(v interface{}) (provider.Provider, error) {
	s := Spec{}
	return &s, ioutil.Intermarshal(v, &s)
}

// CLIActionFn creates a CLI action fuction.
func (s Spec) CLIActionFn() cli.ActionFunc {
	return func(ctx *cli.Context) error {
		req, err := s.parameterizedRequest(ctx)
		if err != nil {
			return err
		}

		code, body, err := doRequest(req)
		if err != nil {
			return err
		}

		fmt.Println(code)
		fmt.Println(string(body))
		return nil
	}
}

// CLIFlags creates a set of CLI flags.
func (s Spec) CLIFlags() []cli.Flag {
	flags := []cli.Flag{}
	allParams := s.allParams()
	for _, param := range allParams {
		var flag cli.Flag
		switch param.Type {
		case "bool":
			flag = &cli.BoolFlag{
				Name:     param.CLIFlagName(),
				Usage:    param.Description,
				Required: param.Required,
			}
		case "int":
			flag = &cli.IntFlag{
				Name:     param.CLIFlagName(),
				Usage:    param.Description,
				Required: param.Required,
			}
		case "number":
			flag = &cli.Float64Flag{
				Name:     param.CLIFlagName(),
				Usage:    param.Description,
				Required: param.Required,
			}
		case "string":
			flag = &cli.StringFlag{
				Name:     param.CLIFlagName(),
				Usage:    param.Description,
				Required: param.Required,
			}
		}

		flags = append(flags, flag)
	}

	return flags
}

// Type returns the type.
func (s Spec) Type() string {
	return "rest"
}

// Validate validates the provider.
func (s Spec) Validate() error {
	if s.Method == "" {
		return fmt.Errorf("invalid %s command spec: missing method", s.Type())
	} else if s.Endpoint == "" {
		return fmt.Errorf("invalid %s command spec: missing endpoint", s.Type())
	}

	for _, param := range s.allParams() {
		if err := param.Validate(); err != nil {
			return err
		}
	}

	return nil
}

func (s *Spec) allParams() provider.ParameterSet {
	return append(append(provider.ParameterSet{}, s.QueryParams...), s.BodyParams...)
}

func (s *Spec) parameterizedRequest(ctx *cli.Context) (*http.Request, error) {
	url, err := url.Parse(s.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("invalid endpoint '%s': %w", s.Endpoint, err)
	}

	req := &http.Request{
		Method: s.Method,
		URL:    url,
	}

	req.Header = http.Header{"Content-Type": {"application/json"}}
	for name, value := range s.Headers {
		req.Header.Set(name, value)
	}

	if len(s.QueryParams) > 0 {
		query := req.URL.Query()
		s.QueryParams.ResolveValues(ctx)
		for _, param := range s.QueryParams {
			query.Add(param.Name, fmt.Sprintf("%v", param.Value()))
		}

		req.URL.RawQuery = query.Encode()
	}

	body := map[string]interface{}{}
	if len(s.BodyParams) > 0 {
		s.BodyParams.ResolveValues(ctx)
		for _, param := range s.BodyParams {
			body[param.Name] = param.Value()
		}
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req.Body = goioutil.NopCloser(bytes.NewBuffer(bodyBytes))

	return req, nil
}

// Performs an HTTPS request.
func doRequest(req *http.Request) (int, []byte, error) {
	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, err
	}

	defer resp.Body.Close()
	body, err := goioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, err
	}

	return resp.StatusCode, body, nil
}
