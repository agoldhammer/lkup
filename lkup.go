package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"gopkg.in/cheggaaa/pb.v1"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/agoldhammer/lkup/parser"
)

var wg sync.WaitGroup

type Chnls []chan *HostInfoType

// var updateWait sync.WaitGroup
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

func (les LogEntries) Print() {
	for n, le := range les {
		fmt.Printf("Log entry %d: %+v\n", n, le)
	}
}

type LogEntries []*parser.LogEntry

type PerpsType map[string]LogEntries

type HostDB map[string]*HostInfoType

func (p PerpsType) Print(hdb *HostDB) {
	for ip, _ := range p {
		fmt.Println("\n+++++++++")
		fmt.Println("----> ", ip)
		(*hdb)[ip].Print()
		p[ip].Print()
	}
}

func (p PerpsType) addLogEntry(le *parser.LogEntry) bool {
	isNewIP := false
	ip := le.IP
	les, ok := p[ip]
	if !ok {
		les = LogEntries{}
		isNewIP = true
	}
	p[ip] = append(les, le)
	return isNewIP
}

var myClient = &http.Client{Timeout: 3 * time.Second}

func getJson(url string, target interface{}) error {
	r, err := myClient.Get(url)
	check(err)
	defer r.Body.Close()
	json.NewDecoder(r.Body).Decode(target)
	return err
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
}

func lkupHost(done <-chan interface{},
	inCh <-chan *HostInfoType) chan *HostInfoType {
	outCh := make(chan *HostInfoType)

	revdns := func(ip string, dnsCh chan string) {
		var name string
		names, err := net.LookupAddr(ip)
		if err == nil {
			name = names[0]
		} else {
			name = "unknown"
		}
		dnsCh <- name
		close(dnsCh)
	}

	go func() {
		defer close(outCh)
		wg.Add(1)
		defer wg.Done()
		var name string
		for hostinfo := range inCh {
			dnsCh := make(chan string)
			go revdns(hostinfo.IP, dnsCh)
			select {
			case name = <-dnsCh:
			case <-time.After(1 * time.Second):
				name = "Timed Out!"
			}
			hostinfo.Hostname = name
			select {
			case <-done:
				return
			case outCh <- hostinfo:
			}
		}
	}()

	return outCh
}

func lkupGeoloc(done <-chan interface{},
	inCh <-chan *HostInfoType) chan *HostInfoType {
	outCh := make(chan *HostInfoType)

	go func() {
		defer close(outCh)
		wg.Add(1)
		defer wg.Done()
		geoip := "https://freegeoip.net/json/"
		for hostinfo := range inCh {
			geo := Geodata{}
			err := getJson(geoip+hostinfo.IP, &geo)
			check(err)
			hostinfo.Geo = &geo
			// mon <- fmt.Sprintf("hostinfo = %+v\n", hostinfo)
			select {
			case <-done:
				return
			case outCh <- hostinfo:
			}
		}
	}()

	return outCh
}

func monitor(done chan interface{}) chan string {
	mon := make(chan string)

	go func() {
		defer close(mon)
		for msg := range mon {
			fmt.Printf("Monitor: %v\n", msg)
			select {
			case <-done:
				return
			default:
				continue
			}
		}
	}()

	return mon
}

func multiplexer(done <-chan interface{},
	cs Chnls) chan *HostInfoType {
	// see https://blog.golang.org/pipelines
	var wg2 sync.WaitGroup
	out := make(chan *HostInfoType)

	output := func(c chan *HostInfoType) {
		for hostinfo := range c {
			select {
			case out <- hostinfo:
			case <-done:
			}
		}
		wg2.Done()
	}

	wg2.Add(len(cs))
	for _, c := range cs {
		go output(c)
	}

	go func() {
		wg2.Wait()
		close(out)
	}()

	return out
}

func (hdb HostDB) updateHostDB(done chan interface{}, inCh chan *HostInfoType) {

	go func() {
		for hostinfo := range inCh {
			hdb[hostinfo.IP] = hostinfo
			// fmt.Printf("update: %v\n", hostinfo)
			select {
			case <-done:
				return
			default:
				continue
			}
		}
	}()
}

func makeLookupPipeline(done <-chan interface{}) (chan *HostInfoType,
	chan *HostInfoType) {
	inCh := make(chan *HostInfoType)
	hostCh := lkupHost(done, inCh)
	outCh := lkupGeoloc(done, hostCh)
	return outCh, inCh
}

func makePipelines(done <-chan interface{}, count int) (Chnls, Chnls) {
	outChs := make(Chnls, count)
	inChs := make(Chnls, count)
	for i := 0; i < count; i++ {
		outCh, inCh := makeLookupPipeline(done)
		outChs[i] = outCh
		inChs[i] = inCh
	}
	return outChs, inChs
}

func process(logEntries []*parser.LogEntry) {
	done := make(chan interface{})
	mon = monitor(done)
	perps := make(PerpsType)
	hostdb := make(HostDB)
	// pipeoutCh, pipeinCh := makeLookupPipeline(done)
	count := len(logEntries)
	outChs, inChs := makePipelines(done, count)
	updateCh := multiplexer(done, outChs)
	hostdb.updateHostDB(done, updateCh)
	bar := pb.StartNew(count)
	for i, logEntry := range logEntries {
		// mon <- logEntry.IP
		bar.Increment()
		isNewIP := perps.addLogEntry(logEntry)
		// lookup hostname and geodata only if not already in database
		if isNewIP {
			// mon <- logEntry.IP
			hostInfo := new(HostInfoType)
			hostInfo.IP = logEntry.IP
			inChs[i] <- hostInfo
		}
	}
	for _, inCh := range inChs {
		close(inCh)
	}
	// mon <- fmt.Sprintf("Processing %v entries\n", len(perps))
	wg.Wait()
	perps.Print(&hostdb)
	close(done)
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
