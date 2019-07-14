package main

import (
	"fmt"
	"knotfree/knotfree"
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
		go knotfree.LightSwitch("aaaaaa" + istr)
		go knotfree.LightController("bbbbb"+istr, "aaaaaa"+istr)
	}
}

func main() {

	//knotfree.RunTCPOverPubsub()

	if 3 == 1+1 {
		return
	}

	if len(os.Args) > 1 && os.Args[1] == "client" {
		go runClients(1)
	} else {
		go knotfree.Server()
		go runClients(1)
	}

	for {
	}

}
