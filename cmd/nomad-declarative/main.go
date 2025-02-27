package main

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log"
	"os"
	"strings"

	"github.com/Vaelatern/nomad-declarative/internal/confparse"
	"github.com/Vaelatern/nomad-declarative/internal/templating"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

func ParseJob(job confparse.Job, root fs.FS, fileWrite func(string, []byte) error) error {
	pack := job.Pack["name"].(string)

	packRoot, err := fs.Sub(root, pack)
	if err != nil {
		return fmt.Errorf("Error grabbing pack named %s: %v", pack, err)
	}

	packTemplates, err := fs.Sub(packRoot, "templates")
	if err != nil {
		return fmt.Errorf("Error grabbing pack templates for %s: %v", pack, err)
	}

	var commonTemplates fs.FS
	commonRoot, _ := fs.Sub(root, "_common")
	if err != nil {
		commonTemplates, _ = fs.Sub(commonRoot, "templates")
	}

	tpls, raws, err := templating.OutputFiles(packTemplates)
	if err != nil {
		return err
	}

	var tpl *template.Template
	tpl, err = templating.Template(packTemplates, commonTemplates)
	if err != nil {
		log.Fatal(fmt.Errorf("Can't get template: %v", err))
	}

	for _, path := range tpls {
		finalTpl, err := tpl.ParseFS(packTemplates, path)
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
			fileWrite(path, formatted.Bytes())
		}
	}

	for _, path := range raws {
		fp, err := packTemplates.Open(path)
		if err != nil {
			log.Fatal(fmt.Errorf("Can't Copy %s: %v", path, err))
		}
		output, err := io.ReadAll(fp)
		if err != nil {
			log.Fatal(fmt.Errorf("Can't read all contents of %s: %v", path, err))
		}
		fileWrite(path, output)
	}
	return nil
}

func main() {
	workDir := os.DirFS(".")

	rootPath := "./testpacks"

	srcDir := os.DirFS(rootPath)

	a, err := workDir.Open("config.toml")
	if err != nil {
		log.Fatal(fmt.Errorf("Can't open config %v", err))
	}

	jobs, _ := confparse.ParseTOMLToJobs(a)
	for _, job := range jobs {
		err := ParseJob(job, srcDir, func(name string, contents []byte) error {
			os.Stdout.Write(contents)
			return nil
		})
		if err != nil {
			fmt.Println(fmt.Errorf("Failed to parse job %s: %v", job.JobName, err))
		}
	}
}
