package oauth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/pkg/browser"
	"golang.org/x/oauth2"
)

// authCodeTimeout bounds how long we wait for the user to complete the browser
// login before giving up.
const authCodeTimeout = 3 * time.Minute

// loginAuthCode runs the authorization-code + PKCE flow: it stands up a loopback
// callback server, opens the browser to the consent screen, and exchanges the
// returned code (bound by the PKCE verifier) for a token. opener may be nil to
// use the system browser.
func loginAuthCode(ctx context.Context, cfg Config, opener Opener) (*oauth2.Token, error) {
	redirect := cfg.redirectURL()
	u, err := url.Parse(redirect)
	if err != nil {
		return nil, fmt.Errorf("oauth2: invalid redirect URL %q: %w", redirect, err)
	}

	ln, err := net.Listen("tcp", u.Host)
	if err != nil {
		return nil, fmt.Errorf("oauth2: cannot listen on %s for the redirect (set --redirect-url): %w", u.Host, err)
	}
	defer ln.Close()

	verifier := oauth2.GenerateVerifier()
	state, err := randomState()
	if err != nil {
		return nil, err
	}

	oc := &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		Endpoint:     oauth2.Endpoint{AuthURL: cfg.AuthURL, TokenURL: cfg.TokenURL},
		RedirectURL:  redirect,
		Scopes:       cfg.Scopes,
	}
	authURL := oc.AuthCodeURL(state,
		oauth2.AccessTypeOffline, // request a refresh token
		oauth2.S256ChallengeOption(verifier),
	)

	ctx, cancel := context.WithTimeout(ctx, authCodeTimeout)
	defer cancel()

	type callback struct {
		code string
		err  error
	}
	results := make(chan callback, 1)

	mux := http.NewServeMux()
	mux.HandleFunc(u.Path, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if e := q.Get("error"); e != "" {
			writePage(w, "Authorization failed. You can close this tab.")
			results <- callback{err: fmt.Errorf("oauth2: authorization error: %s", e)}
			return
		}
		if q.Get("state") != state {
			writePage(w, "Authorization failed (state mismatch). You can close this tab.")
			results <- callback{err: fmt.Errorf("oauth2: state mismatch")}
			return
		}
		writePage(w, "Authorization complete. You can close this tab and return to clic.")
		results <- callback{code: q.Get("code")}
	})

	srv := &http.Server{Handler: mux}
	go srv.Serve(ln)
	defer srv.Shutdown(context.Background())

	if opener == nil {
		opener = browser.OpenURL
	}
	_ = opener(authURL)
	fmt.Fprintf(os.Stderr, "Opening your browser to sign in. If it doesn't open, visit:\n\n  %s\n\n", authURL)

	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("oauth2: timed out waiting for browser login: %w", ctx.Err())
	case res := <-results:
		if res.err != nil {
			return nil, res.err
		}
		tok, err := oc.Exchange(ctx, res.code, oauth2.VerifierOption(verifier))
		if err != nil {
			return nil, fmt.Errorf("oauth2: token exchange failed: %w", err)
		}
		return tok, nil
	}
}

// randomState returns a cryptographically-random CSRF state value.
func randomState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("oauth2: generating state: %w", err)
	}
	return hex.EncodeToString(b), nil
}

func writePage(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, "<!doctype html><html><body style=\"font-family:sans-serif;padding:3rem;text-align:center\"><h2>clic</h2><p>%s</p></body></html>", msg)
}
