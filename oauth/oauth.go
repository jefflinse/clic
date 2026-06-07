package oauth

import (
	"context"
	"errors"
	"fmt"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

// ErrLoginRequired is returned by Token when a valid token is not cached and the
// configured flow needs interactive login (authorization_code). Callers should
// run Login (e.g. via `clic login`) to obtain one.
var ErrLoginRequired = errors.New("oauth2: interactive login required")

// Opener launches the user's browser to the given URL. It is injectable so the
// authorization-code flow can be driven in tests without a real browser.
type Opener func(url string) error

// Token returns a valid access token without any interaction: it reuses a cached
// token, silently refreshes an expired one when a refresh token is present, and
// otherwise fetches a fresh token for the non-interactive client-credentials
// flow. For authorization_code with no usable cached token it returns
// ErrLoginRequired.
func Token(ctx context.Context, cfg Config) (string, error) {
	key := cacheKey(cfg)

	if tok, ok := loadToken(key); ok {
		if tok.Valid() {
			return tok.AccessToken, nil
		}
		if tok.RefreshToken != "" {
			if refreshed, err := refreshToken(ctx, cfg, tok); err == nil {
				_ = saveToken(key, refreshed)
				return refreshed.AccessToken, nil
			}
			// refresh failed (e.g. revoked); fall through to re-acquire
		}
	}

	if cfg.Flow == FlowClientCredentials {
		tok, err := fetchClientCredentials(ctx, cfg)
		if err != nil {
			return "", err
		}
		_ = saveToken(key, tok)
		return tok.AccessToken, nil
	}

	return "", ErrLoginRequired
}

// Login obtains a token interactively when needed and caches it: a direct fetch
// for client-credentials, or the browser-based authorization-code + PKCE flow.
// opener may be nil to use the default system browser.
func Login(ctx context.Context, cfg Config, opener Opener) (string, error) {
	key := cacheKey(cfg)

	var (
		tok *oauth2.Token
		err error
	)
	switch cfg.Flow {
	case FlowClientCredentials:
		tok, err = fetchClientCredentials(ctx, cfg)
	case FlowAuthorizationCode:
		tok, err = loginAuthCode(ctx, cfg, opener)
	default:
		return "", fmt.Errorf("oauth2: unsupported flow %q", cfg.Flow)
	}
	if err != nil {
		return "", err
	}
	_ = saveToken(key, tok)
	return tok.AccessToken, nil
}

// Logout removes any cached token for the given credential set.
func Logout(cfg Config) error {
	return removeToken(cacheKey(cfg))
}

// HasValidToken reports whether a non-expired token is already cached, without
// fetching or refreshing. The studio uses it to show auth status at a glance.
func HasValidToken(cfg Config) bool {
	tok, ok := loadToken(cacheKey(cfg))
	return ok && tok.Valid()
}

// CachedToken returns the cached access token if one is present and unexpired,
// without any network call. The studio uses it to seed auth state on launch.
func CachedToken(cfg Config) (string, bool) {
	if tok, ok := loadToken(cacheKey(cfg)); ok && tok.Valid() {
		return tok.AccessToken, true
	}
	return "", false
}

func fetchClientCredentials(ctx context.Context, cfg Config) (*oauth2.Token, error) {
	cc := &clientcredentials.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		TokenURL:     cfg.TokenURL,
		Scopes:       cfg.Scopes,
	}
	tok, err := cc.Token(ctx)
	if err != nil {
		return nil, fmt.Errorf("oauth2: client-credentials token request failed: %w", err)
	}
	return tok, nil
}

// refreshToken mints a fresh access token from a refresh token using oauth2's
// auto-refreshing TokenSource.
func refreshToken(ctx context.Context, cfg Config, tok *oauth2.Token) (*oauth2.Token, error) {
	oc := &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		Endpoint:     oauth2.Endpoint{TokenURL: cfg.TokenURL},
		Scopes:       cfg.Scopes,
	}
	return oc.TokenSource(ctx, tok).Token()
}
