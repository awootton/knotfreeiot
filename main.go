package main

import (
	"fmt"
	"knotfree/clients"
	"knotfree/iot"
	"knotfree/types"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"
)

// func init() {
// 	subscriptionmgr.Qmessage = &knotfree.Qmessage
// }

var prefix = ""

func runClients(amt int) {
	clients.ExpectedConnections = amt
	fmt.Println("Starting clients = " + strconv.Itoa(amt))
	for i := 0; i < amt; i++ {
		istr := strconv.Itoa(i)
		go clients.LightSwitch(prefix+"aaaaaa"+istr, prefix+"bbbbb"+istr)
		go clients.LightController(prefix+"bbbbb"+istr, prefix+"aaaaaa"+istr)
	}
}

func runServer() {
	iot.ResetAllTheConnectionsMap(100 * 1000)

	var subscribeMgr types.SubscriptionsIntf
	subscribeMgr = iot.NewPubsubManager(100 * 1000)
	psMgr = subscribeMgr
	types.NewGenericEventAccumulator(subscrFRepofrtFunct)

	iot.Server(subscribeMgr)
}

var subscrFRepofrtFunct = func(seconds float32) []string {
	strlist := make([]string, 0, 5)
	count := psMgr.GetAllSubsCount()
	strlist = append(strlist, "Topic count="+strconv.FormatUint(count, 10))
	return strlist
}

var psMgr types.SubscriptionsIntf

// add 127.0.0.1 knotfreeserver to /etc/hosts
func main() {

	fmt.Println("Hello3")
	prefix = "_" + strconv.FormatUint(uint64(rand.Uint32()), 16) + "_/"
	fmt.Println("using prefix " + prefix)

	args := os.Args
	arglen := len(args)
	_ = arglen

	if len(os.Args) > 1 && os.Args[1] == "client" {
		n := 9999
		if len(os.Args) > 2 {
			tmp, err := strconv.ParseInt(os.Args[2], 10, 32)
			if err == nil {
				n = int(tmp)
			} else {
				fmt.Println(err)
			}
		}
		go types.StartRunningReports()
		go runClients(n)
	} else if len(os.Args) > 1 && os.Args[1] == "server" {
		go types.StartRunningReports()
		go runServer()
	} else {
		go types.StartRunningReports()
		go runServer()
		go runClients(2000)
	}

	http.HandleFunc("/", HelloServer)
	http.ListenAndServe(":8080", nil)

	for {
		time.Sleep(60 * time.Minute)
	}
}

var serveCount = 1

// HelloServer is
func HelloServer(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello, %s! %v \n", r.URL.Path[1:], serveCount)
	serveCount++
}
