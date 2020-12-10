package io

import (
	"encoding/json"

	"github.com/goccy/go-yaml"
)

// Unmarshal unmarshals the supplied YAML or JSON into the target.
func Unmarshal(data []byte, v interface{}) error {
	unmarshal := yaml.Unmarshal
	if data[0] == '{' {
		unmarshal = json.Unmarshal
	}

	return unmarshal(data, v)
}
