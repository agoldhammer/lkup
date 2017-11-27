package main

import (
	"encoding/json"
	"fmt"
	"github.com/agoldhammer/lkup/parser"
	"net"
	"net/http"
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

type Geo2 map[string]interface{}

var myClient = &http.Client{Timeout: 10 * time.Second}

func getJson(url string, target interface{}) error {
	r, err := myClient.Get(url)
	check(err)
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(target)
}

func lkupGeoloc(ip string) Geo2 {
	geoip := "https://freegeoip.net/json/"
	ip2 := geoip + ip
	//	geo := Geodata{}
	geo2 := Geo2{}
	getJson(ip2, &geo2)
	return geo2
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func lookup(logEntry parser.LogEntry) {
	// fmt.Println(ip)
	ip := logEntry.IP
	names := make([]string, 10)
	names, _ = net.LookupAddr(ip)
	// fmt.Println(ip, names)
	geoloc := lkupGeoloc(ip)
	// fmt.Printf("logEntry = %+v\n", logEntry)
	fmt.Println(ip, names, geoloc["country_name"])
}

func main() {
	logEntries := parser.ParseErrorLog()
	for _, logEntry := range logEntries {
		// fmt.Printf("logEntry.IP = %+v\n", logEntry.IP)
		go lookup(logEntry)
	}
	time.Sleep(time.Second * 5)
}

// To split read into lines
// content, err := ioutil.ReadFile(filename)
// if err != nil {
// 	    //Do something
//
// }
// lines := strings.Split(string(content), "\n")
