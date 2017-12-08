package main

import (
	"fmt"
	"reflect"
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
	rawLogEntries := parseLog("e", false, "")
	for _, rle := range rawLogEntries {
		ty := reflect.TypeOf(*rle)
		assert.Equal(t, "LogEntry", ty.Name(), "Got Non LogEntry")
	}
}

func TestSorting(t *testing.T) {
	rawLogEntries := parseLog("e", false, "")
	perps, _ := process(rawLogEntries)
	timeIndex := perps.makeTimeIndex()
	timeIndex = timeIndex.Sort()
	for _, timeToken := range timeIndex {
		// fmt.Printf("timeToken = %v %v\n", timeToken.IP, timeToken.t)
		ty := reflect.TypeOf(*timeToken)
		assert.Equal(t, "timeTokenT", ty.Name(), "Got Non Time Token")
	}
}
