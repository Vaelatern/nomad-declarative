package confparse

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"text/template"

	"github.com/BurntSushi/toml"
	"github.com/Masterminds/sprig/v3"
)

func TemplateSuperpowers(r io.Reader) (io.Reader, error) {
	// Read input
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	tmpl, err := template.New("config").Parse(string(data))
	if err != nil {
		return nil, err
	}

	// Build function map
	funcMap := sprig.FuncMap()

	// env(key string, default...string) string
	funcMap["env"] = func(key string, defaultVal ...string) string {
		if val := os.Getenv(key); val != "" {
			return val
		}
		if len(defaultVal) > 0 {
			return defaultVal[0]
		}
		return ""
	}

	// toToml(v any) string
	funcMap["toToml"] = func(v any) string {
		var buf bytes.Buffer
		enc := toml.NewEncoder(&buf)
		if err := enc.Encode(v); err != nil {
			return fmt.Sprintf("[toToml error: %v]", err)
		}
		return buf.String()
	}

	// Apply functions to template
	tmpl = tmpl.Funcs(funcMap)

	// Execute template with input data as string (or keep as []byte if preferred)
	var out bytes.Buffer
	if err := tmpl.Execute(&out, string(data)); err != nil {
		return nil, err
	}

	return bytes.NewReader(out.Bytes()), nil
}
