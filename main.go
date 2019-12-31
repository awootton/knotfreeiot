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

package main

import (
	"flag"
	"fmt"
	"knotfreeiot/aaprotocol"
	"knotfreeiot/iot"
	"knotfreeiot/iot/reporting"
	"knotfreeiot/iot/tiers"
	"knotfreeiot/mqttprotocol"
	"knotfreeiot/strprotocol"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Hint: add 127.0.0.1 knotfreeserver to /etc/hosts
func main() {

	fmt.Println("Hello knotfreeserver")

	tiers.TwoByTwoTest()

	aa := flag.Bool("aa", false, "use aa protocol")
	str := flag.Bool("str", false, "use str protocol")
	mqtt := flag.Bool("mqtt", false, "use mqtt protocol")
	client := flag.Int("client", 0, "start a client test with an int of clients.")
	server := flag.Bool("server", false, "start a server.")

	// eg. ["-client=10","-server","-str"","-aa"]  starts 10 clients in each of two protocols

	flag.Parse()

	if *server {
		if *aa {
			go aaProtocolServerDemo()
		}
		if *str {
			go strProtocolServerDemo()
		}
		if *mqtt {
			go mqttProtocolServerDemo()
		}
	}
	if *client > 0 {
		if *aa {
			go aaProtocolClientDemo(*client)
		}
		if *str {
			go strProtocolClientDemo(*client)
		}
		if *mqtt {
			go mqttProtocolClientDemo(*client)
		}
	}

	go func() {
		http.HandleFunc("/", HelloServer)
		err := http.ListenAndServe(":8080", nil)
		if err != nil {
			fmt.Println("ListenAndServe err ", err)
		}
	}()

	fmt.Println("starting reporter")

	reportTicker := time.NewTicker(10 * time.Second)
	for t := range reportTicker.C {

		strlist := strings.Builder{}
		var m runtime.MemStats
		runtime.ReadMemStats(&m) // FIXME: this deadlocks and hangs.

		strlist.WriteString("Bytes= " + strconv.FormatUint(bToMb(m.HeapAlloc), 10) + " MiB ")
		strlist.WriteString("Sys= " + strconv.FormatUint(bToMb(m.Sys), 10) + " MiB ")
		strlist.WriteString("GC= " + strconv.FormatUint(bToMb(uint64(m.NumGC)), 10) + " ")

		_ = t

		fmt.Println("", strlist.String())
	}
}

func strProtocolServerDemo() {

	fmt.Println("Starting strProtocolServerDemo")
	config := strprotocol.StartServerDemo(getSubscribeMgr(), "7374")
	_ = config
	for {
		time.Sleep(time.Minute)
	}
}

func aaProtocolServerDemo() {

	fmt.Println("Starting aaProtocolServerDemo")
	config := aaprotocol.StartServerDemo(getSubscribeMgr())
	_ = config
	for {
		time.Sleep(time.Minute)
	}
}

func mqttProtocolServerDemo() {

	fmt.Println("Starting mqttProtocolServerDemo")
	config := mqttprotocol.StartServerDemo(getSubscribeMgr())
	_ = config
	for {
		time.Sleep(time.Minute)
	}
}

func strProtocolClientDemo(count int) {

	fmt.Println("Starting strProtocolClientDemo", count)
	lights, switches := strprotocol.StartClientsDemo(count)

	for 1 == 1 {
		time.Sleep(time.Minute)
	}
	_ = lights
	_ = switches
}

func aaProtocolClientDemo(count int) {

	fmt.Println("Starting aaProtocolClientDemo", count)
	lights, switches := aaprotocol.StartClientsDemo(count)

	for 1 == 1 {
		time.Sleep(time.Minute)
	}
	_ = lights
	_ = switches

}

func mqttProtocolClientDemo(count int) {

	fmt.Println("Starting mqttProtocolClientDemo", count)
	lights, switches := mqttprotocol.StartClientsDemo(count)
	for 1 == 1 {
		time.Sleep(time.Minute)
	}
	_ = lights
	_ = switches

}

// HelloServer is
func HelloServer(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello, %s! %v \n", r.URL.Path[1:], reporting.GetLatestReport())
}

var mainLogThing = reporting.NewStringEventAccumulator(16)

// func startReportingHere() {
// 	aReportFunc := func(seconds float32) []string {
// 		strlist := make([]string, 0, 2)
// 		var m runtime.MemStats
// 		runtime.ReadMemStats(&m)

// 		strlist = append(strlist, "Bytes="+strconv.FormatUint(bToMb(m.HeapAlloc), 10)+"MiB")
// 		strlist = append(strlist, "Sys="+strconv.FormatUint(bToMb(m.Sys), 10)+"MiB")
// 		strlist = append(strlist, "GC="+strconv.FormatUint(bToMb(uint64(m.NumGC)), 10))

// 		return strlist
// 	}
// 	reporting.NewGenericEventAccumulator(aReportFunc)
//go reporting.StartRunningReports()
//}

func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}

var subscribeMgr iot.PubsubIntf
var subscribeMgrMutex sync.Mutex

func getSubscribeMgr() iot.PubsubIntf {
	subscribeMgrMutex.Lock()
	if subscribeMgr == nil {
		subscribeMgr = iot.NewPubsubManager(100 * 1000)
	}
	subscribeMgrMutex.Unlock()
	return subscribeMgr
}
