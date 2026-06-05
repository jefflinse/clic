package clic

import (
	"fmt"

	"github.com/jefflinse/clic/openapi"
	"github.com/jefflinse/clic/source"
	"github.com/jefflinse/clic/spec"
)

// LoadSpec loads a spec from a file path or URL, determines its format (honoring
// the forced format when not FormatUnknown), and returns the compiled clic spec.
// OpenAPI documents are compiled to a clic spec; clic specs are parsed directly.
func LoadSpec(location string, force spec.Format) (*spec.App, error) {
	data, err := source.Load(location)
	if err != nil {
		return nil, err
	}

	detected := spec.DetectFormat(data)

	format := force
	if format == spec.FormatUnknown {
		format = detected
	} else if detected != spec.FormatUnknown && detected != format {
		return nil, fmt.Errorf("%s: expected a %s spec but it looks like %s", location, format, detected)
	}

	switch format {
	case spec.FormatOpenAPI:
		return openapi.Compile(data)
	case spec.FormatClic:
		return spec.NewAppSpec(data)
	default:
		return nil, fmt.Errorf("could not determine the format of %q; use --openapi or --spec to force it", location)
	}
}
