package ioutil

import (
	"encoding/json"
	"fmt"

	"github.com/goccy/go-yaml"
)

// Unmarshal unmarshals the supplied JSON or YAML into the target.
func Unmarshal(data []byte, v interface{}) error {
	if len(data) == 0 {
		return fmt.Errorf("nothing to unmarshal")
	}

	unmarshaler := yaml.Unmarshal
	if data[0] == '{' {
		unmarshaler = json.Unmarshal
	}

	return unmarshaler(data, v)
}
