package spec

// A Provider defines a command's behavior when invoked.
type Provider interface {
	GetParameters() ParameterSet
	IsEmpty() bool
	Name() string
	Validate() (Provider, error)
}
