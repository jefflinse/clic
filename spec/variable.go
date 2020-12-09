package spec

// Variable represents a clic spec variable.
type Variable struct {
	Name  string `json:"name"  yaml:"name"`
	Type  string `json:"type"  yaml:"type"`
	Value string `json:"value" yaml:"value"`
}
