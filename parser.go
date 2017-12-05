package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Server string
}

type LogEntry struct {
	IP   string
	Time string
	Text string
}

// ReadLines : To split file into lines
func ReadLines(filename string) []string {
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		check(err)
	}
	lines := strings.Split(string(content), "\n")
	return lines
}

// ReadRemoteFile : To split remote file into lines
func ReadRemoteFile(server, filename string) []string {
	resp, err := http.Get("http://" + server + filename)
	if err != nil {
		check(err)
	}
	defer resp.Body.Close()
	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		check(err)
	}
	lines := strings.Split(string(content), "\n")
	return lines
}

func parseLog(which string, remoteFlag bool, svr string) []*LogEntry {
	// which should be one of e, a, or o to select appropriate
	//  log file

	const (
		errexp    = `\[(.+)] \[core:info] \[.+] \[client (\S+):\S+](.+)`
		accessexp = `(\S+).+\[(.+)] "([^"]+)"`
		otherexp  = `\S+\s(\S+).+\[(.+)][^"]+"([^"]+)"`
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
	fmt.Printf("remoteFlag = %+v\n", remoteFlag)
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
			logEntry := new(LogEntry)
			// fmt.Printf("parts = %+v\n", parts)
			logEntry.Time = parts[npart[0]]
			logEntry.IP = parts[npart[1]]
			logEntry.Text = parts[npart[2]]
			logEntries = append(logEntries, logEntry)

		}
	}
	return logEntries
}

// Reads info from config file
func ReadConfig() Config {
	var configfile = "lkup.config"
	_, err := os.Stat(configfile)
	if err != nil {
		check(err)
	}

	var config Config
	if _, err := toml.DecodeFile(configfile, &config); err != nil {
		check(err)
	}
	log.Print(config.Server)
	return config
}

func main() {
	config := ReadConfig()
	accFlag := flag.Bool("a", false, "Process access.log")
	otherFlag := flag.Bool("o", false, "Process others_vhosts_access.log")
	errorFlag := flag.Bool("e", false, "Process error.log")
	remoteFlag := flag.Bool("r", false, "Read file from remote server")
	flag.Parse()
	var selector string
	if *accFlag {
		selector = "a"
	} else if *otherFlag {
		selector = "o"
	} else if *errorFlag {
		selector = "e"
	} else {
		fmt.Println("Error, exiting lkup")
		os.Exit(1)
	}
	logEntries := parseLog(selector, *remoteFlag, config.Server)
	process(logEntries)
}
