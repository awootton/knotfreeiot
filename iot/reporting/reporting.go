// Copyright 2019 Alan Tracey Wootton
// Copyright 2019,2020 Alan Tracey Wootton
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package reporting

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// TODO: cleanup

// DoStartEventCollectorReporting - set to run the reporter
//var DoStartEventCollectorReporting = true

//var delayBetweenReports = 6 per minute
var delayBetweenReports = 10 * time.Second

// Reporting - besides the StringEventAccumulator other reports will be needed.
// they can implement this interface. See
type Reporting interface {
	report(float32) []string
}

var mutex = sync.Mutex{}
var reporters = make([]Reporting, 0, 25)

// GenericEventAccumulator is for when we don't want to collect events or make sums
// and what we need to to just contribute some values to the report.
type GenericEventAccumulator struct {
	reporter func(float32) []string
}

// StringEventAccumulator a
type StringEventAccumulator struct {
	sync.RWMutex
	countMap map[string]float32
	onceSet  map[string]bool
	strlen   int  // for trimming keys
	quiet    bool // don't Println incoming msgs
}

// NewStringEventAccumulator is
func NewStringEventAccumulator(maxstrlen int) *StringEventAccumulator {
	cm := StringEventAccumulator{}
	cm.countMap = make(map[string]float32)
	cm.onceSet = make(map[string]bool)
	if maxstrlen < 4 {
		maxstrlen = 4
	}
	cm.strlen = maxstrlen
	cm.quiet = true
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

// SetQuiet is
func (collector *StringEventAccumulator) SetQuiet(q bool) {
	collector.quiet = q
}

func (me *GenericEventAccumulator) report(seconds float32) []string {
	return me.reporter(seconds)
}

// addReporter appends r to the global list of reporters.
func addReporter(r Reporting) {
	mutex.Lock()
	reporters = append(reporters, r)
	mutex.Unlock()
}

// Collect - Users will call this when strings happen and we'll count the rate.
// eg. atwCm.Collect("This is a serious bug")
func (collector *StringEventAccumulator) Collect(str string) {
	collector.Sum(str, 1)
}

// CollectOnce - prints just one time.
// eg. if you call this a million times: atwCm.Collect("This is a serious bug")
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

var latestReport = "unset"

// GetLatestReport is global.
func GetLatestReport() string {
	return latestReport
}

// StartRunningReports is
func StartRunningReports() {
	mutex.Lock()
	tmp := reportCount
	mutex.Unlock()
	if tmp > 0 {
		return
	}
	mutex.Lock()
	reportCount = 1
	mutex.Unlock()
	reportTicker := time.NewTicker(delayBetweenReports)
	previousTime := time.Now()

	for t := range reportTicker.C {
		var sb strings.Builder
		width := 0
		elapsed := t.Sub(previousTime)
		rcopy := make([]Reporting, len(reporters))
		mutex.Lock()
		copy(rcopy, reporters)
		mutex.Unlock()
		for _, reporter := range rcopy {
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
		// sssarr := memStats()
		// for _, str := range sssarr {
		// 	sb.WriteString(str)
		// 	sb.WriteString("    ")
		// 	width += len(str) + 1
		// 	if width > 120 {
		// 		sb.WriteString("\n")
		// 		width = 0
		// 	}
		// }
		t := time.Now()
		report := "Report#" + strconv.Itoa(reportCount) + " " + t.Format("2006 01 02 15:04:05") + "\n"
		report = report + sb.String()
		latestReport = report
		fmt.Println(report) // don't delete this Println
		fmt.Println("")
		previousTime = t
		reportCount++
	}
}

func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}

func bToKb(b uint64) uint64 {
	return b / 1024
}