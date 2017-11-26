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
	lines := ReadLines("error.log")
	for _, line := range lines {
		re := regexp.MustCompile(`client (\S+):`)
		result := re.FindAllStringSubmatch(line, -1)
		if result != nil {
			fmt.Printf("%v", result)
		}
	}
}
