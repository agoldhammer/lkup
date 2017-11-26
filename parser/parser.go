package parser

import (
	"fmt"
	"io/ioutil"
	"regexp"
	"strings"
)

// To split read into lines
func ReadLines(filename string) []string {
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		//Do something

	}
	lines := strings.Split(string(content), "\n")
	return lines
}

func ParseErrorLog() {
	errexp := `\[(.+)] \[core:info] \[.+] \[client (\S+):\S+](.+)`
	lines := ReadLines("error.log")
	for _, line := range lines {
		re := regexp.MustCompile(errexp)
		// re := regexp.MustCompile(`client (\S+):`)
		result := re.FindAllStringSubmatch(line, -1)
		if result != nil {
			for _, piece := range result[0] {
				fmt.Printf("piece: %v\n", piece)
			}
		}
	}
}
