package spec

// An App represents a complete clic application.
type App struct {
	Name        string    `json:"name"                  yaml:"name"`
	Description string    `json:"description,omitempty" yaml:"description,omitempty"`
	Commands    []Command `json:"commands"              yaml:"commands"`
}
