package main

import (
	"encoding/json"
	"fmt"
	"github.com/agoldhammer/lkup/parser"
	"net"
	"net/http"
	"sync"
	"time"
)

var wg sync.WaitGroup

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

type HostInfoType struct {
	Hostname string
	Geo      Geodata
}

type InfoType struct {
	Hostinfo   HostInfoType
	LogEntries []parser.LogEntry
}

func (info InfoType) Print() {
	fmt.Printf("**HostInfo**%+v\n", info.Hostinfo)
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

func (p PerpsType) addLogEntry(le parser.LogEntry) {
	ip := le.IP
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

func lookup(logEntry parser.LogEntry, update chan HostInfoType) {
	// fmt.Println(ip)
	var name string
	defer wg.Done()
	ip := logEntry.IP
	names := make([]string, 10)
	names, _ = net.LookupAddr(ip)
	// fmt.Println(ip, names)
	geoloc := lkupGeoloc(ip)
	if names != nil {
		name = names[0]
	} else {
		name = "unknown"
	}
	hostinfo := HostInfoType{name, geoloc}
	// fmt.Printf("logEntry = %+v\n", logEntry)
	// fmt.Println(ip, names, geoloc.CountryName)
	update <- hostinfo
	// fmt.Printf("hostinfo = %+v\n", hostinfo)
}

func (p PerpsType) updatePerps(update chan HostInfoType) {
	for hinfo := range update {
		ip := hinfo.Geo.IP
		info := p[ip]
		info.Hostinfo = hinfo
		p[ip] = info
	}
}

func main() {
	update := make(chan HostInfoType, 10)
	perps := make(PerpsType)
	// pptr := &perps
	logEntries := parser.ParseAccessLog()
	go perps.updatePerps(update)
	for _, logEntry := range logEntries {
		// lookup hostname and geodata only if not already in database
		if _, ok := perps[logEntry.IP]; !ok {
			wg.Add(1)
			go lookup(logEntry, update)
		}
		perps.addLogEntry(logEntry)
	}
	fmt.Printf("Processing %v entries\n", len(perps))
	wg.Wait()
	close(update)
	// time.Sleep(time.Second * 5)
	perps.Print()
}
