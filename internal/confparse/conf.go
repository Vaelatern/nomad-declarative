package confparse

import (
	"fmt"
	"io"

	"github.com/BurntSushi/toml"
)

// Expected values for PackSettings keys:
// name -- set automatically
// origin -- Defaults to "./packs"
// origin-name -- Defaults to the name set automatically
type PackSettings map[string]interface{}
type JobArgs map[string]interface{}

type Job struct {
	JobName string
	Args    JobArgs
	Pack    PackSettings
}

type JobAsArgs struct {
	JobName   string
	Args      map[string]interface{}
	Pack      PackSettings
	FileName  string
	NameArgs  []interface{}
	NameIndex int
}

func (j *JobAsArgs) Append(in any) (string, error) {
	j.NameArgs = append(j.NameArgs, in)
	return "", nil
}

type Jobs map[string]Job

// ParseTOMLToJobs parses a TOML input from an io.Reader and returns a map of Jobs.
// It is important for later merging that there are no extra defaults set here.
func ParseTOMLToJobs(reader io.Reader) (Jobs, error) {
	wrappedReader, err := TemplateSuperpowers(reader)
	if err != nil {
		return nil, fmt.Errorf("Failed to go-template the toml itself: %v", err)
	}

	// Load the entire TOML data into a generic map
	var rawConfig map[string]map[string]interface{}
	if _, err := toml.NewDecoder(wrappedReader).Decode(&rawConfig); err != nil {
		return nil, fmt.Errorf("failed to decode TOML: %w", err)
	}

	jobs := make(Jobs)

	// Iterate over the packs and jobs
	for packName, packContents := range rawConfig {
		// Extract pack-level arguments (if any)
		packArgs := make(PackSettings)
		packArgs["name"] = packName
		// first populate these...
		for k, v := range packContents {
			if k[0] == '_' {
				packArgs[k[1:]] = v
			}
		}

		// then do the jobs
		for jobName, jobArgs := range packContents {
			if _, ok := jobArgs.(map[string]interface{}); !ok || jobName[0] == '_' {
				continue
			}
			jobArgsAsDict := jobArgs.(map[string]interface{})
			if _, ok := jobArgsAsDict["jobname"]; !ok {
				jobArgsAsDict["jobname"] = jobName
			}
			jobs[jobName] = Job{
				JobName: jobName,
				Args:    jobArgsAsDict,
				Pack:    packArgs,
			}
		}
	}

	return jobs, nil
}

func MergeJobs(a Jobs, override Jobs) Jobs {
	result := make(Jobs)

	// Copy all jobs from 'a' to result
	for jobName, job := range a {
		result[jobName] = Job{
			JobName: job.JobName,
			Args:    make(JobArgs),
			Pack:    make(PackSettings),
		}
		// Copy Args
		for k, v := range job.Args {
			result[jobName].Args[k] = v
		}
		// Copy Pack
		for k, v := range job.Pack {
			result[jobName].Pack[k] = v
		}
	}

	// Apply overrides
	for jobName, overrideJob := range override {
		// Get or initialize the job in result
		job, exists := result[jobName]
		if !exists {
			job = Job{
				JobName: overrideJob.JobName,
				Args:    make(JobArgs),
				Pack:    make(PackSettings),
			}
		}

		// Override JobName if non-empty
		if overrideJob.JobName != "" {
			job.JobName = overrideJob.JobName
		}

		// Override Args
		for k, v := range overrideJob.Args {
			job.Args[k] = v
		}

		// Override Pack
		for k, v := range overrideJob.Pack {
			job.Pack[k] = v
		}

		// Reassign the modified job back to the map
		result[jobName] = job
	}

	return result
}
