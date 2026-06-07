// Package oauth acquires, caches, and refreshes OAuth2 access tokens for clic.
// It is deliberately free of any clic-internal dependencies (callers map their
// own auth config onto oauth.Config), so it can be used from both the headless
// CLI and the interactive studio without import cycles.
package oauth

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"
)

// Grant flows clic can perform. The string values match provider.Flow* so the
// caller-side mapping is a direct copy.
const (
	FlowClientCredentials = "client_credentials"
	FlowAuthorizationCode = "authorization_code"
)

// DefaultRedirectURL is the loopback redirect used for the authorization-code
// flow when the caller does not override it. The fixed port lets it be
// pre-registered with the OAuth provider.
const DefaultRedirectURL = "http://127.0.0.1:9799/callback"

// Config is everything oauth needs to obtain a token for one credential set.
type Config struct {
	Flow         string
	ClientID     string
	ClientSecret string
	AuthURL      string // authorization endpoint (authorization_code)
	TokenURL     string
	Scopes       []string
	RedirectURL  string // loopback redirect (authorization_code); DefaultRedirectURL if empty
}

// redirectURL returns the configured redirect or the default.
func (c Config) redirectURL() string {
	if c.RedirectURL != "" {
		return c.RedirectURL
	}
	return DefaultRedirectURL
}

// cacheKey derives a stable, filesystem-safe identifier for a credential set so
// its token can be cached and reused across invocations. It intentionally omits
// the client secret.
func cacheKey(c Config) string {
	scopes := append([]string(nil), c.Scopes...)
	sort.Strings(scopes)
	seed := strings.Join([]string{c.Flow, c.TokenURL, c.ClientID, strings.Join(scopes, " ")}, "|")
	sum := sha256.Sum256([]byte(seed))
	return hex.EncodeToString(sum[:])
}
