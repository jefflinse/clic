package oauth

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

// isolateHome points the token cache at a throwaway dir so tests never touch the
// real ~/.clic.
func isolateHome(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir) // windows
}

func tokenJSON(w http.ResponseWriter, access, refresh string, expiresIn int) {
	w.Header().Set("Content-Type", "application/json")
	body := map[string]any{"access_token": access, "token_type": "Bearer", "expires_in": expiresIn}
	if refresh != "" {
		body["refresh_token"] = refresh
	}
	_ = json.NewEncoder(w).Encode(body)
}

func TestToken_ClientCredentialsFetchesCachesReuses(t *testing.T) {
	isolateHome(t)
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		require.NoError(t, r.ParseForm())
		assert.Equal(t, "client_credentials", r.Form.Get("grant_type"))
		tokenJSON(w, "access-1", "", 3600)
	}))
	defer srv.Close()

	cfg := Config{Flow: FlowClientCredentials, ClientID: "id", ClientSecret: "secret", TokenURL: srv.URL, Scopes: []string{"read"}}

	tok, err := Token(context.Background(), cfg)
	require.NoError(t, err)
	assert.Equal(t, "access-1", tok)
	assert.Equal(t, 1, calls)

	// second call is served from the cache — no new token request
	tok2, err := Token(context.Background(), cfg)
	require.NoError(t, err)
	assert.Equal(t, "access-1", tok2)
	assert.Equal(t, 1, calls)

	assert.True(t, HasValidToken(cfg))
	require.NoError(t, Logout(cfg))
	assert.False(t, HasValidToken(cfg))
}

func TestToken_RefreshesExpiredToken(t *testing.T) {
	isolateHome(t)
	var grant string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseForm())
		grant = r.Form.Get("grant_type")
		tokenJSON(w, "access-refreshed", "refresh-2", 3600)
	}))
	defer srv.Close()

	cfg := Config{Flow: FlowAuthorizationCode, ClientID: "id", TokenURL: srv.URL}
	// seed an expired token that carries a refresh token
	require.NoError(t, saveToken(cacheKey(cfg), &oauth2.Token{
		AccessToken:  "stale",
		RefreshToken: "refresh-1",
		Expiry:       time.Now().Add(-time.Hour),
	}))

	tok, err := Token(context.Background(), cfg)
	require.NoError(t, err)
	assert.Equal(t, "access-refreshed", tok)
	assert.Equal(t, "refresh_token", grant)
}

func TestToken_AuthCodeWithoutCacheRequiresLogin(t *testing.T) {
	isolateHome(t)
	cfg := Config{Flow: FlowAuthorizationCode, ClientID: "id", TokenURL: "http://unused"}
	_, err := Token(context.Background(), cfg)
	assert.ErrorIs(t, err, ErrLoginRequired)
}

func TestCacheKey_StableAndScopeOrderIndependent(t *testing.T) {
	a := Config{Flow: FlowClientCredentials, ClientID: "id", TokenURL: "https://t", Scopes: []string{"a", "b"}}
	b := Config{Flow: FlowClientCredentials, ClientID: "id", TokenURL: "https://t", Scopes: []string{"b", "a"}}
	assert.Equal(t, cacheKey(a), cacheKey(b))

	c := Config{Flow: FlowClientCredentials, ClientID: "other", TokenURL: "https://t"}
	assert.NotEqual(t, cacheKey(a), cacheKey(c))
}

func TestLoginAuthCode_PKCEAndExchange(t *testing.T) {
	isolateHome(t)

	var gotVerifier, gotGrant string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseForm())
		gotGrant = r.Form.Get("grant_type")
		gotVerifier = r.Form.Get("code_verifier")
		assert.Equal(t, "the-code", r.Form.Get("code"))
		tokenJSON(w, "access-ac", "refresh-ac", 3600)
	}))
	defer srv.Close()

	const redirect = "http://127.0.0.1:47821/callback"
	cfg := Config{
		Flow:        FlowAuthorizationCode,
		ClientID:    "id",
		AuthURL:     "https://auth.invalid/authorize",
		TokenURL:    srv.URL,
		RedirectURL: redirect,
		Scopes:      []string{"openid"},
	}

	// the opener stands in for the browser: it inspects the consent URL (asserting
	// PKCE), then drives the loopback callback with a valid code + state.
	var challengePresent, challengeS256 bool
	opener := func(authURL string) error {
		u, err := url.Parse(authURL)
		if err != nil {
			return err
		}
		q := u.Query()
		challengePresent = q.Get("code_challenge") != ""
		challengeS256 = q.Get("code_challenge_method") == "S256"
		cb := redirect + "?code=the-code&state=" + url.QueryEscape(q.Get("state"))
		resp, err := http.Get(cb)
		if err != nil {
			return err
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		return resp.Body.Close()
	}

	token, err := Login(context.Background(), cfg, opener)
	require.NoError(t, err)
	assert.Equal(t, "access-ac", token)
	assert.True(t, challengePresent, "PKCE code_challenge should be on the auth URL")
	assert.True(t, challengeS256, "PKCE method should be S256")
	assert.NotEmpty(t, gotVerifier, "code_verifier should be sent on exchange")
	assert.Equal(t, "authorization_code", gotGrant)

	// the token (with its refresh token) is now cached
	assert.True(t, HasValidToken(cfg))
}

func TestLoginAuthCode_RejectsStateMismatch(t *testing.T) {
	isolateHome(t)
	const redirect = "http://127.0.0.1:47822/callback"
	cfg := Config{Flow: FlowAuthorizationCode, ClientID: "id", AuthURL: "https://auth.invalid/authorize", TokenURL: "https://unused", RedirectURL: redirect}

	opener := func(authURL string) error {
		resp, err := http.Get(redirect + "?code=x&state=WRONG")
		if err != nil {
			return err
		}
		return resp.Body.Close()
	}

	_, err := Login(context.Background(), cfg, opener)
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "state mismatch"))
	assert.False(t, errors.Is(err, ErrLoginRequired))
}
