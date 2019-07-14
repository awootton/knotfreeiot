package knotfree

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// DoStartEventCollectorReporting - set to run the reporter
var DoStartEventCollectorReporting = false

//var delayBetweenReports = 20 * time.Second
var delayBetweenReports = 10 * time.Second

// Reporting - besides the StringEventAccumulator other reports will be needed.
type Reporting interface {
	report(float32) []string
}

var reporters = make([]Reporting, 0, 25)

func init() {
	//reporters = make([]Reporting, 0, 25)
	//testLogThing = NewStringEventAccumulator(16)
	if DoStartEventCollectorReporting {
		go startRunningReports()
	}
}

// StringEventAccumulator a
type StringEventAccumulator struct {
	sync.RWMutex
	countMap map[string]float32
	strlen   int  // for trimming keys
	quiet    bool // don't println incoming msgs
}

// NewStringEventAccumulator s
func NewStringEventAccumulator(maxstrlen int) *StringEventAccumulator {
	cm := StringEventAccumulator{}
	cm.countMap = make(map[string]float32)
	cm.strlen = maxstrlen
	AddReporter(&cm)
	return &cm
}

// AddReporter appends r to the global list of reporters.
func AddReporter(r Reporting) {
	reporters = append(reporters, r)
	//fmt.Println("AddReporter count=" + strconv.Itoa(len(reporters)))
}

// Collect - Users will call this when strings happen and we'll count the rate.
// eg. atwCm.Collect("This is a serious bug")
func (collector *StringEventAccumulator) Collect(str string) {
	if collector.quiet == false {
		fmt.Println(str)
	}
	if len(str) == 0 {
		str = "collected_empty_str"
	}
	if len(str) > collector.strlen {
		str = str[0:collector.strlen]
	}
	collector.Lock()
	v, ok := collector.countMap[str]
	if ok == false {
		collector.countMap[str] = 1
	} else {
		collector.countMap[str] = v + 1
	}
	collector.Unlock()
}

// Sum - add the amount to the item instead of adding 1 like above
func (collector *StringEventAccumulator) Sum(str string, amt int) {
	if collector.quiet == false {
		fmt.Println(str)
	}
	if len(str) == 0 {
		str = "collected_empty_str"
	}
	if len(str) > collector.strlen {
		str = str[0:collector.strlen]
	}
	collector.Lock()
	v, ok := collector.countMap[str]
	if ok == false {
		collector.countMap[str] = float32(amt)
	} else {
		collector.countMap[str] = v + float32(amt)
	}
	collector.Unlock()
}

// seconds is in seconds
func (collector *StringEventAccumulator) report(seconds float32) []string {
	lll := make([]string, 0)
	var keys = make([]string, 0, len(collector.countMap))
	collector.Lock()
	for k := range collector.countMap {
		keys = append(keys, k)
	}
	collector.Unlock()
	sort.Strings(keys)
	for _, k := range keys {
		collector.Lock()
		counts := collector.countMap[k]
		collector.countMap[k] = counts * 0.5
		if counts/(2*seconds) < 0.01 {
			delete(collector.countMap, k)
		}
		collector.Unlock()
		val := "        " + strconv.FormatFloat(float64(counts/(2*seconds)), 'f', 2, 32)
		val = val[len(val)-8 : len(val)]
		lll = append(lll, k+":"+val+"/s")
	}
	return lll
}

// want count per sec
var reportTicker time.Ticker
var reportCount = 0

func startRunningReports() {
	reportTicker := time.NewTicker(delayBetweenReports)
	previousTime := time.Now()

	for t := range reportTicker.C {
		var sb strings.Builder
		width := 0
		elapsed := t.Sub(previousTime)
		for _, reporter := range reporters {
			sssarr := reporter.report(float32(elapsed / time.Second))
			for _, str := range sssarr {
				sb.WriteString(str)
				sb.WriteString("    ")
				width += len(str) + 1
				if width > 120 {
					sb.WriteString("\n")
					width = 0
				}
			}
		}
		fmt.Println("Report#" + strconv.Itoa(reportCount))
		fmt.Println(sb.String())
		fmt.Println("")
		previousTime = t
	}
}

//
//var testLogThing *StringEventAccumulator
