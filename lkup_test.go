package main

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfig(t *testing.T) {
	config := ReadConfig()
	fmt.Println("Configured server is: ", config.Server)
	if config.Server == "" {

		t.Error("expected server name string, got", config.Server)
	}
}

func TestLogEntries(t *testing.T) {
	config := ReadConfig()
	exclude := makeExclude(config)
	rawLogEntries := parseLog("a", "", false, exclude)
	for _, rle := range rawLogEntries {
		ty := reflect.TypeOf(*rle)
		assert.Equal(t, "LogEntry", ty.Name(), "Got Non LogEntry")
	}
}

func TestSorting(t *testing.T) {
	config := ReadConfig()
	exclude := makeExclude(config)
	rawLogEntries := parseLog("a", "", false, exclude)
	perps, _ := process(rawLogEntries)
	timeIndex := perps.makeTimeIndex()
	timeIndex = timeIndex.Sort()
	for _, timeToken := range timeIndex {
		// fmt.Printf("timeToken = %v %v\n", timeToken.IP, timeToken.t)
		ty := reflect.TypeOf(*timeToken)
		assert.Equal(t, "timeTokenT", ty.Name(), "Got Non Time Token")
	}
}

func TestLocalLog(t *testing.T) {
	log := LocalLog{"error.log"}
	lines := log.ReadLines()
	// test all log lines begin with open bracket
	for _, line := range lines {
		if !strings.HasPrefix(line, "[") {
			if line != "" {
				fmt.Println(line)
				t.Fail()
			}
		}
	}
}

func TestGeoLoc(t *testing.T) {
	geoip := "http://api.ipstack.com/"
	suffix := "?access_key=2511e0d2a311aff3101c232172c9e2cf&output=json&legacy=1"
	ip := "54.245.183.198"
	// hostinfo = HostInfoType()
	geo := Geodata{}
	// error will leave default geo, which is OK
	err := getJSON(geoip+ip+suffix, &geo)
	if err != nil {
		t.Log(err, geo)
	}
	t.Log(geo)
	fmt.Println(geo)
}

func Test_makePipelines(t *testing.T) {
	type args struct {
		done  <-chan interface{}
		count int
	}
	tests := []struct {
		name  string
		args  args
		want  Chnls
		want1 Chnls
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := makePipelines(tt.args.done, tt.args.count)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("makePipelines() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("makePipelines() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
