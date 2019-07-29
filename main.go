package main

import (
	"bufio"
	"fmt"
	"runtime"
	"sync/atomic"

	"knotfree/iot2"
	"knotfree/iot2/reporting"

	"math/rand"
	"net/http"
	"strconv"
	"time"
)

var prefix = ""

// the old way in oldstuff/iot
// func runClients(amt int) {
// 	clients.ExpectedConnections = amt
// 	fmt.Println("Starting clients = " + strconv.Itoa(amt))
// 	for i := 0; i < amt; i++ {
// 		istr := strconv.Itoa(i)
// 		go clients.LightSwitch(prefix+"aaaaaa"+istr, prefix+"bbbbb"+istr)
// 		go clients.LightController(prefix+"bbbbb"+istr, prefix+"aaaaaa"+istr)
// 	}
// }

func runAlight(ss *iot2.SockStruct) {

	// context to read loop decl here

	go func(ss *iot2.SockStruct) {
		reader := bufio.NewReader(ss.GetConn())
		for { // our reading loop
			text, err := reader.ReadString('\n')
			if err != nil {
				ss.Close(err)
				return
			}
			str := text[0 : len(text)-1]
			//fmt.Println("Light received", str)
			cmd, payload := iot2.GetFirstWord(str)
			if cmd == "got" {
				// just pub back to switch
				topicfrom, payload := iot2.GetFirstWord(payload)
				myID := ss.GetSequence()
				topic := "switch_" + strconv.FormatUint(0x000FFFFF&myID, 16)

				command := "pub " + topic + " " + payload
				iot2.ServerOfStringsWrite(ss, command)
				_ = topicfrom
			} else {
				// it's just the echo of out own pub and sub commands.
				// fmt.Println("not handled", cmd, payload)
			}
		}
	}(ss)

	myID := ss.GetSequence()
	idstr := strconv.FormatUint(0x000FFFFF&myID, 16)

	topic := "light_" + idstr
	// now convert to command
	cmd := "sub " + topic
	iot2.ServerOfStringsWrite(ss, cmd) // send subscribe command to server

	for {
		// our sending loop
		// lights don't send. light controllers, aka switches, send commands.
		time.Sleep(10 * time.Second)
	}
}

func runAswitch(ss *iot2.SockStruct) {

	myID := ss.GetSequence()
	idstr := strconv.FormatUint(0x000FFFFF&myID, 16)

	ourCommand := "Hello From Switch_" + idstr

	waiting := int32(0)
	when := time.Now()

	go func(ss *iot2.SockStruct) {
		reader := bufio.NewReader(ss.GetConn())
		for { // our reading loop
			text, err := reader.ReadString('\n')
			if err != nil {
				ss.Close(err)
				return
			}
			str := text[0 : len(text)-1] // remove \n
			cmd, tmp := iot2.GetFirstWord(str)
			topic, payload := iot2.GetFirstWord(tmp)
			if cmd == "got" {
				if payload == ourCommand {
					atomic.AddInt32(&waiting, -1)
					duration := time.Now().Sub(when)
					if duration < time.Millisecond*100 {
						clientLogThing.Collect("happy joy")
					} else if duration < time.Second {
						clientLogThing.Collect("ok")
					} else {
						clientLogThing.Collect("too slow")
					}
				}
			}
			_ = topic
		}
	}(ss)

	topic := "switch_" + idstr
	// now convert to command
	cmd := "sub " + topic
	iot2.ServerOfStringsWrite(ss, cmd) // send subscribe command to se

	for {
		// our sending loop
		topic := "light_" + idstr
		command := "pub " + topic + " " + ourCommand
		atomic.AddInt32(&waiting, 1)
		when = time.Now()
		iot2.ServerOfStringsWrite(ss, command) // send pub command to server
		time.Sleep(10 * time.Second)
	}
}

var clientLogThing *reporting.StringEventAccumulator

