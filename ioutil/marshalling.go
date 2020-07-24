package ioutil

import (
	"encoding/json"
	"fmt"

	"github.com/goccy/go-yaml"
)

// Intermarshal marshals an object of unknown type to JSON and then unmarshals it into the target type.
func Intermarshal(source interface{}, target interface{}) error {
	data, err := json.Marshal(source)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, &target); err != nil {
		return err
	}

	return nil
}

// Unmarshal unmarshals the supplied JSON or YAML into the target.
func Unmarshal(data []byte, target interface{}) error {
	if len(data) == 0 {
		return fmt.Errorf("empty data encountered")
	}

	unmarshaler := yaml.Unmarshal
	if data[0] == '{' {
		unmarshaler = json.Unmarshal
	}

	return unmarshaler(data, target)
}
