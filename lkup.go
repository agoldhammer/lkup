package main

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/agoldhammer/lkup/parser"
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
	Geo      *Geodata
}

func (hostinfo *HostInfoType) Print() {
	fmt.Printf("Hostname: %v\n", hostinfo.Hostname)
	fmt.Printf("Country Code: %v\n", hostinfo.Geo.CountryCode)
	fmt.Printf("Geo = %+v\n", hostinfo.Geo)
}

type InfoType struct {
	Hostinfo   *HostInfoType
	LogEntries []*parser.LogEntry
}

func (info *InfoType) Print() {
	// fmt.Printf("**HostInfo**%+v\n", info.Hostinfo)
	info.Hostinfo.Print()
	for n, le := range info.LogEntries {
		fmt.Printf("Log entry %d: %+v\n", n, le)
	}
}

type PerpsType map[string]*InfoType

func (p PerpsType) Print() {
	for ip, _ := range p {
		fmt.Println("\n+++++++++")
		fmt.Println("----> ", ip)
		p[ip].Print()
	}
}

func (p PerpsType) addLogEntry(le *parser.LogEntry,
	update chan *HostInfoType) {
	ip := le.IP
	info, ok := p[ip]
	if !ok {
		// make sure not to start lookup before info has been added
		defer lookup(le, update)
		info = new(InfoType)
	}
	info.LogEntries = append(info.LogEntries, le)
	p[ip] = info
}

func (p PerpsType) updatePerps(update chan *HostInfoType) {
	// add hostinfo from channel update, caller shd close when done
	for hinfo := range update {
		ip := hinfo.Geo.IP
		info := p[ip]
		info.Hostinfo = hinfo
		p[ip] = info
	}
}

// type Geo2 map[string]interface{}

var myClient = &http.Client{Timeout: 10 * time.Second}

func getJson(url string, target interface{}) error {
	r, err := myClient.Get(url)
	check(err)
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(target)
}

func lkupGeoloc(ip string) *Geodata {
	geoip := "https://freegeoip.net/json/"
	ip2 := geoip + ip
	geo := Geodata{}
	getJson(ip2, &geo)
	return &geo
	// geo2 := Geo2{}
	// getJson(ip2, &geo2)
	// return geo2
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func lookup(logEntry *parser.LogEntry, update chan *HostInfoType) {
	wg.Add(1)
	go doAsyncLookups(logEntry.IP, update)
}

func doAsyncLookups(ip string, update chan *HostInfoType) {
	// fmt.Println(ip)
	var name string
	defer wg.Done()
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
	update <- &hostinfo
	// fmt.Printf("hostinfo = %+v\n", hostinfo)
}

func monitor(mon chan string) {
	for msg := range mon {
		fmt.Printf("Processing %v\n", msg)
	}
	fmt.Println("All processed")
}

func main() {
	update := make(chan *HostInfoType, 5)
	mon := make(chan string)
	go monitor(mon)
	perps := make(PerpsType)
	logEntries := parser.ParseAccessLog()
	// this is the receiver routine, which updates the perps db
	go perps.updatePerps(update)
	// start one go routine for each log entry
	for _, logEntry := range logEntries {
		// lookup hostname and geodata only if not already in database
		// fmt.Println(logEntry)
		mon <- logEntry.IP
		perps.addLogEntry(logEntry, update)
	}
	msg := fmt.Sprintf("Processing %v entries\n", len(perps))
	mon <- msg
	wg.Wait()
	close(mon)
	close(update)
	perps.Print()
}
