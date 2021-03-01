package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"regexp"
	"time"

	"github.com/BurntSushi/toml"
)

// Config : from lkup config file
type Config struct {
	Server string
	Omit   string
}

// LogEntry : info from log file
type LogEntry struct {
	IP   string
	Time time.Time
	Text string
}

// Logsrc : represents log file
type Logsrc struct {
	fname   string
	file    *os.File
	scanner *bufio.Scanner
}

func makeLogsrc(fname string) *Logsrc {
	logsrc := Logsrc{fname: fname}
	file, err := os.Open(fname)
	if err != nil {
		log.Fatal(err)
	} else {
		logsrc.file = file
	}
	logsrc.scanner = bufio.NewScanner(file)
	return &logsrc
}

func makeLogsrcFromStdin() *Logsrc {
	logsrc := Logsrc{fname: "stdin"}
	logsrc.file = os.Stdin
	logsrc.scanner = bufio.NewScanner(os.Stdin)
	return &logsrc
}

// LogParser : parses logs from files or stdin
type LogParser struct {
	fname           string
	logsrc          *Logsrc
	parseExpression string
	order           [3]uint8
}

// parseLog parses logentry data according to specified regexp
func parseLog(which, server string, remoteFlag bool,
	exclude map[string]bool) []*LogEntry {
	// which should be one of e, a, or o to select appropriate
	//  log file

	var logparser LogParser
	fmt.Printf("parseLog called: %s\n", which)
	switch which {

	case "a":
		logparser = LogParser{fname: "stdin",
			logsrc:          makeLogsrcFromStdin(),
			parseExpression: `(\S+).+\[(.+)] "([^"]+)"`,
			order:           [3]uint8{2, 1, 3}}

	default:
		// log.Fatal("bad option to parser")
		logparser = LogParser{fname: which,
			logsrc:          makeLogsrc(which),
			parseExpression: `(\S+).+\[(.+)] "([^"]+)"`,
			order:           [3]uint8{2, 1, 3}}
	}

	fmt.Printf("Reading local file %s\n", logparser.fname)
	// locallog := LocalLog{logparser.fname}
	// lines = locallog.ReadLines()
	npart := logparser.order
	logEntries := []*LogEntry{}
	scanner := logparser.logsrc.scanner
	re := regexp.MustCompile(logparser.parseExpression)
	for scanner.Scan() {
		line := scanner.Text()
		// fmt.Println(line)
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
