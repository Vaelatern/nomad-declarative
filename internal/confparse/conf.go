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
	JobName string
	Args    map[string]interface{}
	Pack    PackSettings
}

type Jobs map[string]Job

// ParseTOMLToJobs parses a TOML input from an io.Reader and returns a map of Jobs.
func ParseTOMLToJobs(reader io.Reader) (Jobs, error) {
	// Load the entire TOML data into a generic map
	var rawConfig map[string]map[string]interface{}
	if _, err := toml.NewDecoder(reader).Decode(&rawConfig); err != nil {
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
