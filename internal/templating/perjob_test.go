package templating

import (
	"html/template"
	"reflect"
	"testing"
)

func Test_toJson(t *testing.T) {
	type args struct {
		in any
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name:    "",
			args:    args{in: []string{}},
			want:    "[]",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := toJson(tt.args.in)
			if (err != nil) != tt.wantErr {
				t.Errorf("toJson() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("toJson() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_helperFuncs(t *testing.T) {
	tests := []struct {
		name string
		want template.FuncMap
	}{
		// This doesn't work because the function pointer is different
		// {
		// 	name: "",
		// 	want: template.FuncMap{
		// 		"tojson": toJson,
		// 	},
		// },
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := helperFuncs(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("helperFuncs() = %v, want %v", got, tt.want)
			}
		})
	}
}
