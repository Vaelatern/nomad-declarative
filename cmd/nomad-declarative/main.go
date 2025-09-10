package main

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"

	"github.com/hairyhenderson/go-fsimpl"
	"github.com/hairyhenderson/go-fsimpl/blobfs"
	"github.com/hairyhenderson/go-fsimpl/filefs"
	"github.com/hairyhenderson/go-fsimpl/gitfs"
	"github.com/hairyhenderson/go-fsimpl/httpfs"

	"github.com/Vaelatern/nomad-declarative/internal/confparse"
	"github.com/Vaelatern/nomad-declarative/internal/submission"
	"github.com/Vaelatern/nomad-declarative/internal/templating"
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
		origin = job.Pack["origin"].(string)
	}

	if strings.HasPrefix(origin, "./") || !strings.Contains(origin, "://") {
		cwd, err := os.Getwd()
		if err == nil {
			origin = "file://" + cwd + "/" + origin
		} else {
			return fmt.Errorf("Current Working Directory for origin \"%s\" failed: %v", origin, err)
		}
	}

	if origin != DEFAULT_ORIGIN {
		mux := fsimpl.NewMux()
		mux.Add(filefs.FS)
		mux.Add(httpfs.FS)
		mux.Add(blobfs.FS)
		mux.Add(gitfs.FS)
		fsys, err := mux.Lookup(origin)
		if err != nil {
			return fmt.Errorf("Can't grab fsimpl filesystem: %v", err)
		}
		root = fsys
	}

	if _, err := fs.Stat(root, "."); err != nil {
		return fmt.Errorf("Seems like our pack root \"%s\" does not exist", root)
	}

	packRoot, err := fs.Sub(root, pack)
	if err != nil {
		return fmt.Errorf("Error grabbing pack named %s: %v", pack, err)
	}

	if _, err := fs.Stat(packRoot, "."); err != nil {
		return fmt.Errorf("Seems like our specific pack root \"%s\" does not exist", packRoot)
	}

	packTemplates, err := fs.Sub(packRoot, "templates")
	if err != nil {
		return fmt.Errorf("Error grabbing pack templates for %s: %v", pack, err)
	}
	if _, err := fs.Stat(packTemplates, "."); err != nil {
		return fmt.Errorf("Seems like we can't find the \"templates\" dir inside our pack root \"%s\"", packRoot)
	}

	var commonTemplates fs.FS
	commonRoot, _ := fs.Sub(root, "_common")
	if err != nil {
		commonTemplates, _ = fs.Sub(commonRoot, "templates")
	}

	tpls, raws, err := templating.OutputFiles(packTemplates)
	if err != nil {
		return fmt.Errorf("Error grabbing template output files: %v", err)
	}

	var tpl *template.Template
	tpl, err = templating.Template(packTemplates, commonTemplates)
	if err != nil {
		return fmt.Errorf("Can't get template: %v", err)
	}

	for _, filePath := range tpls {
		curTpl, _ := tpl.Clone()
		finalTpl, err := curTpl.ParseFS(packTemplates, filePath)
		if err != nil {
			return fmt.Errorf("Can't ParseFS in job %s @ %s, on %s: %v", job.JobName, origin, filePath, err)
		}

		// Check if the path is to be decoded
		outPath := filePath[:len(filePath)-len(".tpl")]
		var nameBuffer *bytes.Buffer
		if strings.HasPrefix(outPath, "b64(") && strings.HasSuffix(outPath, ")") {
			encoded := outPath[len("b64(") : len(outPath)-len(")")]
			decoded, err := base64.StdEncoding.DecodeString(encoded)
			if err != nil {
				return fmt.Errorf("failed to decode base64: %v", err)
			}
			nameBuffer = bytes.NewBuffer([]byte{})
			outPath = string(decoded)
			nameTpl, _ := curTpl.Clone()
			nameTpl = nameTpl.Funcs(template.FuncMap{"PASS": jobToPass.Append})
			func() {
				defer func() {
					if r := recover(); r != nil {
						fmt.Println("Panic parsing template: ", outPath, r)
					}
				}()
				nameTpl, err = nameTpl.Parse(outPath)
				if err != nil {
					log.Fatal(fmt.Errorf("Can't Parse name template %s: %v", outPath, err))
				}
				err = nameTpl.Execute(nameBuffer, jobToPass)
				if err != nil {
					log.Fatal(fmt.Errorf("Can't Execute name template %s: %v", outPath, err))
				}
			}()
		} else {
			nameBuffer = bytes.NewBufferString(outPath)
		}

		// Range on newline because it makes it easiest. Scan defaults to ScanLines
		outNames := bufio.NewScanner(nameBuffer)
		jobToPass.NameIndex = -1
		for outNames.Scan() {
			jobToPass.NameIndex += 1
			outName := outNames.Text()
			if outName == "" { // easy escape for bad templating work
				continue
			}
			// Parse job into a buffer...
			var buffer bytes.Buffer
			func() {
				defer func() {
					if r := recover(); r != nil {
						fmt.Println("Panic parsing template: ", filePath, r)
					}
				}()
				err = finalTpl.ExecuteTemplate(&buffer, filePath, jobToPass)
				if err != nil {
					log.Fatal(fmt.Errorf("Can't Execute on %s: %v", filePath, err))
				}
			}()

			// Then prepare to write and write it
			if strings.HasSuffix(outName, ".nomad") || strings.HasSuffix(outName, ".hcl") {
				formatted, diag := hclwrite.ParseConfig(buffer.Bytes(), "", hcl.Pos{Line: 1, Column: 1})
				if diag.HasErrors() {
					fmt.Printf("%v", fmt.Errorf("failed to parse HCL in %s: %s\n\n%s", outName, diag.Error(), buffer.Bytes()))
				}
				fileWrite(path.Join(job.JobName, outName), formatted.Bytes())
			} else {
				fileWrite(path.Join(job.JobName, outName), buffer.Bytes())
			}
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
		fileWrite(path.Join(job.JobName, filePath), output)
	}
	return nil
}

