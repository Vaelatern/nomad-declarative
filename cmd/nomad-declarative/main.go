package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path"

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
		err = finalTpl.ExecuteTemplate(os.Stdout, path, job)
		if err != nil {
			log.Fatal(fmt.Errorf("Can't Execute on %s: %v", path, err))
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
