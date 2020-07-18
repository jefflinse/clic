package spec_test

import (
	"testing"

	"github.com/jefflinse/handyman/spec"
	"github.com/stretchr/testify/assert"
)

func TestNewInvalidSpecError(t *testing.T) {
	err := spec.NewInvalidSpecError("the reason")
	assert.EqualError(t, err, "invalid spec: the reason")
}
