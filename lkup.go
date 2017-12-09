package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"gopkg.in/cheggaaa/pb.v1"
)

var wg sync.WaitGroup

type Chnls []chan *HostInfoType

// var updateWait sync.WaitGroup
var mon chan string // monitor channel

func check(e error) {
	if e != nil {
		fmt.Println(e)
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

func (g *Geodata) String() string {
	a := fmt.Sprintf("*%v %v %v %v\n", g.CountryName, g.RegionName, g.City, g.Zip)
	b := fmt.Sprintf("*%v (Lat/Long %v %v) Metro: %v\n", g.TZ, g.Lat, g.Long, g.MetroCode)
	return a + b
}

func (hostinfo *HostInfoType) Print() {
	fmt.Printf("*Hostname: %v\n", hostinfo.Hostname)
	fmt.Printf("*Country Code: %v\n", hostinfo.Geo.CountryCode)
	// fmt.Printf("Geo = %+v\n", hostinfo.Geo)
	fmt.Printf("%v", hostinfo.Geo)
}

// LogEntries ------------------------
func (les LogEntries) Print() {
	for _, le := range les {
		fmt.Printf("*: %+v\n", *le)
	}
}

type LogEntries []*LogEntry

// ----------------------------------------------------

// HostDB is a map from IPs to HostInfo structures with IP, name, and geodata
type HostDB map[string]*HostInfoType

// updateHostDB takes HostInfo from its input channel and stores in HostDB
func (hdb HostDB) updateHostDB(done chan interface{}, inCh chan *HostInfoType) {

	wg.Add(1)
	go func() {
		defer wg.Done()
		for hostinfo := range inCh {
			hdb[hostinfo.IP] = hostinfo
			select {
			case <-done:
				return
			default:
				continue
			}
		}
	}()
}

// --------------------------------------------

// PerpsType maps IPs to a slice of LogEntry structs containing
// unparsed log entry strings
type PerpsType map[string]LogEntries

// PrintSorted formats and prints hostinfo and logentries
// For each IP, the latest logentry time is used to determine sort order.
// Logentries are grouped by IP, with IPs ranked by this sort order,
// so latest accessed IP will appear last
// ips to exclude are specified in config file, parsed in main
func PrintSorted(p PerpsType, hdb HostDB, exclude map[string]bool) {
	timeIndex := p.makeTimeIndex()
	timeIndex = timeIndex.Sort()
	for _, timeToken := range timeIndex {
		if !exclude[timeToken.IP] {
			ip := timeToken.IP
			fmt.Println("\n+++++++++")
			fmt.Println("----> ", ip)
			hdb[ip].Print()
			fmt.Println("....")
			p[ip].Print()
		}
	}
}

func (p PerpsType) addLogEntry(le *LogEntry) bool {
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

// -----------------------------

var myClient = &http.Client{Timeout: 3 * time.Second}

func getJSON(url string, target interface{}) error {
	r, err := myClient.Get(url)
	check(err)
	defer r.Body.Close()
	json.NewDecoder(r.Body).Decode(target)
	return err
}

// Pipeline functions
// lkupHost uses reverse DNS to find hostname
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
		name := "DNS fail"
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

// lkupGeoloc obtains geolocation data from freegeoip.net
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
			// error will leave default geo, which is OK
			getJSON(geoip+hostinfo.IP, &geo)
			hostinfo.Geo = &geo
			select {
			case <-done:
				return
			case outCh <- hostinfo:
			}
		}
	}()

	return outCh
}

// monitor is a monitoring channel
// TODO: replace with logger
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

// multiplexer combines output from multiple pipelines to send to updateHostDB
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

//makeLookupPipeline creates a single host data lookup pipeline
func makeLookupPipeline(done <-chan interface{}) (chan *HostInfoType,
	chan *HostInfoType) {
	inCh := make(chan *HostInfoType)
	hostCh := lkupHost(done, inCh)
	outCh := lkupGeoloc(done, hostCh)
	return outCh, inCh
}

// makePipelines creates count host lookup pipelines
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

// process is the toplevel function. It creates one pipeline for each new IP.
// It multiplexes the pipelines into updateHostDB. It also creates the
// monitor channel. Waits until all data has been stored, then prints.
func process(logEntries []*LogEntry) (PerpsType, HostDB) {
	/*
		Store list of logEntries in perps map. For each new IP encountered,
		create a pipeline to lookup hostname and geo information and store
		these in the hostdb map. Print out info for each ip. Close all pipelines
	*/
	done := make(chan interface{})
	mon = monitor(done)
	perps := make(PerpsType)
	hostdb := make(HostDB)
	bar := pb.StartNew(len(logEntries))
	newIPs := []string{}

	for _, logEntry := range logEntries {
		bar.Increment()
		isNewIP := perps.addLogEntry(logEntry)
		// lookup hostname and geodata only if not already in database
		if isNewIP {
			newIPs = append(newIPs, logEntry.IP)
		}
	}

	count := len(newIPs)
	outChs, inChs := makePipelines(done, count)
	updateCh := multiplexer(done, outChs)
	hostdb.updateHostDB(done, updateCh)

	for i, ip := range newIPs {
		hostInfo := new(HostInfoType)
		hostInfo.IP = ip
		inChs[i] <- hostInfo
	}

	for _, inCh := range inChs {
		close(inCh)
	}
	// mon <- fmt.Sprintf("Processing %v entries\n", len(perps))
	wg.Wait()
	close(done)
	return perps, hostdb
}

func main() {
	config := ReadConfig()
	omit := config.Omit
	exclude := make(map[string]bool)
	for _, ip := range strings.Split(omit, " ") {
		exclude[ip] = true
	}
	accFlag := flag.Bool("a", false, "Process access.log")
	otherFlag := flag.Bool("o", false, "Process others_vhosts_access.log")
	errorFlag := flag.Bool("e", false, "Process error.log")
	remoteFlag := flag.Bool("r", false, "Read file from remote server")
	dateFlag := flag.Bool("d", false, "TODO for testing dates")
	flag.Parse()
	if *dateFlag {
		t := dparse("22/Nov/2017:18:47:58 +0000")
		fmt.Printf("%v\n", t)
		os.Exit(0)
	}
	var selector string
	if *accFlag {
		selector = "a"
	} else if *otherFlag {
		selector = "o"
	} else if *errorFlag {
		selector = "e"
	} else {
		fmt.Println("Error, exiting lkup")
		os.Exit(1)
	}
	rawLogEntries := parseLog(selector, *remoteFlag, config.Server)
	perps, hostdb := process(rawLogEntries)
	PrintSorted(perps, hostdb, exclude)
}
