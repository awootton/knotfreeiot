package main

import (
	"fmt"
	"knotfree/clients"
	"knotfree/iot"
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

// add 127.0.0.1 knotfreeserver to /etc/hosts
func main() {

	fmt.Println("Hello")
	prefix = "_" + strconv.FormatUint(uint64(rand.Uint32()), 16) + "_/"
	fmt.Println("using prefix " + prefix)

	if 3 == 1+1 {
		iot.RunTCPOverPubsub()
		return
	}

	if len(os.Args) > 1 && os.Args[1] == "client" {
		go runClients(12000)
	} else if len(os.Args) > 1 && os.Args[1] == "server" {
		go iot.Server()
	} else {
		go iot.Server()
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
