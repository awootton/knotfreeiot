package main

import (
	"flag"
	"fmt"
	"knotfreeiot/aaprotocol"
	"knotfreeiot/iot"
	"knotfreeiot/iot/reporting"
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

	for 1 == 2 {
		time.Sleep(time.Minute)
	}

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

func strProtocolServerDemo() {

	fmt.Println("Starting strProtocolServerDemo")
	config := strprotocol.StartServerDemo(getSubscribeMgr())
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
