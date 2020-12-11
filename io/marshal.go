package io

import (
	"encoding/json"
	"log"

	"github.com/goccy/go-yaml"
)

// Unmarshal unmarshals the supplied YAML or JSON into the target.
func Unmarshal(data []byte, v interface{}) error {
	var unmarshal func(data []byte, v interface{}) error
	if data[0] == '{' {
		unmarshal = json.Unmarshal
		log.Println("unmarshaling JSON")
	} else {
		unmarshal = yaml.Unmarshal
		log.Println("unmarshaling YAML")
	}

	return unmarshal(data, v)
}
