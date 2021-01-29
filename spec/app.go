package spec

import (
	"fmt"
)

// An App represents a clic app.
type App struct {
	Name     string    `json:"name"`
	Commands []Command `json:"commands"`
}

// MergeInto merges this app spec into another one, returning the combined spec.
func (a App) MergeInto(other App) (App, error) {
	if a.Name != other.Name {
		return a, fmt.Errorf("failed to merge app specs: names '%s' and '%s' do not match", a.Name, other.Name)
	}

	merged := other
	for _, incoming := range a.Commands {
		for _, current := range merged.Commands {
			if incoming.Name == current.Name {
				return App{}, fmt.Errorf("failed to merge app specs: multiple definitions for '%s' command", current.Name)
			}
			merged.Commands = append(merged.Commands, incoming)
		}
	}

	return merged, nil
}

// Validate returns an error if the app spec is invalid.
func (a App) Validate() (App, error) {
	if a.Name == "" {
		return a, fmt.Errorf("invalid app spec: missing name")
	}

	vcs := []Command{}
	for _, c := range a.Commands {
		vc, err := c.Validate()
		if err != nil {
			return a, err
		}

		vcs = append(vcs, vc)
	}

	return App{
		Name:     a.Name,
		Commands: vcs,
	}, nil
}

// MergeAppSpecs merges multiple app specs into a single one.
func MergeAppSpecs(specs ...App) (App, error) {
	if len(specs) == 0 {
		panic("MergeAppSpecs() called with too few app specs")
	}

	merged := specs[0]
	var err error
	for i := 1; i < len(specs); i++ {
		merged, err = specs[i].MergeInto(merged)
		if err != nil {
			return App{}, err
		}
	}

	return merged, nil
}
