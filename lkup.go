package main

import (
	"fmt"
	"io/ioutil"
	"net"
	"regexp"
	"time"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func lkupReadFile(fname string) string {
	dat, err := ioutil.ReadFile(fname)
	check(err)
	fmt.Print(string(dat))
	return string(dat)
}

func lookup(ip string) {
	fmt.Println(ip)
	names := make([]string, 10)
	names, _ = net.LookupAddr(ip)
	fmt.Println(ip, names)
}

func main() {
	dat := lkupReadFile("error.log")
	fmt.Print(dat)
	fmt.Println("regexp")
	re := regexp.MustCompile(`client (\S+):`)
	result := re.FindAllStringSubmatch(dat, -1)
	for _, res := range result {
		ip := res[1]
		go lookup(ip)
	}
	// var err error
	//	ips := []string{"172.75.31.11", "72.92.101.34", "75.92.101.34"}
	//	for _, ip := range ips {
	//		go lookup(ip)
	//	}
	time.Sleep(time.Second * 5)
}
