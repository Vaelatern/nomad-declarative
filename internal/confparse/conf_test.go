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
			Args:    JobArgs{"job_args_1": int64(123), "job_arg_2": "abc"},
			Pack:    PackSettings{"name": "pack1", "source": "https://github.com/example/example"},
		},
		"job2": Job{
			JobName: "job2",
			Args:    JobArgs{"job_args": int64(333)},
			Pack:    PackSettings{"name": "pack1", "source": "https://github.com/example/example"},
		},
		"job3": Job{
			JobName: "job3",
			Args:    JobArgs{"job_args_1": int64(456), "job_arg_2": "def"},
			Pack:    PackSettings{"name": "pack2"},
		},
		"job4": Job{
			JobName: "job4",
			Args:    JobArgs{},
			Pack:    PackSettings{"name": "pack3"},
		},
		"job5": Job{
			JobName: "job5",
			Args:    JobArgs{},
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
