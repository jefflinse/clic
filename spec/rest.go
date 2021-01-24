package spec

import (
	"fmt"
)

// REST is a provider for calling REST endpoints.
type REST struct {
	Method   string `json:"method"`
	Endpoint string `json:"endpoint"`
}

// Name returns the name of the provider.
func (r REST) Name() string {
	return "rest"
}

// Validate returns an error if the provider is invalid.
func (r REST) Validate() (Provider, error) {
	if r.Method == "" {
		return r, fmt.Errorf("invalid rest provider: missing method")
	} else if r.Endpoint == "" {
		return r, fmt.Errorf("invalid rest provider: missing endpoint")
	}

	return REST{
		Method:   r.Method,
		Endpoint: r.Endpoint,
	}, nil
}
