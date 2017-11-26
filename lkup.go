package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"regexp"
	"time"
)

type Geodata struct {
	IP          string  `json:"ip"`
	CountryCode string  `json:"country_code"`
	CountryName string  `json:"country_name"`
	RegionCode  string  `json:"region_code"`
	RegionName  string  `json:"region_name"`
	City        string  `json:"city"`
	Zip         string  `json:"zip_code"`
	TZ          string  `json:"time_zone"`
	Lat         float64 `json:"latitude"`
	Long        float64 `json:"longitude"`
	MetroCode   int32   `json:"metro_code"`
}

var myClient = &http.Client{Timeout: 10 * time.Second}

func getJson(url string, target interface{}) error {
	r, err := myClient.Get(url)
	check(err)
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(target)
}

func lkupGeoloc(ip string) Geodata {
	geoip := "https://freegeoip.net/json/"
	ip2 := geoip + ip
	geo := Geodata{}
	getJson(ip2, &geo)
	return geo
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func lkupReadFile(fname string) string {
	dat, err := ioutil.ReadFile(fname)
	check(err)
	// fmt.Print(string(dat))
	return string(dat)
}

func lookup(ip string) {
	// fmt.Println(ip)
	names := make([]string, 10)
	names, _ = net.LookupAddr(ip)
	// fmt.Println(ip, names)
	geoloc := lkupGeoloc(ip)
	fmt.Println(ip, names, geoloc)
}

func main() {
	dat := lkupReadFile("error.log")
	// fmt.Print(dat)
	// fmt.Println("regexp")
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

// To split read into lines
// content, err := ioutil.ReadFile(filename)
// if err != nil {
// 	    //Do something
//
// }
// lines := strings.Split(string(content), "\n")
