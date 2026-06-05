package provider

import (
	"context"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// Names of the app-level persistent flags used for server selection and auth.
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

// RegisterServerFlag registers the persistent --server override flag.
func RegisterServerFlag(cmd *cobra.Command, defaultServer string) {
	cmd.PersistentFlags().String(FlagServer, defaultServer, "override the API server base URL")
}

// RegisterFlags registers the persistent auth flags for this scheme.
func (a *AuthScheme) RegisterFlags(cmd *cobra.Command) {
	flags := cmd.PersistentFlags()
	switch strings.ToLower(a.Type) {
	case AuthBearer:
		flags.String(FlagToken, "", "bearer token (env: CLIC_TOKEN)")
	case AuthBasic:
		flags.String(FlagUsername, "", "basic-auth username (env: CLIC_USERNAME)")
		flags.String(FlagPassword, "", "basic-auth password (env: CLIC_PASSWORD)")
	case AuthAPIKey:
		flags.String(FlagAPIKey, "", "API key (env: CLIC_API_KEY)")
	}
}

// Apply adds credentials to the request, reading values from the given flags
// and falling back to CLIC_* environment variables.
func (a *AuthScheme) Apply(req *http.Request, flags *pflag.FlagSet) {
	switch strings.ToLower(a.Type) {
	case AuthBearer:
		if token := flagOrEnv(flags, FlagToken); token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
	case AuthBasic:
		user := flagOrEnv(flags, FlagUsername)
		pass := flagOrEnv(flags, FlagPassword)
		if user != "" || pass != "" {
			req.SetBasicAuth(user, pass)
		}
	case AuthAPIKey:
		key := flagOrEnv(flags, FlagAPIKey)
		if key == "" {
			return
		}
		if strings.EqualFold(a.In, "query") {
			query := req.URL.Query()
			query.Set(a.Name, key)
			req.URL.RawQuery = query.Encode()
		} else {
			req.Header.Set(a.Name, key)
		}
	}
}

// flagOrEnv returns a flag's value if set and non-empty, otherwise the value of
// the corresponding CLIC_<FLAG> environment variable.
func flagOrEnv(flags *pflag.FlagSet, name string) string {
	if flags != nil {
		if flags.Lookup(name) != nil {
			if value, err := flags.GetString(name); err == nil && value != "" {
				return value
			}
		}
	}

	return os.Getenv("CLIC_" + strings.ToUpper(strings.ReplaceAll(name, "-", "_")))
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
