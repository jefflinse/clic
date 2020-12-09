package spec

// Command represents a command line command or subcommand.
type Command struct {
	Name        string `json:"name"                  yaml:"name"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	Type        string `json:"type"                  yaml:"type"`
}
