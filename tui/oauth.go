package tui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jefflinse/clic/oauth"
	"github.com/jefflinse/clic/provider"
)

// loginResultMsg reports the outcome of an OAuth2 sign-in started from the studio.
type loginResultMsg struct {
	token string
	err   error
}

// initAuth inspects the context's auth scheme; when it is oauth2 the studio
// tracks an access token of its own (seeded from the on-disk cache) and exposes
// a sign-in action. Non-oauth2 apps leave authOAuth false and skip all of this.
func (s *studio) initAuth() {
	scheme := provider.AuthFromContext(s.ctx)
	if scheme == nil || scheme.Type != provider.AuthOAuth2 {
		return
	}
	s.authOAuth = true
	s.authCfg = buildOAuthConfig(scheme, provider.OptionsFromContext(s.ctx))
	if tok, ok := oauth.CachedToken(s.authCfg); ok {
		s.authToken = tok
	}
}

// buildOAuthConfig maps an auth scheme and resolved options onto an oauth.Config,
// honoring flag overrides for the flow and scopes. It mirrors the headless
// mapping in clic/oauth.go, kept here so tui depends only on provider + oauth.
func buildOAuthConfig(scheme *provider.AuthScheme, opts *provider.Options) oauth.Config {
	flow := scheme.Flow
	if opts.OAuthFlow != "" {
		flow = opts.OAuthFlow
	}
	scopes := scheme.Scopes
	if len(opts.Scopes) > 0 {
		scopes = opts.Scopes
	}
	return oauth.Config{
		Flow:         flow,
		ClientID:     opts.ClientID,
		ClientSecret: opts.ClientSecret,
		AuthURL:      scheme.AuthURL,
		TokenURL:     scheme.TokenURL,
		Scopes:       scopes,
		RedirectURL:  opts.RedirectURL,
	}
}

// execCtx returns the context requests run under, injecting the studio's current
// OAuth2 access token as the bearer credential when one is held.
func (s *studio) execCtx() context.Context {
	if s.authToken == "" {
		return s.ctx
	}
	return ctxWithToken(s.ctx, s.authToken)
}

// ctxWithToken returns a context whose options carry the given bearer token,
// without mutating the shared options value.
func ctxWithToken(base context.Context, token string) context.Context {
	o := *provider.OptionsFromContext(base)
	o.Token = token
	return provider.WithOptions(base, &o)
}

// startLogin runs the OAuth2 sign-in (browser flow for authorization-code, a
// direct fetch for client-credentials) off the UI goroutine, reporting back via
// loginResultMsg.
func (s *studio) startLogin() tea.Cmd {
	if !s.authOAuth {
		s.flash = "this app does not use OAuth2"
		return nil
	}
	if s.loggingIn {
		return nil
	}
	s.loggingIn = true
	base, cfg := s.ctx, s.authCfg
	return func() tea.Msg {
		token, err := oauth.Login(base, cfg, nil)
		return loginResultMsg{token: token, err: err}
	}
}

// authStatus is the top-bar indicator for OAuth2 apps.
func (s *studio) authStatus() string {
	switch {
	case s.loggingIn:
		return s.th.latency.Render("⧗ signing in…")
	case s.authToken != "":
		return s.th.server.Render("🔓 authed")
	default:
		return s.th.latency.Render("🔒 sign in (A)")
	}
}
