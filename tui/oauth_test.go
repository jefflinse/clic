package tui

import (
	"context"
	"testing"

	"github.com/jefflinse/clic/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func oauthCtx(scheme *provider.AuthScheme, opts *provider.Options) context.Context {
	ctx := context.Background()
	if opts != nil {
		ctx = provider.WithOptions(ctx, opts)
	}
	if scheme != nil {
		ctx = provider.WithAuth(ctx, scheme)
	}
	return ctx
}

func ccScheme() *provider.AuthScheme {
	return &provider.AuthScheme{
		Type:     provider.AuthOAuth2,
		Flow:     provider.FlowClientCredentials,
		TokenURL: "https://auth.example.com/token",
		Scopes:   []string{"read"},
	}
}

func TestStudio_DetectsOAuthAndShowsSignIn(t *testing.T) {
	s := newStudio(oauthCtx(ccScheme(), &provider.Options{ClientID: "id"}), testApp())
	sized(s, 120, 40)

	require.True(t, s.authOAuth)
	assert.Empty(t, s.authToken)
	assert.Contains(t, s.View(), "sign in")
}

func TestStudio_NonOAuthAppHasNoAuthState(t *testing.T) {
	s := newStudio(context.Background(), testApp())
	sized(s, 120, 40)

	assert.False(t, s.authOAuth)
	// startLogin is a no-op (with a flash) for non-oauth apps
	cmd := s.startLogin()
	assert.Nil(t, cmd)
	assert.False(t, s.loggingIn)
	assert.Contains(t, s.flash, "does not use OAuth2")
}

func TestStudio_LoginResultInjectsToken(t *testing.T) {
	s := newStudio(oauthCtx(ccScheme(), &provider.Options{ClientID: "id"}), testApp())
	sized(s, 120, 40)

	s.Update(loginResultMsg{token: "tok-xyz"})
	assert.Equal(t, "tok-xyz", s.authToken)

	// the execution context now carries the token as a bearer credential
	assert.Equal(t, "tok-xyz", provider.OptionsFromContext(s.execCtx()).Token)
	assert.Contains(t, s.View(), "authed")
}

func TestStudio_LoginFailureFlashes(t *testing.T) {
	s := newStudio(oauthCtx(ccScheme(), &provider.Options{}), testApp())
	sized(s, 120, 40)

	s.loggingIn = true
	s.Update(loginResultMsg{err: assertErr("boom")})
	assert.False(t, s.loggingIn)
	assert.Contains(t, s.flash, "sign-in failed")
}

func TestStudio_StartLoginMarksInFlight(t *testing.T) {
	s := newStudio(oauthCtx(ccScheme(), &provider.Options{ClientID: "id"}), testApp())
	sized(s, 120, 40)

	cmd := s.startLogin() // returns the (unexecuted) login command
	require.NotNil(t, cmd)
	assert.True(t, s.loggingIn)
	assert.Contains(t, s.View(), "signing in")
}

func TestBuildOAuthConfig_FlagOverrides(t *testing.T) {
	scheme := &provider.AuthScheme{
		Type:     provider.AuthOAuth2,
		Flow:     provider.FlowClientCredentials,
		AuthURL:  "https://auth/authorize",
		TokenURL: "https://auth/token",
		Scopes:   []string{"default"},
	}
	opts := &provider.Options{
		ClientID:     "cid",
		ClientSecret: "csec",
		OAuthFlow:    provider.FlowAuthorizationCode, // override
		Scopes:       []string{"x", "y"},             // override
		RedirectURL:  "http://127.0.0.1:5555/cb",
	}

	cfg := buildOAuthConfig(scheme, opts)
	assert.Equal(t, provider.FlowAuthorizationCode, cfg.Flow)
	assert.Equal(t, []string{"x", "y"}, cfg.Scopes)
	assert.Equal(t, "cid", cfg.ClientID)
	assert.Equal(t, "csec", cfg.ClientSecret)
	assert.Equal(t, "https://auth/token", cfg.TokenURL)
	assert.Equal(t, "http://127.0.0.1:5555/cb", cfg.RedirectURL)
}

type assertErr string

func (e assertErr) Error() string { return string(e) }
