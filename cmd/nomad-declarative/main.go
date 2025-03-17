package main

import (
	"bytes"
	"flag"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/Vaelatern/nomad-declarative/internal/confparse"
	"github.com/Vaelatern/nomad-declarative/internal/templating"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

const DEFAULT_ORIGIN = "./packs"

func ParseJob(job confparse.Job, root fs.FS, fileWrite func(string, []byte) error) error {
	var jobToPass confparse.JobAsArgs
	jobToPass.Pack = job.Pack
	jobToPass.Args = job.Args
	jobToPass.JobName = job.JobName
	pack := job.Pack["name"].(string)
	if job.Pack["origin-name"] != nil && job.Pack["origin-name"].(string) != "" {
		pack = job.Pack["origin-name"].(string)
	}
	origin := DEFAULT_ORIGIN
	if job.Pack["origin"] != nil && job.Pack["origin"].(string) != "" {
		origin = job.Pack["origin-name"].(string)
	}

	if origin != DEFAULT_ORIGIN {
		root = os.DirFS(origin)
	}

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

	for _, filePath := range tpls {
		finalTpl, err := tpl.ParseFS(packTemplates, filePath)
		if err != nil {
			log.Fatal(fmt.Errorf("Can't ParseFS on %s: %v", filePath, err))
		}
		var buffer bytes.Buffer
		err = finalTpl.ExecuteTemplate(&buffer, filePath, jobToPass)
		if err != nil {
			log.Fatal(fmt.Errorf("Can't Execute on %s: %v", filePath, err))
		}
		outPath := filePath[:len(filePath)-len(".tpl")]
		if strings.HasSuffix(outPath, ".nomad") || strings.HasSuffix(outPath, ".hcl") {
			formatted, diag := hclwrite.ParseConfig(buffer.Bytes(), "", hcl.Pos{Line: 1, Column: 1})
			if diag.HasErrors() {
				fmt.Printf("%v", fmt.Errorf("failed to parse HCL: %s", diag.Error()))
			}
			fileWrite(path.Join(pack, filePath), formatted.Bytes())
		}
	}

	for _, filePath := range raws {
		fp, err := packTemplates.Open(filePath)
		if err != nil {
			log.Fatal(fmt.Errorf("Can't Copy %s: %v", filePath, err))
		}
		output, err := io.ReadAll(fp)
		if err != nil {
			log.Fatal(fmt.Errorf("Can't read all contents of %s: %v", filePath, err))
		}
		fileWrite(path.Join(pack, filePath), output)
	}
	return nil
}

func chooseConfigFile() string {
	// Define the config flag
	configPtr := flag.String("config", "", "path to config file")

	// Parse the flags
	flag.Parse()

	// Default configFile value
	var configFile string

	// Check if --config flag was provided and has a value
	if *configPtr != "" {
		configFile = *configPtr
	} else {
		// Look for first non-flag argument
		args := flag.Args() // Gets all non-flag arguments
		if len(args) > 0 {
			configFile = args[0]
		} else {
			// Set a default if no config specified (optional)
			configFile = "config.toml"
		}
	}

	return configFile
}

func main() {
	outPath := "./output"
	workDir := os.DirFS(".")

	rootPath := DEFAULT_ORIGIN

	srcDir := os.DirFS(rootPath)

	a, err := workDir.Open(chooseConfigFile())
	if err != nil {
		log.Fatal(fmt.Errorf("Can't open config %v", err))
	}

	jobs, _ := confparse.ParseTOMLToJobs(a)
	for _, job := range jobs {
		err := ParseJob(job, srcDir, func(name string, contents []byte) error {
			tgtPath := filepath.Join(outPath, name)
			tgtDirPath := filepath.Dir(tgtPath)
			err := os.MkdirAll(tgtDirPath, 0755)
			if err != nil {
				return err
			}
			f, err := os.Create(tgtPath)
			if err != nil {
				return err
			}
			defer f.Close()
			_, err = f.Write(contents)
			if err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			fmt.Println(fmt.Errorf("Failed to parse job %s: %v", job.JobName, err))
		}
	}
}
