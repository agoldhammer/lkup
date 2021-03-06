/*
lkup processes apache log files in 3 different formats. It can stream the
files from a remote server. IPs are associated with hostnames and geodata
before printing.
*/
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
	"gopkg.in/cheggaaa/pb.v1"
)

// Main wait group for processing pipelines
var wg sync.WaitGroup
var bar *pb.ProgressBar

// Fetching security data requires higher level subscription to ipstack
// so this code does nothing
type security struct {
	IsProxy     bool   `json:"is_proxy,omitempty"`
	ProxyType   string `json:"proxy_type,omitempty"`
	IsCrawler   bool   `json:"is_crawler,omitempty"`
	CrawlerName string `json:"crawler_name,omitempty"`
	CrawlerType string `json:"crawler_type,omitempty"`
	IsTor       bool   `json:"is_tor,omitempty"`
	ThreatLevel string `json:"threat_level,omitempty"`
	ThreatTypes string `json:"threat_types,omitempty"`
}

// Geodata : Used by freegeoip lookup
type Geodata struct {
	IP          string   `json:"ip"`
	CountryCode string   `json:"country_code"`
	CountryName string   `json:"country_name"`
	RegionCode  string   `json:"region_code"`
	RegionName  string   `json:"region_name"`
	City        string   `json:"city"`
	Zip         string   `json:"zip_code"`
	TZ          string   `json:"time_zone"`
	Lat         float64  `json:"latitude"`
	Long        float64  `json:"longitude"`
	MetroCode   int32    `json:"metro_code"`
	Hostname    string   `json:"hostname"`
	Sec         security `json:"security"`
}

// HostInfoType : info about host
type HostInfoType struct {
	IP       string
	Hostname string
	Geo      *Geodata
}

// Chnls : Channels used by main processing pipeline
type Chnls []chan *HostInfoType

// Stringer for Geodata
func (g *Geodata) String() string {
	a := fmt.Sprintf("*%v %v %v %v\n", g.CountryName, g.RegionName, g.City, g.Zip)
	b := fmt.Sprintf("*%v (Lat/Long %v %v) Metro: %v\n", g.TZ, g.Lat, g.Long, g.MetroCode)
	return a + b
}

// Print : Colorized printing for HostInfo
func (hostinfo *HostInfoType) Print() {
	cy := color.New(color.FgCyan)
	yellow := color.New(color.FgYellow)
	red := color.New(color.FgRed)
	//cy.Printf("*Hostname: %v\n", hostinfo.Hostname)
	cy.Printf("*Hostname: %v\n", hostinfo.Geo.Hostname)
	yellow.Printf("*Country Code: %v\n", hostinfo.Geo.CountryCode)
	// fmt.Printf("Geo = %+v\n", hostinfo.Geo)
	cy.Printf("%v", hostinfo.Geo)
	sec := hostinfo.Geo.Sec
	if sec.IsProxy || sec.IsCrawler || sec.IsTor {
		red.Printf("Security: %v\n", sec)
	} else {
		cy.Println("No security flags")
	}
}

// main function reads command line, calls log parser to set things up
// The flow is as follows
// The command line flags determine whether input is taken from stdin or from a file\
// parseLog is then called to return a map from ip->LogEntry, where
// a LogEntry consists of 3 parts: ip of a log record, time, and server action (Text)
// This map is then passed to process.

func main() {
	config := ReadConfig()
	exclude := makeExclude(config)
	version := flag.Bool("v", false, "Print version and exit")
	accFlag := flag.Bool("a", false, "Read log from stdin")
	flag.Parse()
	if *version {
		fmt.Println("lkup version 0.40")
		os.Exit(0)
	}
	var selector string
	if *accFlag {
		selector = "a"
	} else if len(os.Args) == 2 {
		selector = os.Args[1]
	} else {
		fmt.Println("usage: lkup [option] [filename]")
		fmt.Println("-a: read from stdin; -v: print version")
		os.Exit(0)
	}
	rawLogEntries := parseLog(selector, exclude)
	if len(rawLogEntries) == 0 {
		fmt.Println("No log entries to process, exiting")
		os.Exit(1)
	}

	perps, hostdb := process(rawLogEntries)
	PrintSorted(perps, hostdb)
}

