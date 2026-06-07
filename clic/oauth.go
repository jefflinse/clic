package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/jefflinse/clic"
	"github.com/jefflinse/clic/oauth"
	"github.com/jefflinse/clic/provider"
	"github.com/jefflinse/clic/spec"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
)

// oauthConfig maps a spec's oauth2 auth scheme and clic's resolved options onto
// an oauth.Config. Flag-provided scopes and flow override the spec's defaults.
// It lives here (not in the oauth package) to keep oauth provider-free.
func oauthConfig(scheme *provider.AuthScheme, opts *provider.Options) oauth.Config {
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

// resolveOAuth obtains an access token for an oauth2-secured spec and stores it
// in opts.Token (the bearer path then applies it). It reuses any cached token,
// auto-launches the browser login when one is needed and a terminal is attached,
// and otherwise tells the user to run `clic login`. It is a no-op for non-oauth2
// specs.
func resolveOAuth(ctx context.Context, scheme *provider.AuthScheme, opts *provider.Options) error {
	if scheme == nil || scheme.Type != provider.AuthOAuth2 {
		return nil
	}
	cfg := oauthConfig(scheme, opts)
	if cfg.TokenURL == "" {
		return fmt.Errorf("oauth2: spec declares no token URL")
	}

	token, err := oauth.Token(ctx, cfg)
	if errors.Is(err, oauth.ErrLoginRequired) {
		if !isInteractiveTerminal() {
			return fmt.Errorf("oauth2 login required: run `clic login <spec>` first")
		}
		token, err = oauth.Login(ctx, cfg, nil)
	}
	if err != nil {
		return fmt.Errorf("oauth2: %w", err)
	}

	opts.Token = token
	return nil
}

// isInteractiveTerminal reports whether stdout is a terminal, so we only auto-
// launch a browser for a human at a prompt (never in scripts or CI).
func isInteractiveTerminal() bool {
	return isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsCygwinTerminal(os.Stdout.Fd())
}

// loginCmd authenticates an OAuth2-secured spec and caches the token, running
// the browser flow for authorization-code grants.
func loginCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "login <spec>",
		Short: "authenticate an OAuth2-secured spec and cache the access token",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := oauthForSpec(cmd, args[0])
			if err != nil {
				return err
			}
			if _, err := oauth.Login(cmd.Context(), cfg, nil); err != nil {
				return err
			}
			fmt.Printf("Authenticated. Token cached for %s.\n", args[0])
			return nil
		},
	}
}

// logoutCmd removes the cached OAuth2 token for a spec.
func logoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout <spec>",
		Short: "remove the cached OAuth2 token for a spec",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := oauthForSpec(cmd, args[0])
			if err != nil {
				return err
			}
			if err := oauth.Logout(cfg); err != nil {
				return err
			}
			fmt.Printf("Removed cached token for %s.\n", args[0])
			return nil
		},
	}
}

// oauthForSpec loads a spec, verifies it uses OAuth2, and builds its oauth.Config
// from the resolved global options.
func oauthForSpec(cmd *cobra.Command, location string) (oauth.Config, error) {
	appSpec, err := clic.LoadSpec(resolveLocation(location), spec.FormatUnknown)
	if err != nil {
		return oauth.Config{}, err
	}
	if appSpec.Auth == nil || appSpec.Auth.Type != provider.AuthOAuth2 {
		return oauth.Config{}, fmt.Errorf("%s has no OAuth2 authentication", location)
	}
	return oauthConfig(appSpec.Auth, provider.ResolveOptions(cmd.Flags())), nil
}
