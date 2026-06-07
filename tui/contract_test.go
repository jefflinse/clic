package tui

import (
	"testing"

	"github.com/jefflinse/clic/provider"
	"github.com/stretchr/testify/assert"
)

func TestContractChip_Conforms(t *testing.T) {
	r := newResponsePane(newTheme())
	r.setSize(80, 20)
	res := jsonResult(`{"id":1}`)
	res.Contract = &provider.ContractResult{Checked: true, Status: "200"}
	r.setResult(res)

	assert.Contains(t, r.summary(), "conforms")
	assert.NotContains(t, r.body(), "contract violation")
}

func TestContractChip_Violations(t *testing.T) {
	r := newResponsePane(newTheme())
	r.setSize(80, 20)
	res := jsonResult(`{"id":"x"}`)
	res.Contract = &provider.ContractResult{
		Checked:    true,
		Status:     "200",
		Violations: []string{"id: value must be an integer"},
	}
	r.setResult(res)

	assert.Contains(t, r.summary(), "⚠")
	// the violation detail is surfaced as a banner above the body
	body := r.body()
	assert.Contains(t, body, "contract violation")
	assert.Contains(t, body, "id: value must be an integer")
}

func TestContractChip_NoneWhenUnchecked(t *testing.T) {
	r := newResponsePane(newTheme())
	r.setSize(80, 20)
	res := jsonResult(`{"id":1}`)
	res.Contract = &provider.ContractResult{Checked: false}
	r.setResult(res)

	s := r.summary()
	assert.NotContains(t, s, "conforms")
	assert.NotContains(t, s, "⚠")
}

func TestContractChip_NoneWhenNoContract(t *testing.T) {
	r := newResponsePane(newTheme())
	r.setSize(80, 20)
	r.setResult(jsonResult(`{"id":1}`))

	assert.NotContains(t, r.summary(), "conforms")
}
