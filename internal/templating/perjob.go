package templating

import (
	"encoding/json"
	"html/template"
	"io/fs"
	"path"
)

func toJson(in any) (string, error) {
	rV, err := json.Marshal(in)
	if err != nil {
		return "", err
	}
	return string(rV), err
}

func helperFuncs() template.FuncMap {
	return template.FuncMap{
		"tojson": toJson,
	}
}

func Template(source fs.FS, subject string) (*template.Template, error) {
	base_template, err := template.ParseFS(source, "*/*/_*.tpl", "*/_*.tpl", "_*.tpl")
	if err != nil {
		return nil, err
	}

	template := base_template.
		Delims("[[", "]]").
		Option("missingkey=error").
		Funcs(helperFuncs())

	return template, nil
}

func ListTemplates(source fs.FS) []string {
	finalDirs := []string{}

	_ = fs.WalkDir(source, ".", func(entry string, d fs.DirEntry, err error) error {
		if path.Base(entry)[0] != '_' {
			finalDirs = append(finalDirs, entry)
		}
		return nil
	})

	return finalDirs
}
