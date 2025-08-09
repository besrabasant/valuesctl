package tmpl

import (
	"bytes"
	"fmt"
	"os"
	"text/template"

	y3 "gopkg.in/yaml.v3"
)

// RenderFromFiles (unchanged; still available if you need it elsewhere)
func RenderFromFiles(tplPath, cfgPath string) ([]byte, error) {
	cfgBytes, err := os.ReadFile(cfgPath)
	if err != nil {
		return nil, err
	}
	var data map[string]any
	if err := y3.Unmarshal(cfgBytes, &data); err != nil {
		return nil, err
	}
	return RenderWithData(tplPath, data)
}

// RenderWithData executes the template with a provided data map.
func RenderWithData(tplPath string, data map[string]any) ([]byte, error) {
	tplBytes, err := os.ReadFile(tplPath)
	if err != nil {
		return nil, err
	}
	tpl := template.Must(template.New("values").Funcs(template.FuncMap{
		"csv": func(ss any) string {
			switch v := ss.(type) {
			case []string:
				return joinStrings(v, ",")
			case []any:
				out := make([]string, len(v))
				for i := range v {
					out[i] = fmt.Sprintf("%v", v[i])
				}
				return joinStrings(out, ",")
			default:
				return ""
			}
		},
		"jsonarr": func(ss any) string {
			var elems []string
			switch v := ss.(type) {
			case []string:
				elems = v
			case []any:
				elems = make([]string, len(v))
				for i := range v {
					elems[i] = fmt.Sprintf("%v", v[i])
				}
			default:
				return "[]"
			}
			var b bytes.Buffer
			b.WriteByte('[')
			for i, s := range elems {
				if i > 0 {
					b.WriteByte(',')
				}
				b.WriteByte('"')
				for _, r := range s {
					if r == '"' {
						b.WriteString(`\"`)
					} else {
						b.WriteRune(r)
					}
				}
				b.WriteByte('"')
			}
			b.WriteByte(']')
			return b.String()
		},
	}).Option("missingkey=error").Parse(string(tplBytes)))

	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func joinStrings(in []string, sep string) string {
	switch len(in) {
	case 0:
		return ""
	case 1:
		return in[0]
	default:
		var b bytes.Buffer
		for i, s := range in {
			if i > 0 {
				b.WriteString(sep)
			}
			b.WriteString(s)
		}
		return b.String()
	}
}
