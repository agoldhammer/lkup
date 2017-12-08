package main

import (
	"sort"
	"time"
)

type timeTokenT struct {
	IP string
	t  time.Time
}

type timeIndexT []*timeTokenT

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

func (timeIndex timeIndexT) Sort() timeIndexT {
	sort.Slice(timeIndex, func(i, j int) bool {
		return timeIndex[i].t.Before(timeIndex[j].t)
	})
	return timeIndex
}
