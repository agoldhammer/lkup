package main

import (
	"os"
	"reflect"
	"testing"
)

func Test_makeLog(t *testing.T) {
	type args struct {
		fname string
	}
	myargs := args{"test.log"}
	tests := []struct {
		name string
		args args
		want string
	}{
		{"file", myargs, "test.log"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := makeLogsrc(tt.args.fname); got.fname != tt.args.fname {
				t.Errorf("Logsrc %v has wrong fname %v", got, tt.want)
			}
		})
	}
}

func Test_makeLogsrcFromStdin(t *testing.T) {
	tests := []struct {
		name string
	}{
		{"stdin test"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := makeLogsrcFromStdin(); got.file != os.Stdin {
				t.Errorf("makeLogsrcFromStdin() = %v does not have file = os.Stdin", got)
			}
		})
	}
}

func TestReadConfig(t *testing.T) {
	testconfig := Config{Server: "", Omit: "71.192.181.208"}
	tests := []struct {
		name string
		want Config
	}{
		{"config reader test", testconfig},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ReadConfig(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ReadConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}