// process is where the action takes place
// It creates a pipeline of input channels and output channels connected by a multiplexer
// After fan-in, the output channels feed into the hostdb
// The hostdb is then sorted and printed.
// Progress is tracked by a progress bar

func process(logEntries []*LogEntry) (PerpsType, HostDB) {
	/*
		Store list of logEntries in perps map. For each new IP encountered,
		create a pipeline to lookup hostname and geo information and store
		these in the hostdb map. Print out info for each ip. Close all pipelines
	*/
	done := make(chan interface{})
	perps := make(PerpsType)
	hostdb := make(HostDB)

	newIPs := []string{}

	for _, logEntry := range logEntries {
		isNewIP := perps.addLogEntry(logEntry)
		// lookup hostname and geodata only if not already in database
		if isNewIP {
			newIPs = append(newIPs, logEntry.IP)
		}
	}

	count := len(newIPs)
	bar = pb.StartNew(count)
	outChs, inChs := makePipelines(done, count)
	updateCh := multiplexer(done, outChs)
	hostdb.updateHostDB(done, updateCh)

	for i, ip := range newIPs {
		// bar.Increment()

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
	bar.Finish()
	return perps, hostdb
}

// makeExclude creates exclusion map from omit entry in config file
func makeExclude(config Config) map[string]bool {

	omit := config.Omit
	exclude := make(map[string]bool)
	for _, ip := range strings.Split(omit, " ") {
		exclude[ip] = true
	}
	return exclude
}

// LogEntries : ------------------------
type LogEntries []*LogEntry

// Print : print log entry type
func (les LogEntries) Print() {
	for _, le := range les {
		fmt.Printf("*: %+v\n", *le)
	}
}

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
func PrintSorted(p PerpsType, hdb HostDB) {
	timeIndex := p.makeTimeIndex()
	timeIndex = timeIndex.Sort()
	nprinted := 0
	for _, timeToken := range timeIndex {
		nprinted++
		ip := timeToken.IP
		fmt.Println("\n+++++++++")
		fmt.Println("----> ", ip)
		hdb[ip].Print()
		fmt.Println("....")
		p[ip].Print()
	}
}

// addLogEntry adds logentry to perps db, returns true if IP is new
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

var myClient = &http.Client{Timeout: 1 * time.Second}

// getJSON decodes JSON returned from geoip lookup
func getJSON(url string, target interface{}) error {
	r, err := myClient.Get(url)
	if err == nil {
		defer r.Body.Close()
		json.NewDecoder(r.Body).Decode(target)
	}
	return err
}

// lkupGeoloc obtains geolocation data from freegeoip.net
// The new API comes with a completely new endpoint (api.ipstack.com) and requires
// you to append your API Access Key to the URL as a GET parameter.
// For complete integration instructions, please head over to the API Documentation at https://ipstack.com/documentation. While the new API offers a completely reworked response structure with many additional data points, we also offer the option to receive results in the old freegeoip.net format in JSON or XML.

// To receive your API results in the old freegeoip format, please simply append &legacy=1 to the new API URL.

// JSON Example: http://api.ipstack.com/186.116.207.169?access_key=YOUR_ACCESS_KEY&output=json&legacy=1
// 2511e0d2a311aff3101c232172c9e2cf

func lkupGeoloc(done <-chan interface{},
	inCh <-chan *HostInfoType) chan *HostInfoType {
	outCh := make(chan *HostInfoType)

	go func() {
		defer close(outCh)
		wg.Add(1)
		defer wg.Done()
		geoip := "http://api.ipstack.com/"
		suffix := "?access_key=2511e0d2a311aff3101c232172c9e2cf&output=json&hostname=1"
		for hostinfo := range inCh {
			geo := Geodata{}
			// error will leave default geo, which is OK
			err := getJSON(geoip+hostinfo.IP+suffix, &geo)
			if err != nil {
				// log.Printf("Geoloc: err = %+v\n", err)
				geo.Hostname = "Geoloc timed out"
			}
			hostinfo.Geo = &geo
			bar.Increment()
			select {
			case <-done:
				return
			case outCh <- hostinfo:
			}
		}
	}()

	return outCh
}

// multiplexer combines output from multiple pipelines to send to updateHostDB
func multiplexer(done <-chan interface{}, cs Chnls) chan *HostInfoType {
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
	outCh := lkupGeoloc(done, inCh)
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
