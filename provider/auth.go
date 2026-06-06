package provider

import (
	"context"
	"net/http"
	"strings"
)

// Names of clic's global flags used for server selection and auth.
const (
	FlagServer   = "server"
	FlagToken    = "token"
	FlagUsername = "username"
	FlagPassword = "password"
	FlagAPIKey   = "api-key"
)

// Auth scheme types.
const (
	AuthBearer = "bearer"
	AuthBasic  = "basic"
	AuthAPIKey = "apikey"
)

// AuthScheme describes how requests are authenticated, surfaced as CLI flags
// with CLIC_* environment-variable fallback.
type AuthScheme struct {
	Type string `json:"type"           yaml:"type"`           // bearer | basic | apikey
	In   string `json:"in,omitempty"   yaml:"in,omitempty"`   // header | query (apikey)
	Name string `json:"name,omitempty" yaml:"name,omitempty"` // header/query name (apikey)
}

// Apply adds credentials to the request using the values resolved into the
// given options.
func (a *AuthScheme) Apply(req *http.Request, o *Options) {
	switch strings.ToLower(a.Type) {
	case AuthBearer:
		if o.Token != "" {
			req.Header.Set("Authorization", "Bearer "+o.Token)
		}
	case AuthBasic:
		if o.Username != "" || o.Password != "" {
			req.SetBasicAuth(o.Username, o.Password)
		}
	case AuthAPIKey:
		if o.APIKey == "" {
			return
		}
		if strings.EqualFold(a.In, "query") {
			query := req.URL.Query()
			query.Set(a.Name, o.APIKey)
			req.URL.RawQuery = query.Encode()
		} else {
			req.Header.Set(a.Name, o.APIKey)
		}
	}
}

type authCtxKey struct{}

// WithAuth returns a context carrying the given auth scheme.
func WithAuth(ctx context.Context, a *AuthScheme) context.Context {
	return context.WithValue(ctx, authCtxKey{}, a)
}

// AuthFromContext returns the auth scheme carried by the context, if any.
func AuthFromContext(ctx context.Context) *AuthScheme {
	auth, _ := ctx.Value(authCtxKey{}).(*AuthScheme)
	return auth
}
