package spec_test

import (
	"testing"

	"github.com/jefflinse/handyman/spec"
	"github.com/stretchr/testify/assert"
)

func TestNewInvalidCommandSpecError(t *testing.T) {
	err := spec.NewInvalidCommandSpecError("the reason")
	assert.EqualError(t, err, "invalid command spec: the reason")
}
