package main

import (
	"fmt"
	"knotfree/clients"
	"knotfree/iot"
	"net/http"
	"os"
	"strconv"
	"time"
)

// func init() {
// 	subscriptionmgr.Qmessage = &knotfree.Qmessage
// }

func runClients(amt int) {
	fmt.Println("Starting clients = " + strconv.Itoa(amt))
	for i := 0; i < amt; i++ {
		istr := strconv.Itoa(i)
		go clients.LightSwitch("aaaaaa"+istr, "bbbbb"+istr)
		go clients.LightController("bbbbb"+istr, "aaaaaa"+istr)
	}
}

// add 127.0.0.1 knotfreeserver to /etc/hosts
func main() {

	fmt.Println("Hello")

	if 3 == 1+1 {
		iot.RunTCPOverPubsub()
		return
	}

	if len(os.Args) > 1 && os.Args[1] == "client" {
		go runClients(20000)
	} else if len(os.Args) > 1 && os.Args[1] == "server" {
		go iot.Server()
	} else {
		go iot.Server()
		go runClients(1)
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
