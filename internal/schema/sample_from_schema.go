package schema

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sort"
	"strings"

	y3 "gopkg.in/yaml.v3"
	"sigs.k8s.io/yaml"
)

// BuildSampleFromSchema reads a JSON Schema (YAML or JSON) and produces a sample YAML config.
// It prefers "default", then "const", then first "enum", else a zero-ish placeholder by "type".
// For objects/arrays, it recurses into "properties"/"items".
// For anyOf/oneOf/allOf: it picks first branch (and merges for allOf).
func BuildSampleFromSchema(schemaPath string) ([]byte, error) {
	raw, err := os.ReadFile(schemaPath)
	if err != nil {
		return nil, fmt.Errorf("read schema: %w", err)
	}

	// Normalize to JSON
	schemaJSON := raw
	if looksLikeYAML(schemaPath, raw) {
		schemaJSON, err = yaml.YAMLToJSON(raw)
		if err != nil {
			return nil, fmt.Errorf("schema YAML->JSON: %w", err)
		}
	}

	var sch any
	if err := json.Unmarshal(schemaJSON, &sch); err != nil {
		return nil, fmt.Errorf("schema json unmarshal: %w", err)
	}

	sample := sampleNodeWithComments(sch)

	doc := &y3.Node{
		Kind:    y3.DocumentNode,
		Content: []*y3.Node{sample},
	}

	var buf bytes.Buffer
	enc := y3.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(doc); err != nil {
		return nil, fmt.Errorf("yaml encode: %w", err)
	}
	return bytes.TrimSpace(buf.Bytes()), nil
}

func sampleNodeWithComments(s any) *y3.Node {
	m, ok := s.(map[string]any)
	if !ok {
		// Fallback: just scalarize
		return valueToYAMLNode(nil)
	}

	// Default/const/enum take precedence and short-circuit (use the value as-is, no per-key comments)
	if v, ok := m["default"]; ok {
		return valueToYAMLNode(v)
	}
	if v, ok := m["const"]; ok {
		return valueToYAMLNode(v)
	}
	if enum, ok := m["enum"].([]any); ok && len(enum) > 0 {
		return valueToYAMLNode(enum[0])
	}

	// Combine allOf subschemas (shallow merge)
	if allOf, ok := m["allOf"].([]any); ok && len(allOf) > 0 {
		merged := map[string]any{}
		for _, sub := range allOf {
			subSample := sampleForSchemaPlain(sub)
			if subObj, ok := subSample.(map[string]any); ok {
				for k, v := range subObj {
					merged[k] = v
				}
			}
		}
		if len(merged) > 0 {
			// Use merged plain map as a value node (no per-field comments; if you
			// want comments even when allOf applies, you must compute schema-level
			// merged properties and call object builder below instead).
			return valueToYAMLNode(merged)
		}
		// fallback to first branch sample
		return sampleNodeWithComments(allOf[0])
	}

	// anyOf/oneOf: choose first branch
	if anyOf, ok := m["anyOf"].([]any); ok && len(anyOf) > 0 {
		return sampleNodeWithComments(anyOf[0])
	}
	if oneOf, ok := m["oneOf"].([]any); ok && len(oneOf) > 0 {
		return sampleNodeWithComments(oneOf[0])
	}

	switch m["type"] {
	case "object":
		// Build a mapping node and attach HeadComment from each property's description
		props, _ := m["properties"].(map[string]any)
		node := &y3.Node{Kind: y3.MappingNode}

		// Deterministic key order
		keys := sortedKeys(props)
		for _, k := range keys {
			subSchema, _ := props[k].(map[string]any)
			keyNode := &y3.Node{Kind: y3.ScalarNode, Tag: "!!str", Value: k}

			// Add description as a comment above the key
			if desc, ok := subSchema["description"].(string); ok && strings.TrimSpace(desc) != "" {
				keyNode.HeadComment = desc
			}

			valNode := sampleNodeWithComments(subSchema)
			node.Content = append(node.Content, keyNode, valNode)
		}

		// If no properties but additionalProperties is a schema, synthesize one example field
		if len(node.Content) == 0 {
			if aps, ok := m["additionalProperties"].(map[string]any); ok {
				keyNode := &y3.Node{Kind: y3.ScalarNode, Tag: "!!str", Value: "key"}
				if desc, ok := aps["description"].(string); ok && strings.TrimSpace(desc) != "" {
					keyNode.HeadComment = desc
				}
				valNode := sampleNodeWithComments(aps)
				node.Content = append(node.Content, keyNode, valNode)
			}
		}
		return node

	case "array":
		seq := &y3.Node{Kind: y3.SequenceNode}
		if items, ok := m["items"].(map[string]any); ok {
			_ = items
			// default behavior: empty array (uncomment next line to emit one placeholder element)
			// seq.Content = append(seq.Content, sampleNodeWithComments(items))
		}
		return seq

	case "string":
		// format-aware hints
		if fmtStr, ok := m["format"].(string); ok {
			switch fmtStr {
			case "date-time":
				return valueToYAMLNode("2025-01-01T00:00:00Z")
			case "date":
				return valueToYAMLNode("2025-01-01")
			case "time":
				return valueToYAMLNode("00:00:00")
			case "email":
				return valueToYAMLNode("user@example.com")
			case "hostname":
				return valueToYAMLNode("example.com")
			case "uri":
				return valueToYAMLNode("https://example.com")
			}
		}
		return valueToYAMLNode("")

	case "integer":
		return valueToYAMLNode(0)
	case "number":
		return valueToYAMLNode(0.0)
	case "boolean":
		return valueToYAMLNode(false)
	case "null":
		return valueToYAMLNode(nil)
	}

	// If "type" missing but "properties" exist, assume object
	if props, ok := m["properties"].(map[string]any); ok {
		node := &y3.Node{Kind: y3.MappingNode}
		keys := sortedKeys(props)
		for _, k := range keys {
			subSchema, _ := props[k].(map[string]any)
			keyNode := &y3.Node{Kind: y3.ScalarNode, Tag: "!!str", Value: k}
			if desc, ok := subSchema["description"].(string); ok && strings.TrimSpace(desc) != "" {
				keyNode.HeadComment = desc
			}
			valNode := sampleNodeWithComments(subSchema)
			node.Content = append(node.Content, keyNode, valNode)
		}
		return node
	}

	// If "type" missing but "items" exist, assume array
	if items, ok := m["items"].(map[string]any); ok {
		seq := &y3.Node{Kind: y3.SequenceNode}
		_ = items
		// seq.Content = append(seq.Content, sampleNodeWithComments(items))
		return seq
	}

	// Fallback
	return valueToYAMLNode(nil)
}

