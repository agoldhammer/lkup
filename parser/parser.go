package parser

import (
	// "fmt"
	"io/ioutil"
	"regexp"
	"strings"
)

type LogEntry struct {
	IP   string
	Time string
	Text string
}

// To split read into lines
func ReadLines(filename string) []string {
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		//Do something

	}
	lines := strings.Split(string(content), "\n")
	return lines
}

func ParseErrorLog() []LogEntry {
	errexp := `\[(.+)] \[core:info] \[.+] \[client (\S+):\S+](.+)`
	lines := ReadLines("error.log")
	logEntries := []LogEntry{}[:]
	for _, line := range lines {
		re := regexp.MustCompile(errexp)
		// re := regexp.MustCompile(`client (\S+):`)
		result := re.FindAllStringSubmatch(line, -1)
		if result != nil {
			parts := result[0]
			logEntry := LogEntry{}
			// fmt.Printf("parts = %+v\n", parts)
			logEntry.Time = parts[1]
			logEntry.IP = parts[2]
			logEntry.Text = parts[3]
			logEntries = append(logEntries, logEntry)
			// fmt.Println(logEntry.ip, logEntry)
		}
	}
	// fmt.Println(logEntries)
	return logEntries
}
