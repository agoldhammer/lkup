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

type InfoType struct {
	Host       string
	Geo        Geodata
	LogEntries []parser.LogEntry
}

func (info InfoType) Print() {
	fmt.Printf("%s\n", info.Host)
	fmt.Printf("%+v\n", info.Geo)
	for _, le := range info.LogEntries {
		fmt.Printf("%+v\n", le)
	}
}

type PerpsType map[string]InfoType

func (p PerpsType) Print() {
	fmt.Println("===========")
	for ip, _ := range p {
		fmt.Println("----> ", ip)
		p[ip].Print()
	}
}

func (pp *PerpsType) addLogEntry(le parser.LogEntry) {
	ip := le.IP
	p := *pp
	info, ok := p[ip]
	if !ok {
		info = InfoType{}
	}
	info.LogEntries = append(info.LogEntries, le)
	p[ip] = info
}

// type Geo2 map[string]interface{}

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
	// geo2 := Geo2{}
	// getJson(ip2, &geo2)
	// return geo2
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
	fmt.Println(ip, names, geoloc.CountryName)
}

func main() {
	perps := make(PerpsType)
	pptr := &perps
	logEntries := parser.ParseAccessLog()
	for _, logEntry := range logEntries {
		// lookup hostname and geodata only if not already in database
		if _, ok := perps[logEntry.IP]; !ok {
			go lookup(logEntry)
		}
		pptr.addLogEntry(logEntry)
	}
	time.Sleep(time.Second * 5)
	perps.Print()
}

// To split read into lines
// content, err := ioutil.ReadFile(filename)
// if err != nil {
// 	    //Do something
//
// }
// lines := strings.Split(string(content), "\n")
