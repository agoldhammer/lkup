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

// ReadLines : To split file into lines
func ReadLines(filename string) []string {
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatal(err)
	}
	lines := strings.Split(string(content), "\n")
	return lines
}

// ReadRemoteFile : To split remote file into lines
func ReadRemoteFile(server, filename string) []string {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get("http://" + server + filename)
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

// parseLog parses logentry data according to specified regexp
func parseLog(which string, remoteFlag bool, svr string) []*LogEntry {
	// which should be one of e, a, or o to select appropriate
	//  log file

	const (
		// the following works with error as well as info level
		errexp    = `\[(.+)] \[core:.+] \[.+] .*\[client (\S+):\d+] (.+)`
		accessexp = `(\S+).+\[(.+)] "([^"]+)"`
		// the following does not work with error but does with info
		// errexp    = `\[(.+)] \[core:.+] \[.+] \[client (\S+):\S+](.+)`
		otherexp = `\S+\s(\S+).+\[(.+)][^"]+"([^"]+)"`
	)

	parseexp := [3]string{errexp, accessexp, otherexp}

	errord := [3]uint8{1, 2, 3}
	accessord := [3]uint8{2, 1, 3}
	otherord := [3]uint8{2, 1, 3}
	order := [3][3]uint8{errord, accessord, otherord}

	selectmap := map[string]int{"e": 0, "a": 1, "o": 2}
	objmap := [3]string{"error.log", "access.log", "other_vhosts_access.log"}
	selector := selectmap[which]
	fname := objmap[selector]
	rexp := parseexp[selector]
	npart := order[selector]

	var lines []string
	if remoteFlag {
		lines = ReadRemoteFile(svr, fname)
	} else {
		lines = ReadLines(fname)
	}

	logEntries := []*LogEntry{}
	for _, line := range lines {
		re := regexp.MustCompile(rexp)
		result := re.FindAllStringSubmatch(line, -1)
		if result != nil {
			parts := result[0]
			if len(parts) != 4 {
				log.Fatal("Parse error, exiting")
			}
			logEntry := new(LogEntry)
			// fmt.Printf("parts = %+v\n", parts)
			// logEntry.Time = parts[npart[0]]
			logEntry.Time = dparse(parts[npart[0]])
			logEntry.IP = parts[npart[1]]
			logEntry.Text = parts[npart[2]]
			logEntries = append(logEntries, logEntry)

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
