// Copyright 2019 Alan Tracey Wootton

package types

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// DoStartEventCollectorReporting - set to run the reporter
var DoStartEventCollectorReporting = true

//var delayBetweenReports = 6 per minute
var delayBetweenReports = 10 * time.Second

// Reporting - besides the StringEventAccumulator other reports will be needed.
// they can implement this interface. See
type Reporting interface {
	report(float32) []string
}

var reporters = make([]Reporting, 0, 25)

// func init() {
// 	if DoStartEventCollectorReporting {
// 		go startRunningReports()
// 	}
// }

// GenericEventAccumulator is
type GenericEventAccumulator struct {
	reporter func(float32) []string
}

// StringEventAccumulator a
type StringEventAccumulator struct {
	sync.RWMutex
	countMap map[string]float32
	onceSet  map[string]bool
	strlen   int  // for trimming keys
	quiet    bool // don't println incoming msgs
}

// SetQuiet is
func (collector *StringEventAccumulator) SetQuiet(q bool) {
	collector.quiet = q
}

// NewStringEventAccumulator is
func NewStringEventAccumulator(maxstrlen int) *StringEventAccumulator {
	cm := StringEventAccumulator{}
	cm.countMap = make(map[string]float32)
	cm.onceSet = make(map[string]bool)
	cm.strlen = maxstrlen
	cm.quiet = false
	addReporter(&cm)
	return &cm
}

// NewGenericEventAccumulator is
func NewGenericEventAccumulator(reporter func(float32) []string) *GenericEventAccumulator {
	cm := GenericEventAccumulator{}
	cm.reporter = reporter
	addReporter(&cm)
	return &cm
}

func (me *GenericEventAccumulator) report(seconds float32) []string {
	return me.reporter(seconds)
}

// addReporter appends r to the global list of reporters.
func addReporter(r Reporting) {
	reporters = append(reporters, r)
}

// Collect - Users will call this when strings happen and we'll count the rate.
// eg. atwCm.Collect("This is a serious bug")
func (collector *StringEventAccumulator) Collect(str string) {
	if collector.quiet == false {
		fmt.Println(str) // leave this. It's the exception to the rule.
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

// CollectOnce - prints just one time.
// eg. atwCm.Collect("This is a serious bug")
func (collector *StringEventAccumulator) CollectOnce(str string) {

	collector.Lock()
	here, ok := collector.onceSet[str]
	collector.Unlock()
	_ = ok
	if !here {
		fmt.Println(str) // leave this. It's the exception to the rule.
		collector.Lock()
		collector.onceSet[str] = true
		collector.Unlock()
	}
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

	collector.RLock()
	for k := range collector.countMap {
		keys = append(keys, k)
	}
	collector.RUnlock()

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

// StartRunningReports is
func StartRunningReports() {
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
		t := time.Now()
		fmt.Println("Report#" + strconv.Itoa(reportCount) + " " + t.Format("2006 01 02 15:04:05")) // don't delete this Println
		fmt.Println(sb.String())
		fmt.Println("")
		previousTime = t
		reportCount++
	}
}
