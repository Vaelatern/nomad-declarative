package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"

	"github.com/Vaelatern/nomad-declarative/internal/confparse"
	"github.com/Vaelatern/nomad-declarative/internal/templating"
)

func main() {
	workDir := os.DirFS(".")

	rootPath := "./testpacks"
	pack := "gokapi"
	srcRoot := path.Join(rootPath, pack, "templates")
	sharedRoot := path.Join(rootPath, "_common", "templates")
	srcDir := os.DirFS(srcRoot)
	sharedDir := os.DirFS(sharedRoot)
	tpls, raws := templating.OutputFiles(srcDir)

	tpl, err := templating.Template(srcDir, sharedDir)
	if err != nil {
		log.Fatal(fmt.Errorf("Can't get template: %v", err))
	}

	a, err := workDir.Open("config.toml")
	if err != nil {
		log.Fatal(fmt.Errorf("Can't open config %v", err))
	}

	jobs, _ := confparse.ParseTOMLToJobs(a)
	job := jobs["fileshare"]

	for _, path := range tpls {
		finalTpl, err := tpl.ParseFS(srcDir, path)
		if err != nil {
			log.Fatal(fmt.Errorf("Can't ParseFS on %s: %v", path, err))
		}
		var buffer bytes.Buffer
		err = finalTpl.ExecuteTemplate(&buffer, path, job)
		if err != nil {
			log.Fatal(fmt.Errorf("Can't Execute on %s: %v", path, err))
		}
		outPath := path[:len(path)-len(".tpl")]
		if strings.HasSuffix(outPath, ".nomad") || strings.HasSuffix(outPath, ".hcl") {
			formatted, diag := hclwrite.ParseConfig(buffer.Bytes(), "", hcl.Pos{Line: 1, Column: 1})
			if diag.HasErrors() {
				fmt.Printf("%v", fmt.Errorf("failed to parse HCL: %s", diag.Error()))
			}
			os.Stdout.Write(formatted.Bytes())
		}
	}

	for _, path := range raws {
		fp, err := srcDir.Open(path)
		if err != nil {
			log.Fatal(fmt.Errorf("Can't Copy %s: %v", path, err))
		}
		io.Copy(os.Stdout, fp)
	}

}
