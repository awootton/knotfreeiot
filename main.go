package main

import (
	"fmt"
	"knotfree/clients"
	"knotfree/iot"
	"os"
	"strconv"
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

func main() {

	if 3 == 1+1 {
		iot.RunTCPOverPubsub()
		return
	}

	if len(os.Args) > 1 && os.Args[1] == "client" {
		go runClients(1)
	} else {
		go iot.Server()
		go runClients(1)
	}

	for {
	}

}
