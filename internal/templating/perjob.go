package templating

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"strings"
)

func toJson(in any) (string, error) {
	rV, err := json.Marshal(in)
	if err != nil {
		return "", err
	}
	return string(rV), err
}

func unquote(in template.HTML) (template.HTML, error) {
	startIndex := 0
	endIndex := len(in)
	if in[0] == '"' {
		startIndex += 1
	}
	if in[len(in)-1] == '"' {
		endIndex -= 1
	}
	return in[startIndex:endIndex], nil
}

func helperFuncs() template.FuncMap {
	return template.FuncMap{
		"tojson":  toJson,
		"getarg":  getArg,
		"unquote": unquote,
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
