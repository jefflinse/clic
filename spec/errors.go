package spec

import "fmt"

// NewInvalidSpecError creates a new error indicating that a spec is invalid.
func NewInvalidSpecError(reason string) error {
	return fmt.Errorf("invalid spec: %s", reason)
}
