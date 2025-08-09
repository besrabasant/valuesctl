package schema

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// looksLikeYAML decides if the payload should be treated as YAML (true) or JSON (false).
// Rules:
//   1) Extension hint: .json => JSON, .yaml/.yml => YAML
//   2) Sniff first non-whitespace rune (after BOM): '{' or '[' => JSON
//   3) Try JSON unmarshal: success => JSON, else => YAML
func looksLikeYAML(path string, b []byte) bool {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".json":
		return false
	case ".yaml", ".yml":
		return true
	}

	// Strip UTF-8 BOM if present, then trim leading whitespace
	b = stripUTF8BOM(b)
	trim := bytes.TrimLeft(b, " \t\r\n")

	// Empty => default to YAML
	if len(trim) == 0 {
		return true
	}

	// Check first non-whitespace rune
	r, _ := utf8.DecodeRune(trim)
	if r == '{' || r == '[' {
		return false // JSON object/array
	}

	// As a final check, see if it parses cleanly as JSON
	var v any
	if json.Unmarshal(trim, &v) == nil {
		return false
	}

	// Otherwise, assume YAML
	return true
}

func stripUTF8BOM(b []byte) []byte {
	if len(b) >= 3 && b[0] == 0xEF && b[1] == 0xBB && b[2] == 0xBF {
		return b[3:]
	}
	return b
}


func toMapStringAny(v any) (map[string]any, bool) {
	switch t := v.(type) {
	case map[string]any:
		return t, true
	case map[any]any: // from yaml.v3
		out := make(map[string]any, len(t))
		for k, vv := range t {
			out[fmt.Sprintf("%v", k)] = vv
		}
		return out, true
	default:
		return nil, false
	}
}

func cloneJSON(v any) any {
	b, _ := json.Marshal(v)
	var out any
	_ = json.Unmarshal(b, &out)
	return out
}