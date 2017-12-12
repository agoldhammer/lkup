package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Server string
	Omit   string
}

type LogEntry struct {
	IP   string
	Time time.Time
	Text string
}

// LogReader reads a log and returns content as slice of lines
type LogReader interface {
	ReadLines() []string
}

type LocalLog struct {
	fname string
}

type RemoteLog struct {
	server string
	fname  string
}

// LocalLog.ReadLines satisfies LogReader interface for local logs.
func (l LocalLog) ReadLines() []string {
	content, err := ioutil.ReadFile(l.fname)
	if err != nil {
		log.Fatal(err)
	}
	lines := strings.Split(string(content), "\n")
	return lines
}

// RemoteLog.Readlines satisfies LogReader interface for remote logs
func (l RemoteLog) ReadLines() []string {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get("http://" + l.server + l.fname)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	lines := strings.Split(string(content), "\n")
	return lines
}

type LogParser struct {
	fname           string
	parseExpression string
	order           [3]uint8
}

// parseLog parses logentry data according to specified regexp
func parseLog(which, server string, remoteFlag bool,
	exclude map[string]bool) []*LogEntry {
	// which should be one of e, a, or o to select appropriate
	//  log file

	var logparser LogParser
	switch which {
	case "e":
		logparser = LogParser{fname: "error.log",
			parseExpression: `\[(.+)] \[core:.+] \[.+] .*\[client (\S+):\d+] (.+)`,
			order:           [3]uint8{1, 2, 3}}
	case "a":
		logparser = LogParser{fname: "access.log",
			parseExpression: `(\S+).+\[(.+)] "([^"]+)"`,
			order:           [3]uint8{2, 1, 3}}
	case "o":
		logparser = LogParser{fname: "other_vhosts_access.log",
			parseExpression: `\S+\s(\S+).+\[(.+)][^"]+"([^"]+)"`,
			order:           [3]uint8{2, 1, 3}}
	default:
		log.Fatal("bad option to parser")
	}

	var lines []string

	if remoteFlag {
		lines = RemoteLog{server, logparser.fname}.ReadLines()
	} else {
		lines = LocalLog{logparser.fname}.ReadLines()
	}
	npart := logparser.order
	logEntries := []*LogEntry{}
	for _, line := range lines {
		re := regexp.MustCompile(logparser.parseExpression)
		result := re.FindAllStringSubmatch(line, -1)
		if result != nil {
			parts := result[0]
			if len(parts) != 4 {
				log.Fatal("Parse error, exiting")
			}
			logEntry := new(LogEntry)
			logEntry.IP = parts[npart[1]]
			if !exclude[logEntry.IP] {
				logEntry.Time = dparse(parts[npart[0]])
				logEntry.Text = parts[npart[2]]
				logEntries = append(logEntries, logEntry)
			}

		}
	}
	return logEntries
}

// ReadConfig reads parameters from lkup.config file
func ReadConfig() Config {
	var configfile = os.Getenv("HOME") + "/.lkup/lkup.config"
	_, err := os.Stat(configfile)
	if err != nil {
		log.Fatal(err)
	}

	var config Config
	if _, err := toml.DecodeFile(configfile, &config); err != nil {
		log.Fatal(err)
	}
	return config
}
