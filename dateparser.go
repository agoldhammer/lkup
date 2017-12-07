package main

import (
	"strings"
	"time"
)

func dparse(date string) time.Time {
	var layout string
	layout1 := "Wed Nov 22 17:15:35.339693 2017"
	layout2 := "19/Nov/2017:16:17:45 +0000"
	if strings.Contains(date, "/") {
		layout = layout2
	} else {
		layout = layout1
	}
	t, _ := time.Parse(layout, date)
	return t
}