func sampleForSchemaPlain(s any) any {
	m, ok := s.(map[string]any)
	if !ok {
		return nil
	}

	if v, ok := m["default"]; ok {
		return v
	}
	if v, ok := m["const"]; ok {
		return v
	}
	if enum, ok := m["enum"].([]any); ok && len(enum) > 0 {
		return enum[0]
	}

	if allOf, ok := m["allOf"].([]any); ok && len(allOf) > 0 {
		merged := map[string]any{}
		for _, sub := range allOf {
			subSample := sampleForSchemaPlain(sub)
			if subObj, ok := subSample.(map[string]any); ok {
				for k, v := range subObj {
					merged[k] = v
				}
			}
		}
		if len(merged) > 0 {
			return merged
		}
		return sampleForSchemaPlain(allOf[0])
	}

	if anyOf, ok := m["anyOf"].([]any); ok && len(anyOf) > 0 {
		return sampleForSchemaPlain(anyOf[0])
	}
	if oneOf, ok := m["oneOf"].([]any); ok && len(oneOf) > 0 {
		return sampleForSchemaPlain(oneOf[0])
	}

	switch m["type"] {
	case "object":
		props, _ := m["properties"].(map[string]any)
		out := map[string]any{}
		for name, raw := range props {
			out[name] = sampleForSchemaPlain(raw)
		}
		if len(out) == 0 {
			if aps, ok := m["additionalProperties"].(map[string]any); ok {
				out["key"] = sampleForSchemaPlain(aps)
			}
		}
		return out
	case "array":
		if items, ok := m["items"].(map[string]any); ok {
			_ = items
			return []any{}
			// return []any{ sampleForSchemaPlain(items) }
		}
		return []any{}
	case "string":
		if fmtStr, ok := m["format"].(string); ok {
			switch fmtStr {
			case "date-time":
				return "2025-01-01T00:00:00Z"
			case "date":
				return "2025-01-01"
			case "time":
				return "00:00:00"
			case "email":
				return "user@example.com"
			case "hostname":
				return "example.com"
			case "uri":
				return "https://example.com"
			}
		}
		return ""
	case "integer":
		return 0
	case "number":
		return 0.0
	case "boolean":
		return false
	case "null":
		return nil
	}

	if props, ok := m["properties"].(map[string]any); ok {
		out := map[string]any{}
		for name, raw := range props {
			out[name] = sampleForSchemaPlain(raw)
		}
		return out
	}
	if items, ok := m["items"].(map[string]any); ok {
		_ = items
		return []any{}
	}
	return nil
}

// Convert a generic Go value to a yaml.Node (scalar/sequence/mapping).
func valueToYAMLNode(v any) *y3.Node {
	switch t := v.(type) {
	case nil:
		return &y3.Node{Kind: y3.ScalarNode, Tag: "!!null", Value: "null"}
	case string:
		return &y3.Node{Kind: y3.ScalarNode, Tag: "!!str", Value: t}
	case bool:
		if t {
			return &y3.Node{Kind: y3.ScalarNode, Tag: "!!bool", Value: "true"}
		}
		return &y3.Node{Kind: y3.ScalarNode, Tag: "!!bool", Value: "false"}
	case float64:
		// json numbers come as float64; if integral, render as int
		if math.Trunc(t) == t {
			return &y3.Node{Kind: y3.ScalarNode, Tag: "!!int", Value: fmt.Sprintf("%.0f", t)}
		}
		return &y3.Node{Kind: y3.ScalarNode, Tag: "!!float", Value: fmt.Sprintf("%v", t)}
	case int, int32, int64:
		return &y3.Node{Kind: y3.ScalarNode, Tag: "!!int", Value: fmt.Sprintf("%v", t)}
	case []any:
		seq := &y3.Node{Kind: y3.SequenceNode}
		for _, e := range t {
			seq.Content = append(seq.Content, valueToYAMLNode(e))
		}
		return seq
	case map[string]any:
		mp := &y3.Node{Kind: y3.MappingNode}
		ks := sortedKeys(t)
		for _, k := range ks {
			mp.Content = append(mp.Content, &y3.Node{Kind: y3.ScalarNode, Tag: "!!str", Value: k}, valueToYAMLNode(t[k]))
		}
		return mp
	case map[any]any: // in case defaults came from YAML decoding elsewhere
		// convert to map[string]any first
		out := make(map[string]any, len(t))
		for k, v := range t {
			out[fmt.Sprintf("%v", k)] = v
		}
		return valueToYAMLNode(out)
	default:
		// fallback stringification
		return &y3.Node{Kind: y3.ScalarNode, Tag: "!!str", Value: fmt.Sprintf("%v", t)}
	}
}

// ---------------------- helpers ----------------------

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
