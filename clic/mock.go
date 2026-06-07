package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/jefflinse/clic/mock"
	"github.com/jefflinse/clic/source"
	"github.com/jefflinse/clic/spec"
	"github.com/spf13/cobra"
)

func mockCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mock <spec>",
		Short: "serve a mock API from an OpenAPI spec",
		Long: "Serve a stateless mock API from an OpenAPI spec: every operation " +
			"responds with a synthesized example, and incoming requests are " +
			"validated against the spec (responding 422 on a violation). Point " +
			"clic at it with --server to explore the mock interactively.",
		Args: cobra.ExactArgs(1),
		RunE: runMock,
	}

	cmd.Flags().Int("port", 9800, "port to listen on")
	cmd.Flags().Int("status", 0, "force a specific response status (default: auto-select)")
	cmd.Flags().Bool("validate-requests", true, "validate incoming requests and respond 422 on a violation")
	return cmd
}

func runMock(cmd *cobra.Command, args []string) error {
	data, err := source.Load(resolveLocation(args[0]))
	if err != nil {
		return err
	}
	if spec.DetectFormat(data) != spec.FormatOpenAPI {
		return fmt.Errorf("clic mock requires an OpenAPI spec")
	}

	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true
	doc, err := loader.LoadFromData(data)
	if err != nil {
		return fmt.Errorf("failed to parse OpenAPI document: %w", err)
	}

	status, _ := cmd.Flags().GetInt("status")
	validate, _ := cmd.Flags().GetBool("validate-requests")
	handler, routes, err := mock.Handler(doc, mock.Options{Status: status, ValidateRequests: validate})
	if err != nil {
		return err
	}

	port, _ := cmd.Flags().GetInt("port")
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	base := "http://" + addr
	fmt.Printf("clic mock serving %d route(s) at %s\n", len(routes), base)
	for _, r := range routes {
		fmt.Printf("  %-6s %s\n", r.Method, r.Path)
	}
	fmt.Printf("\nExplore it: clic --server %s %s -i\n", base, args[0])
	if validate {
		fmt.Println("Request validation is on; malformed requests get a 422.")
	}
	fmt.Println("\nPress ctrl+c to stop.")

	return serve(cmd.Context(), ln, handler)
}

// serve runs the HTTP server until the context is cancelled or an interrupt is
// received, then shuts it down gracefully.
func serve(ctx context.Context, ln net.Listener, handler http.Handler) error {
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt)
	defer stop()

	srv := &http.Server{Handler: handler}
	errc := make(chan error, 1)
	go func() { errc <- srv.Serve(ln) }()

	select {
	case <-ctx.Done():
		fmt.Println("\nshutting down…")
		return srv.Shutdown(context.Background())
	case err := <-errc:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}
