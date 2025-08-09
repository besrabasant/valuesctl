package schema

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/xeipuuv/gojsonschema"
	"sigs.k8s.io/yaml"
)

// ValidateYAMLWithSchema validates YAML config against a JSON Schema (YAML or JSON) using gojsonschema.
func ValidateYAMLWithSchema(schemaPath, dataPath string) error {
	sraw, err := os.ReadFile(schemaPath)
	if err != nil {
		return fmt.Errorf("read schema: %w", err)
	}
	draw, err := os.ReadFile(dataPath)
	if err != nil {
		return fmt.Errorf("read data: %w", err)
	}

	// Convert YAML to JSON where needed
	schemaJSON := sraw
	if looksLikeYAML(schemaPath, sraw) {
		schemaJSON, err = yaml.YAMLToJSON(sraw)
		if err != nil {
			return fmt.Errorf("schema YAML->JSON: %w", err)
		}
	}
	dataJSON := draw
	if looksLikeYAML(dataPath, draw) {
		dataJSON, err = yaml.YAMLToJSON(draw)
		if err != nil {
			return fmt.Errorf("data YAML->JSON: %w", err)
		}
	}

	sl := gojsonschema.NewBytesLoader(schemaJSON)
	dl := gojsonschema.NewBytesLoader(dataJSON)
	res, err := gojsonschema.Validate(sl, dl)
	if err != nil {
		return fmt.Errorf("jsonschema validate error: %w", err)
	}
	if res.Valid() {
		return nil
	}
	var b strings.Builder
	for _, e := range res.Errors() {
		// e.Context().String() includes path; e.Description() explains
		fmt.Fprintf(&b, "- %s: %s\n", e.Field(), e.Description())
	}
	return errors.New("schema validation failed:\n" + b.String())
}
