package spec

// A Parameter specifies a command parameter.
type Parameter struct {
	Name        string `json:"name"                  yaml:"name"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	Type        string `json:"type"                  yaml:"type"`
	Required    bool   `json:"required,omitempty"    yaml:"required,omitemty"`
	Default     string `json:"default,omitempty"     yaml:"default,omitempty"`
}