func runServer2() {

	clientLogThing = reporting.NewStringEventAccumulator(16)
	clientLogThing.SetQuiet(true)

	var subscribeMgr iot2.PubsubIntf
	subscribeMgr = iot2.NewPubsubManager(100 * 1000)

	//config := iot2.ServerOfAa(subscribeMgr, ":6161")
	config := iot2.ServerOfStrings(subscribeMgr, ":7374")

	defer config.Close(nil)

	addr := "knotfreeserver:7374"

	lights := iot2.NewSockStructConfig(nil)
	iot2.ServerOfStringsInit(lights)
	switches := iot2.NewSockStructConfig(nil)
	iot2.ServerOfStringsInit(switches)

	clientCount := 5000

	subscrFRepofrtFunct := func(seconds float32) []string {
		strlist := make([]string, 0, 5)
		count := config.Len()
		strlist = append(strlist, "Conn count="+strconv.FormatUint(uint64(count), 10))
		topicCpunt, buffAvg := subscribeMgr.GetAllSubsCount()
		strlist = append(strlist, "Topic count="+strconv.FormatUint(uint64(topicCpunt), 10))
		strlist = append(strlist, "Topic buffers="+strconv.FormatUint(uint64(buffAvg), 10))
		strlist = append(strlist, "Lights="+strconv.FormatUint(uint64(lights.Len()), 10))
		strlist = append(strlist, "Switches="+strconv.FormatUint(uint64(switches.Len()), 10))

		var m runtime.MemStats
		runtime.ReadMemStats(&m)

		strlist = append(strlist, "Bytes="+strconv.FormatUint(bToMb(m.HeapAlloc), 10)+"MiB")
		strlist = append(strlist, "Sys="+strconv.FormatUint(bToMb(m.Sys), 10)+"MiB")
		strlist = append(strlist, "GC="+strconv.FormatUint(bToMb(uint64(m.NumGC)), 10))

		return strlist
	}
	reporting.NewGenericEventAccumulator(subscrFRepofrtFunct)
	go reporting.StartRunningReports()

	fmt.Println("start making clients")

	lights.SetCallback(runAlight)
	iot2.MakeBunchOfClients(clientCount, addr, 10*time.Millisecond, lights, clientLogThing)

	fmt.Println("lights started")

	switches.SetCallback(runAswitch)
	iot2.MakeBunchOfClients(clientCount, addr, 10*time.Millisecond, switches, clientLogThing)

	fmt.Println("switches started")

	fmt.Println("StartRunningReports called")

	for {
		time.Sleep(10 * time.Second) //

		// open a client and send a message

		fmt.Println("sockets=", config.Len())
	}
}

// the old way in oldstuff/iot
// func runServer() {
// 	iot.ResetAllTheConnectionsMap(100 * 1000)

// 	var subscribeMgr types.SubscriptionsIntf
// 	subscribeMgr = iot.NewPubsubManager(100 * 1000)
// 	psMgr = subscribeMgr

// 	iot.Server(subscribeMgr)

// 	subscrFRepofrtFunct := func(seconds float32) []string {
// 		strlist := make([]string, 0, 5)
// 		count := subscribeMgr.GetAllSubsCount()
// 		strlist = append(strlist, "Topic count="+strconv.FormatUint(count, 10))
// 		return strlist
// 	}
// 	types.NewGenericEventAccumulator(subscrFRepofrtFunct)

// }

//var psMgr types.SubscriptionsIntf

// Hint: add 127.0.0.1 knotfreeserver to /etc/hosts
func main() {

	fmt.Println("Hello3")
	prefix = "_" + strconv.FormatUint(uint64(rand.Uint32()), 16) + "_/"
	fmt.Println("using prefix " + prefix)

	go runServer2()

	// args := os.Args
	// arglen := len(args)
	// _ = arglen

	// if len(os.Args) > 1 && os.Args[1] == "client" {
	// 	n := 9999
	// 	if len(os.Args) > 2 {
	// 		tmp, err := strconv.ParseInt(os.Args[2], 10, 32)
	// 		if err == nil {
	// 			n = int(tmp)
	// 		} else {
	// 			fmt.Println(err)
	// 		}
	// 	}
	// 	go types.StartRunningReports()
	// 	go runClients(n)
	// } else if len(os.Args) > 1 && os.Args[1] == "server" {
	// 	go types.StartRunningReports()
	// 	go runServer()
	// } else {
	// 	// go types.StartRunningReports()
	// 	// go runServer()
	// 	// go runClients(2000)
	// 	go runServer2()

	// }

	http.HandleFunc("/", HelloServer)
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Println("ListenAndServe err ", err)
	}

	for {
		time.Sleep(60 * time.Minute)
	}
}

// HelloServer is
func HelloServer(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello, %s! %v \n", r.URL.Path[1:], reporting.GetLatestReport())
}

func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}

func bToKb(b uint64) uint64 {
	return b / 1024
}
