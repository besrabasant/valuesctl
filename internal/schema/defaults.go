package schema

import (
	"encoding/json"
	"fmt"
	"os"

	y3 "gopkg.in/yaml.v3"
	"sigs.k8s.io/yaml"
)

// LoadConfigWithSchemaDefaults reads schema (YAML/JSON) and config (YAML),
// and if applyDefaults==true it merges "default" values from the schema into the config.
// Returns a map[string]any ready for templating.
func LoadConfigWithSchemaDefaults(schemaPath, cfgPath string, applyDefaults bool) (map[string]any, error) {
	// Load config YAML -> map
	cfgBytes, err := os.ReadFile(cfgPath)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg any
	if err := y3.Unmarshal(cfgBytes, &cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config yaml: %w", err)
	}

	// If not applying defaults, just coerce to map and return.
	if !applyDefaults || schemaPath == "" {
		m, _ := toMapStringAny(cfg)
		return m, nil
	}

	// Load schema (YAML or JSON) -> JSON -> interface{}
	sraw, err := os.ReadFile(schemaPath)
	if err != nil {
		return nil, fmt.Errorf("read schema: %w", err)
	}
	schemaJSON := sraw
	if looksLikeYAML(schemaPath, sraw) {
		schemaJSON, err = yaml.YAMLToJSON(sraw)
		if err != nil {
			return nil, fmt.Errorf("schema YAML->JSON: %w", err)
		}
	}
	var sch any
	if err := json.Unmarshal(schemaJSON, &sch); err != nil {
		return nil, fmt.Errorf("schema json unmarshal: %w", err)
	}

	// Apply defaults recursively (mutates cfg)
	cfg = applyDefaultsNode(sch, cfg)

	// Ensure map[string]any
	out, _ := toMapStringAny(cfg)
	return out, nil
}

func applyDefaultsNode(schema any, cfg any) any {
	sm, ok := schema.(map[string]any)
	if !ok {
		return cfg
	}

	// If the schema node has a "default" and cfg is nil, use it.
	if def, ok := sm["default"]; ok && (cfg == nil) {
		return cloneJSON(def)
	}

	// allOf: merge subschema defaults into cfg
	if arr, ok := sm["allOf"].([]any); ok {
		for _, sub := range arr {
			cfg = applyDefaultsNode(sub, cfg)
		}
	}

	// oneOf/anyOf: we *could* pick the first; for defaults, we won’t guess.
	// If you want picking-first behavior, uncomment one of these:
	// if arr, ok := sm["oneOf"].([]any); ok && len(arr) > 0 { cfg = applyDefaultsNode(arr[0], cfg) }
	// if arr, ok := sm["anyOf"].([]any); ok && len(arr) > 0 { cfg = applyDefaultsNode(arr[0], cfg) }

	switch sm["type"] {
	case "object":
		props, _ := sm["properties"].(map[string]any)
		// If cfg is nil and schema has object default, handled above.
		// If cfg is nil and no default: start with empty object to receive per-property defaults
		cm, _ := toMapStringAny(cfg)
		if cm == nil {
			cm = map[string]any{}
		}
		for name, sub := range props {
			cur := cm[name]
			cm[name] = applyDefaultsNode(sub, cur)
		}
		return cm

	case "array":
		// If schema has a default for array and cfg==nil, handled above.
		// If cfg is an array and items has defaults for elements, we don’t synthesize elements.
		// (We keep user data intact.)
		return cfg

	default:
		// primitives: if default exists and cfg==nil it was already set
		return cfg
	}
}

