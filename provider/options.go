package provider

import (
	"context"
	"os"
	"strings"

	"github.com/spf13/pflag"
)

// FlagInteractive is clic's persistent flag that opts into interactive prompts
// (e.g. building a request body via a form instead of passing raw JSON).
const FlagInteractive = "interactive"

// Options carries clic's invocation-wide settings. They are resolved from
// clic's own global flags (with CLIC_* environment fallback for credentials)
// and threaded to providers via the context, deliberately kept out of the
// per-command flag namespace so they can never collide with a spec parameter.
type Options struct {
	Server      string
	Interactive bool
	Token       string
	Username    string
	Password    string
	APIKey      string

	// oauth2 credentials and overrides
	ClientID     string
	ClientSecret string
	Scopes       []string
	OAuthFlow    string // override the grant flow when a spec declares several
	RedirectURL  string // loopback redirect for the authorization-code flow
}

type optionsCtxKey struct{}

// WithOptions returns a context carrying the given options.
func WithOptions(ctx context.Context, o *Options) context.Context {
	return context.WithValue(ctx, optionsCtxKey{}, o)
}

// OptionsFromContext returns the options carried by the context, or zero-valued
// options when none are present.
func OptionsFromContext(ctx context.Context) *Options {
	if o, ok := ctx.Value(optionsCtxKey{}).(*Options); ok && o != nil {
		return o
	}
	return &Options{}
}

// RegisterGlobalFlags registers clic's invocation-wide flags on the given flag
// set. These are clic's own flags, distinct from any spec-derived parameters;
// defaultServer pre-populates the --server override (use "" when unknown).
func RegisterGlobalFlags(flags *pflag.FlagSet, defaultServer string) {
	flags.String(FlagServer, defaultServer, "override the API server base URL")
	flags.BoolP(FlagInteractive, "i", false, "interactively prompt for input")
	flags.String(FlagToken, "", "bearer token (env: CLIC_TOKEN)")
	flags.String(FlagUsername, "", "basic-auth username (env: CLIC_USERNAME)")
	flags.String(FlagPassword, "", "basic-auth password (env: CLIC_PASSWORD)")
	flags.String(FlagAPIKey, "", "API key (env: CLIC_API_KEY)")
	flags.String(FlagClientID, "", "OAuth2 client ID (env: CLIC_CLIENT_ID)")
	flags.String(FlagClientSecret, "", "OAuth2 client secret (env: CLIC_CLIENT_SECRET)")
	flags.String(FlagScopes, "", "OAuth2 scopes, comma-separated (env: CLIC_SCOPES)")
	flags.String(FlagOAuthFlow, "", "OAuth2 grant flow override: client_credentials | authorization_code")
	flags.String(FlagRedirectURL, "", "OAuth2 loopback redirect URL for the authorization-code flow")
}

// ResolveOptions reads clic's global flags from the given flag set into an
// Options value, falling back to CLIC_* environment variables for credentials.
func ResolveOptions(flags *pflag.FlagSet) *Options {
	return &Options{
		Server:       flagString(flags, FlagServer),
		Interactive:  flagBool(flags, FlagInteractive),
		Token:        flagOrEnv(flags, FlagToken),
		Username:     flagOrEnv(flags, FlagUsername),
		Password:     flagOrEnv(flags, FlagPassword),
		APIKey:       flagOrEnv(flags, FlagAPIKey),
		ClientID:     flagOrEnv(flags, FlagClientID),
		ClientSecret: flagOrEnv(flags, FlagClientSecret),
		Scopes:       splitScopes(flagOrEnv(flags, FlagScopes)),
		OAuthFlow:    flagString(flags, FlagOAuthFlow),
		RedirectURL:  flagString(flags, FlagRedirectURL),
	}
}

// splitScopes parses a comma- or space-separated scope list into its elements,
// dropping empties.
func splitScopes(s string) []string {
	fields := strings.FieldsFunc(s, func(r rune) bool {
		return r == ',' || r == ' '
	})
	if len(fields) == 0 {
		return nil
	}
	return fields
}

func flagString(flags *pflag.FlagSet, name string) string {
	if flags != nil && flags.Lookup(name) != nil {
		if v, err := flags.GetString(name); err == nil {
			return v
		}
	}
	return ""
}

func flagBool(flags *pflag.FlagSet, name string) bool {
	if flags != nil && flags.Lookup(name) != nil {
		v, _ := flags.GetBool(name)
		return v
	}
	return false
}

// flagOrEnv returns a flag's value if set and non-empty, otherwise the value of
// the corresponding CLIC_<FLAG> environment variable.
func flagOrEnv(flags *pflag.FlagSet, name string) string {
	if v := flagString(flags, name); v != "" {
		return v
	}
	return os.Getenv("CLIC_" + strings.ToUpper(strings.ReplaceAll(name, "-", "_")))
}
