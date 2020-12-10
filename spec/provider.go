package spec

// A Provider defines a command's behavior when invoked.
type Provider interface {
	TraceString() string
	Validate() error
}
