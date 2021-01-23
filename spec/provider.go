package spec

// A Provider defines a command's behavior when invoked.
type Provider interface {
	Name() string
	Validate() (Provider, error)
}
