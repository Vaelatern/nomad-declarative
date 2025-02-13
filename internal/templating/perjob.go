package templating

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"strings"

	"github.com/zclconf/go-cty/cty"
)

func toJson(in any) (string, error) {
	rV, err := json.Marshal(in)
	if err != nil {
		return "", err
	}
	return string(rV), err
}

// getArg navigates through `data` using `in`, and returns a string representation
// If the value is not a string, it is returned as an HCL-encoded string.
func getArg(in any, data map[string]interface{}) (string, error) {
	// Convert `in` to a string key
	key, ok := in.(string)
	if !ok {
		return "", fmt.Errorf("key must be a string")
	}

	// Lookup the value in `data`
	val, exists := data[key]
	if !exists {
		return "", fmt.Errorf("key %q not found in data", key)
	}

	// If the value is already a string, return it
	if strVal, ok := val.(string); ok {
		return strVal, nil
	}

	// Convert non-string values to HCL-compatible string
	hclStr, err := convertToHCLString(val)
	if err != nil {
		return "", err
	}

	return hclStr, nil
}

// convertToHCLString converts an arbitrary Go value to an HCL-compatible string representation
func convertToHCLString(v any) (string, error) {
	// Convert the value to JSON first
	jsonData, err := json.Marshal(v)
	if err != nil {
		return "", err
	}

	// Convert JSON to a cty.Value
	ctyVal, err := json.Unmarshal(jsonData, cty.DynamicPseudoType)
	if err != nil {
		return "", err
	}

	// Encode the cty.Value as an HCL string
	var buf bytes.Buffer
	if err := json.WriteTo(&buf, ctyVal, json.CompactOptions()); err != nil {
		return "", err
	}

	return buf.String(), nil
}
func helperFuncs() template.FuncMap {
	return template.FuncMap{
		"tojson": toJson,
		"getarg": getArg,
		"toHCL":  convertToHCLString,
	}
}

func internalTemplates(source fs.FS) []string {
	finalTpls := []string{}
	_ = fs.WalkDir(source, ".", func(entry string, d fs.DirEntry, err error) error {
		if d == nil || d.IsDir() {
			return nil
		}
		if d.Name()[0] == '_' {
			return nil
		}

		if strings.HasSuffix(d.Name(), ".tpl") {
			finalTpls = append(finalTpls, entry)
		}

		return nil
	})
	return finalTpls
}

func Template(source fs.FS, shared fs.FS) (*template.Template, error) {
	baseTemplate := template.New("base")
	var err error // avoid shadowing baseTemplate
	commonTemplates := internalTemplates(shared)
	if len(commonTemplates) > 0 {
		baseTemplate, err = baseTemplate.ParseFS(shared, commonTemplates...)
		if err != nil {
			return nil, fmt.Errorf("Can't get common templates: %v", err)
		}
	}
	jobTemplates := internalTemplates(source)
	if len(jobTemplates) > 0 {
		baseTemplate, err = baseTemplate.ParseFS(source, jobTemplates...)
		if err != nil {
			return nil, fmt.Errorf("Can't get template templates: %v", err)
		}
	}

	template := baseTemplate.
		Delims("[[", "]]").
		Option("missingkey=error").
		Funcs(helperFuncs())

	return template, nil
}

func OutputFiles(source fs.FS) (finalTpls []string, finalRaws []string) {
	_ = fs.WalkDir(source, ".", func(entry string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		}
		if d.Name()[0] == '_' {
			return nil
		}

		if strings.HasSuffix(d.Name(), ".tpl") {
			finalTpls = append(finalTpls, entry)
		} else {
			finalRaws = append(finalRaws, entry)
		}

		return nil
	})

	return
}