func chooseInsAndOuts() (string, string, bool) {
	// Define the config flag
	doExec := flag.Bool("execute", false, "self execute - run all scripts produced. Set your NOMAD_ADDR correctly first.")
	configPtr := flag.String("config", "", "path to config file")
	outputPtr := flag.String("output", "", "dir to output under")

	consumedConfigFileFromArgs := false

	// Parse the flags
	flag.Parse()
	args := flag.Args() // Gets all non-flag arguments

	var configFile string
	var outputDir string

	// Check if --config flag was provided and has a value
	if *configPtr != "" {
		configFile = *configPtr
	} else {
		// Look for first non-flag argument
		if len(args) > 0 {
			configFile = args[0]
			consumedConfigFileFromArgs = true
		} // no need for an else, the default is handled down the line
	}

	if *outputPtr != "" {
		outputDir = *outputPtr
	} else {
		if consumedConfigFileFromArgs && len(args) > 1 {
			outputDir = args[1]
		} else if !consumedConfigFileFromArgs && len(args) > 0 {
			outputDir = args[0]
		} else {
			outputDir = "./output"
		}
	}

	return configFile, outputDir, *doExec
}

func getJobs(workDir fs.FS, confFile string) (confparse.Jobs, error) {
	if confFile == "" {
		confFile = "config.d" // just in case, the error should guide people this way
		_, err := fs.Stat(workDir, "config.toml")
		if err == nil {
			// Should default to this file if it exists
			confFile = "config.toml"
		}
	}
	info, err := fs.Stat(workDir, confFile)
	if err != nil {
		return nil, fmt.Errorf("can't stat path %s: %v", confFile, err)
	}

	if !info.IsDir() {
		// Handle single file
		f, err := workDir.Open(confFile)
		if err != nil {
			return nil, fmt.Errorf("can't open config %s: %v", confFile, err)
		}
		defer f.Close()
		parsedJobs, err := confparse.ParseTOMLToJobs(f)
		if err != nil {
			return nil, fmt.Errorf("can't process config %s: %v", confFile, err)
		}
		return parsedJobs, nil
	}

	// Am a directory. Let's go a bit more complicated.
	jobs := make(confparse.Jobs)

	entries, err := fs.ReadDir(workDir, confFile)
	if err != nil {
		return nil, fmt.Errorf("can't read directory %s: %v", confFile, err)
	}
	var tomlFiles []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".toml") {
			tomlFiles = append(tomlFiles, entry.Name())
		}
	}
	sort.Strings(tomlFiles) // just make sure because last one wins the merge
	subDir, err := fs.Sub(workDir, confFile)
	if err != nil {
		return nil, fmt.Errorf("Error grabbing subDir: %v", err)
	}
	for _, name := range tomlFiles {
		f, err := subDir.Open(name)
		if err != nil {
			return nil, fmt.Errorf("can't open config file %s: %v", name, err)
		}
		parsedJobs, err := confparse.ParseTOMLToJobs(f)
		f.Close()
		if err != nil {
			return nil, fmt.Errorf("can't process config file %s: %v", name, err)
		}
		jobs = confparse.MergeJobs(jobs, parsedJobs)
	}
	return jobs, nil
}

func main() {
	workDir := os.DirFS(".")

	rootPath := DEFAULT_ORIGIN

	srcDir := os.DirFS(rootPath)

	confFile, outPath, doExec := chooseInsAndOuts()
	jobs, err := getJobs(workDir, confFile)
	if err != nil {
		log.Fatal(fmt.Errorf("Can't open and process config %v", err))
	}

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
			n, err := f.Write(contents)
			if err != nil {
				return err
			}
			// Make executable if a shebang
			if n >= 2 && contents[0] == '#' && contents[1] == '!' {
				err := os.Chmod(tgtPath, 0755)
				if err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			fmt.Println(fmt.Errorf("Failed to parse job %s: %v", job.JobName, err))
		}
	}

	if doExec {
		err := submission.ExecuteFilesAsync(outPath)
		if err != nil {
			log.Fatal(err)
		}
	}
}
