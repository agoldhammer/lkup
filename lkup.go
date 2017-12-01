package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/agoldhammer/lkup/parser"
)

var wg sync.WaitGroup
var mon chan string // monitor channel

func check(e error) {
	if e != nil {
		panic(e)
	}
}

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
	IP       string
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
	pipeInput chan *HostInfoType) {
	ip := le.IP
	info, ok := p[ip]
	if !ok {
		info = new(InfoType)
		// this will look up hostname and geodata
		defer lookup(le, pipeInput)
	}
	info.LogEntries = append(info.LogEntries, le)
	p[ip] = info
}

func (p PerpsType) updatePerps(update chan *HostInfoType) {
	// add hostinfo from channel update, caller shd close when done
	for hinfo := range update {
		// mon <- fmt.Sprintf("updatePerps: hinfo = %+v\n", hinfo)
		ip := hinfo.IP
		info := p[ip]
		info.Hostinfo = hinfo
		p[ip] = info
	}
}

var myClient = &http.Client{Timeout: 3 * time.Second}

func getJson(url string, target interface{}) error {
	r, err := myClient.Get(url)
	check(err)
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(target)
}

func lookup(logEntry *parser.LogEntry,
	pipeInput chan *HostInfoType) {
	hostinfo := new(HostInfoType)
	hostinfo.IP = logEntry.IP
	// mon <- fmt.Sprintf("lookup: hostinfo = %+v\n", hostinfo)
	// mon <- fmt.Sprintf("lookup: hostinfo.IP = %+v\n", hostinfo.IP)
	pipeInput <- hostinfo
}

func lookupAddrWTimeout(answerCh chan string, ip string, timeoutSecs int) {
	ansCh := make(chan string)
	var name string
	var err error
	go func() {
		var name string
		names := make([]string, 3)
		names, err = net.LookupAddr(ip)
		if err == nil {
			name = names[0]
		} else {
			name = "unknown"
		}
		ansCh <- name
		close(ansCh)
	}()
	select {
	case <-time.After(time.Duration(timeoutSecs) * time.Second):
		name = "Timed Out"
	case name = <-ansCh:
	}
	answerCh <- name
	wg.Done()
}

func lkupHost(done <-chan interface{},
	lkupCh <-chan *HostInfoType) <-chan *HostInfoType {
	outCh := make(chan *HostInfoType)
	answerCh := make(chan string)
	go func() {
		for hostinfo := range lkupCh {
			select {
			case <-done:
				return
			default:
				wg.Add(1)
				go lookupAddrWTimeout(answerCh, hostinfo.IP, 1)
				name := <-answerCh
				hostinfo.Hostname = name
				outCh <- hostinfo
			}
		}
	}()
	return outCh
}

func lkupGeoloc(done <-chan interface{},
	inCh <-chan *HostInfoType) <-chan *HostInfoType {
	outCh := make(chan *HostInfoType)
	go func() {
		geoip := "https://freegeoip.net/json/"
		for hostinfo := range inCh {
			select {
			case <-done:
				return
			default:
				wg.Add(1)
				go func(hostinfo *HostInfoType) {
					defer wg.Done()
					geo := Geodata{}
					getJson(geoip+hostinfo.IP, &geo)
					hostinfo.Geo = &geo
					// fmt.Printf("geo: hostinfo = %+v\n", hostinfo)
					outCh <- hostinfo
				}(hostinfo)
			}
		}
	}()
	return outCh
}

func monitor(mon chan string) {
	for msg := range mon {
		fmt.Printf("Processing %v\n", msg)
	}
}

func process(logEntries []*parser.LogEntry) {
	update := make(chan *HostInfoType, 5)
	mon = make(chan string)
	done := make(chan interface{})
	pipeInput := make(chan *HostInfoType)
	go monitor(mon)
	perps := make(PerpsType)
	// this is the receiver routine, which updates the perps db
	go perps.updatePerps(update)
	pipeline := lkupGeoloc(done, lkupHost(done, pipeInput))
	go func() {
		for hostinfo := range pipeline {
			update <- hostinfo
		}
	}()
	// start one go routine for each log entry
	for _, logEntry := range logEntries {
		// lookup hostname and geodata only if not already in database
		// fmt.Println(logEntry)
		mon <- logEntry.IP
		perps.addLogEntry(logEntry, pipeInput)
	}
	close(pipeInput)
	mon <- fmt.Sprintf("Processing %v entries\n", len(perps))
	wg.Wait()
	close(done)
	// close(update)
	perps.Print()
}

func main() {
	accFlag := flag.Bool("a", false, "Process access.log")
	otherFlag := flag.Bool("o", false, "Process others_vhosts_access.log")
	errorFlag := flag.Bool("e", false, "Process error.log")
	var logEntries []*parser.LogEntry
	flag.Parse()
	if *accFlag {
		logEntries = parser.ParseAccessLog()
	} else if *otherFlag {
		logEntries = parser.ParseOtherAccessLog()
	} else if *errorFlag {
		logEntries = parser.ParseErrorLog()
	} else {
		fmt.Println("Error, exiting lkup")
		os.Exit(1)
	}
	process(logEntries)
}
