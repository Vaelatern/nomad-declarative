package confparse

import (
	"io"
	"reflect"
	"strings"
	"testing"
)

func TestParseTOMLToJobs(t *testing.T) {
	tomlData := `
[pack1]
_source = "https://github.com/example/example"
job5 = {}

[pack1.job1]
job_args_1 = 123
job_arg_2 = "abc"

[pack1.job2]
job_args = 333

[pack2.job3]
job_args_1 = 456
job_arg_2 = "def"

[pack3.job4]
`

	// Define the expected output
	expectedJobs := Jobs{
		"job1": Job{
			JobName: "job1",
			Args:    JobArgs{"job_args_1": int64(123), "job_arg_2": "abc", "jobname": "job1"},
			Pack:    PackSettings{"name": "pack1", "source": "https://github.com/example/example"},
		},
		"job2": Job{
			JobName: "job2",
			Args:    JobArgs{"job_args": int64(333), "jobname": "job2"},
			Pack:    PackSettings{"name": "pack1", "source": "https://github.com/example/example"},
		},
		"job3": Job{
			JobName: "job3",
			Args:    JobArgs{"job_args_1": int64(456), "job_arg_2": "def", "jobname": "job3"},
			Pack:    PackSettings{"name": "pack2"},
		},
		"job4": Job{
			JobName: "job4",
			Args:    JobArgs{"jobname": "job4"},
			Pack:    PackSettings{"name": "pack3"},
		},
		"job5": Job{
			JobName: "job5",
			Args:    JobArgs{"jobname": "job5"},
			Pack:    PackSettings{"name": "pack1", "source": "https://github.com/example/example"},
		},
	}

	type args struct {
		reader io.Reader
	}
	tests := []struct {
		name    string
		args    args
		want    Jobs
		wantErr bool
	}{
		{
			name: "Basic jobs test",
			args: args{
				reader: strings.NewReader(tomlData),
			},
			want: expectedJobs,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := ParseTOMLToJobs(tt.args.reader)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseTOMLToJobs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseTOMLToJobs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMultipleParseTOMLToJobsAndMerge(t *testing.T) {
	tomlDataA := `
[pack1]
_source = "https://github.com/example/example"
job5 = {}

[pack1.job1]
job_args_1 = 123
job_arg_2 = "abc"

[pack1.job2]
job_args = 333

[pack2.job3]
job_args_1 = 456
job_arg_2 = "def"

[pack3.job4]
`

	tomlDataB := `
[pack1.job1]
job_args_3 = 1234

[pack3.job6]
job_args_1 = 456
job_arg_2 = "def"
`

	// Define the expected output
	expectedJobs := Jobs{
		"job1": Job{
			JobName: "job1",
			Args:    JobArgs{"job_args_1": int64(123), "job_arg_2": "abc", "jobname": "job1", "job_args_3": int64(1234)},
			Pack:    PackSettings{"name": "pack1", "source": "https://github.com/example/example"},
		},
		"job2": Job{
			JobName: "job2",
			Args:    JobArgs{"job_args": int64(333), "jobname": "job2"},
			Pack:    PackSettings{"name": "pack1", "source": "https://github.com/example/example"},
		},
		"job3": Job{
			JobName: "job3",
			Args:    JobArgs{"job_args_1": int64(456), "job_arg_2": "def", "jobname": "job3"},
			Pack:    PackSettings{"name": "pack2"},
		},
		"job4": Job{
			JobName: "job4",
			Args:    JobArgs{"jobname": "job4"},
			Pack:    PackSettings{"name": "pack3"},
		},
		"job5": Job{
			JobName: "job5",
			Args:    JobArgs{"jobname": "job5"},
			Pack:    PackSettings{"name": "pack1", "source": "https://github.com/example/example"},
		},
		"job6": Job{
			JobName: "job6",
			Args:    JobArgs{"job_args_1": int64(456), "job_arg_2": "def", "jobname": "job6"},
			Pack:    PackSettings{"name": "pack3"},
		},
	}

	jobsA, err := ParseTOMLToJobs(strings.NewReader(tomlDataA))
	if err != nil {
		t.Errorf("ParseTOMLToJobs(tomlDataA) error = %v", err)
		return
	}
	jobsB, err := ParseTOMLToJobs(strings.NewReader(tomlDataB))
	if err != nil {
		t.Errorf("ParseTOMLToJobs(tomlDataB) error = %v", err)
		return
	}

	jobsTotal := MergeJobs(jobsA, jobsB)
	if !reflect.DeepEqual(jobsTotal, expectedJobs) {
		t.Errorf("ParseTOMLToJobs() = %v, want %v", jobsTotal, expectedJobs)
	}
}
