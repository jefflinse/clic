package provider

import "context"

// ResultSink captures the structured result of a headless command execution.
// When a sink is present in the context, providers write their Result to it
// instead of printing to stdout, letting callers (such as the contract-test
// runner) inspect the outcome while reusing the normal command-execution path.
type ResultSink struct {
	Result *Result
}

type resultSinkCtxKey struct{}

// WithResultSink returns a context carrying the given result sink.
func WithResultSink(ctx context.Context, sink *ResultSink) context.Context {
	return context.WithValue(ctx, resultSinkCtxKey{}, sink)
}

// ResultSinkFromContext returns the result sink carried by the context, or nil
// when none is present (the normal stdout-printing path).
func ResultSinkFromContext(ctx context.Context) *ResultSink {
	if s, ok := ctx.Value(resultSinkCtxKey{}).(*ResultSink); ok {
		return s
	}
	return nil
}
