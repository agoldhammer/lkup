package main

import (
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/araddon/dateparse"
)

type datepartsType struct {
	Day       string
	MonthName string
	Year      string
	Time      string
	Zone      string
}

func rewriteDate(date string) string {

	monthMap := make(map[string]string)
	monthMap["Jan"] = "01"
	monthMap["Feb"] = "02"
	monthMap["Mar"] = "03"
	monthMap["Apr"] = "04"
	monthMap["May"] = "05"
	monthMap["Jun"] = "06"
	monthMap["Jul"] = "07"
	monthMap["Aug"] = "08"
	monthMap["Sep"] = "09"
	monthMap["Oct"] = "10"
	monthMap["Nov"] = "11"
	monthMap["Dec"] = "12"
	// This regexp splits date into 5 parts
	// day month year time +0000
	sepdatetime := `(\w+)/(\w+)/(\w+):(\S+)\s(\+0000)`
	re := regexp.MustCompile(sepdatetime)
	result := re.FindAllStringSubmatch(date, -1)
	// fmt.Printf("result = %+v\n", result)
	parts := result[0]
	var dateparts datepartsType
	dateparts.Day = parts[1]
	dateparts.MonthName = parts[2]
	dateparts.Year = parts[3]
	dateparts.Time = parts[4]
	dateparts.Zone = parts[5]
	dateparts.MonthName = monthMap[dateparts.MonthName]
	mdy := []string{dateparts.Year, dateparts.MonthName, dateparts.Day}
	rewrittenDate := strings.Join(mdy, "-")
	rewrittenDate = rewrittenDate + " " + dateparts.Time + " " + dateparts.Zone
	// fmt.Printf("rewrittenDate = %+v\n", rewrittenDate)
	return rewrittenDate
}

// can't deal with this:
// 22/Nov/2017:18:47:58 +0000
// need to have "12 Feb 2006, 19:17"
func dparse(date string) time.Time {
	var t time.Time
	if strings.Contains(date, "/") {
		date = rewriteDate(date)
	}
	loc, err := time.LoadLocation("UTC")
	time.Local = loc
	t, err = dateparse.ParseLocal(date)
	if err != nil {
		log.Fatal(err)
	}
	return t
}
