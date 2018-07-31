package main

import (
	"sort"
	"time"
)

// A time token contains the IP and (latest) time of logentry associated
// with that IP.
type timeTokenT struct {
	IP string
	t  time.Time
}

// timeIndexT type is a slice of timeTokens
type timeIndexT []*timeTokenT

// makeTimeIndex looks for the latest logentry time for each IP in perps
// It returns a timeIndex, which is a slice of timeTokens.
func (p PerpsType) makeTimeIndex() timeIndexT {
	timeIndex := timeIndexT{}
	for ip := range p {
		logentries := p[ip]
		n := len(logentries)
		latest := logentries[n-1]
		token := timeTokenT{latest.IP, latest.Time}
		timeIndex = append(timeIndex, &token)
	}
	return timeIndex
}

// Sort sorts a timeIndex from earliest to latest
func (timeIndex timeIndexT) Sort() timeIndexT {
	sort.Slice(timeIndex, func(i, j int) bool {
		return timeIndex[i].t.Before(timeIndex[j].t)
	})
	return timeIndex
}
