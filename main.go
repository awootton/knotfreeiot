package main

import (
	"flag"
	"fmt"
	"knotfreeiot/aaprotocol"
	"knotfreeiot/iot/reporting"
	"knotfreeiot/strprotocol"
	"net/http"
	"runtime"
	"time"
)

func strProtocolServerDemo() {

	fmt.Println("Starting strProtocolServerDemo")
	config := strprotocol.StartServerDemo(100 * 1000)
	_ = config
	for {
		time.Sleep(time.Minute)
	}
}

func aaProtocolServerDemo() {

	fmt.Println("Starting aaProtocolServerDemo")
	config := aaprotocol.StartServerDemo(100 * 1000)
	_ = config
	for {
		time.Sleep(time.Minute)
	}
}

func strProtocolClientDemo(count int) {

	fmt.Println("Starting strProtocolClientDemo", count)
	lights, switches := strprotocol.StartClientsDemo(count)
	_ = lights
	_ = switches
	for {
		time.Sleep(time.Minute)
	}
}

func aaProtocolClientDemo(count int) {

	fmt.Println("Starting aaProtocolClientDemo", count)
	lights, switches := aaprotocol.StartClientsDemo(count)
	_ = lights
	_ = switches
	for {
		time.Sleep(time.Minute)
	}
}

// Hint: add 127.0.0.1 knotfreeserver to /etc/hosts
func main() {

	fmt.Println("Hello knotfreeserver")

	aa := flag.Bool("aa", false, "use aa protocol")
	str := flag.Bool("str", false, "use str protocol")
	client := flag.Int("client", 0, "start a client test, else start the servers")
	server := flag.Bool("server", false, "start a server even if we're also starting the clients test")

	flag.Parse()

	if *server {
		if *aa {
			go aaProtocolServerDemo()
		}
		if *str {
			go strProtocolServerDemo()
		}
	}
	if *client > 0 {
		if *aa {
			go aaProtocolClientDemo(*client)
		}
		if *str {
			go strProtocolClientDemo(*client)
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

		// strlist := strings.Builder{}
		var m runtime.MemStats
		runtime.ReadMemStats(&m) // FIXME: this deadlocks and hangs.

		// 	// 	strlist.WriteString("Bytes=" + strconv.FormatUint(bToMb(m.HeapAlloc), 10) + "MiB")
		// 	// 	strlist.WriteString("Sys=" + strconv.FormatUint(bToMb(m.Sys), 10) + "MiB")
		// 	// 	strlist.WriteString("GC=" + strconv.FormatUint(bToMb(uint64(m.NumGC)), 10))

		_ = t
	}
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
