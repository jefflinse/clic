package main

import (
	"context"
	"strings"

	"github.com/jefflinse/clic/provider"
	"github.com/jefflinse/clic/spec"
	"github.com/jefflinse/clic/tui"
)

// launchStudio opens the full-screen interactive studio for the given spec. The
// passthrough args (everything after the spec) name an optional command path to
// pre-select and focus on launch.
func launchStudio(ctx context.Context, appSpec *spec.App, opts *provider.Options, passthrough []string) error {
	studioApp := tui.StudioApp{
		Name:        appSpec.Name,
		Description: appSpec.Description,
		Server:      effectiveServer(appSpec, opts),
		Commands:    toStudioCommands(appSpec.Commands),
	}

	return tui.RunStudio(ctx, studioApp, commandPath(passthrough))
}

// effectiveServer is the server URL the studio displays and requests target:
// the --server override when given, otherwise the spec's own server.
func effectiveServer(appSpec *spec.App, opts *provider.Options) string {
	if opts != nil && opts.Server != "" {
		return opts.Server
	}
	return appSpec.Server
}

// toStudioCommands maps the spec's command tree onto the studio's view of it.
func toStudioCommands(cmds []*spec.Command) []tui.Command {
	out := make([]tui.Command, 0, len(cmds))
	for _, c := range cmds {
		out = append(out, tui.Command{
			Name:        c.Name,
			Description: c.Description,
			Provider:    c.Provider,
			Subcommands: toStudioCommands(c.Subcommands),
		})
	}
	return out
}

// commandPath extracts the leading non-flag arguments as a command path to
// pre-select (e.g. ["pets","getById"] from `getById --verbose`).
func commandPath(args []string) []string {
	var path []string
	for _, a := range args {
		if strings.HasPrefix(a, "-") {
			break
		}
		path = append(path, a)
	}
	return path
}
