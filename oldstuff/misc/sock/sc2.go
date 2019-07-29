package main

import (
	"fmt"
	"knotfree/oldstuff/iot"
	"time"
)

// ChanAndSubWithTCP2 sis
func ChanAndSubWithTCP2(chanCount, testCount int) {

	fmt.Println("hello ChanAndSubWithTCP2")

	iot.ResetAllTheConnectionsMap(chanCount)

	subMgr := iot.NewPubsubManager(testCount)
	_ = subMgr

	var serverSockets []*BytesDuplexChannel

	localCallback := func(dc *BytesDuplexChannel) {
		fmt.Println("server accepted ")
		serverSockets = append(serverSockets, dc)
		// server reads from the Up-load channel
		for {
			got := <-*dc.Up
			fmt.Println("somone sent to server ", string(got))
			// echo it back
			*dc.Down <- got
		}
	}
	servererr := func(dc *BytesDuplexChannel, err error) {
		fmt.Println("server is closing ", err)
	}
	Serve(localCallback, servererr, 0)

	localSendBack := func(dc *BytesDuplexChannel) {
		fmt.Println("client dialed ")
		// this goes up to the server
		*dc.Up <- []byte("The quick fox jumped over the lazy dog.")
		for {
			reply := <-*dc.Down
			fmt.Println("Server replied:", string(reply))
		}

	}
	clientrerr := func(dc *BytesDuplexChannel, err error) {
		fmt.Println("client is closing ", err)
	}
	client, _ := Call(localSendBack, clientrerr, 0)
	fmt.Println("sleeping ")
	time.Sleep(10 * time.Second)
	// for _, xx := range serverSockets {
	// 	xx.Close()
	// }
	client.Close()
	time.Sleep(10 * time.Second)

}
