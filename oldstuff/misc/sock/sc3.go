package main

import (
	"fmt"
	"knotfreeiot/oldstuff/iot"
	"sync"
	"sync/atomic"
	"time"
)

// ChanAndSubWithTCP3 is when we try it with io. interfaces instead of channels
func ChanAndSubWithTCP3(chanCount, testCount int) {

	//fmt.Println("hello ChanAndSubWithTCP3")

	iot.ResetAllTheConnectionsMap(chanCount)

	subMgr := iot.NewPubsubManager(testCount)
	_ = subMgr

	var sslock sync.RWMutex
	var serverSockets []*BytesDuplexChannel

	var lenServers = func() int {
		sslock.Lock()
		x := len(serverSockets)
		sslock.Unlock()
		return x
	}

	serverCallback := func(dc *BytesDuplexChannel) {
		sslock.Lock()
		serverSockets = append(serverSockets, dc)
		sslock.Unlock()
		// and then just return and there are no gr running
	}
	servererr := func(dc *BytesDuplexChannel, err error) {
		//("server is closing", err)
	}

	serverConfig := ServeNoGo(serverCallback, servererr, 0)
	//fmt.Println("server is started ", lenServers())

	// --- now the client sockets

	clientsCalledBack := int32(0)

	clietSendBack := func(dc *BytesDuplexChannel) {
		//fmt.Println("client dialed ")
		// we don't need to remember these because the caller got them directly
		atomic.AddInt32(&clientsCalledBack, 1)
	}
	clientrerr := func(dc *BytesDuplexChannel, err error) {
		//fmt.Println("client is closing ", err)
	}

	var clientSockets []*BytesDuplexChannel

	for i := 0; i < chanCount; i++ {
		client, err := CallNoGo(clietSendBack, clientrerr, 0)
		if err != nil {
			fmt.Println("not the day", err)
		}
		clientSockets = append(clientSockets, client)
	}
	//fmt.Println("clients started ")

	for lenServers() < chanCount {
		time.Sleep(50 * time.Millisecond)
	}
	//fmt.Println("everybody connected ")

	testString := "client to server, can you hear me?"

	for i := 0; i < chanCount; i++ {
		client := clientSockets[i]
		n, err := client.Write([]byte(testString))
		_ = n
		_ = err
		//fmt.Println("client to.. ", n, err)
	}
	// and now the other channels must be getting puffy buffers
	// so quick, before it times out...
	for i := 0; i < chanCount; i++ {
		server := serverSockets[i]
		buff := make([]byte, len(testString))
		n, err := server.Read(buff[0:])
		//fmt.Println("server got.. ", n, err, string(buff))
		_ = n
		_ = err
		if string(buff) != testString {
			fmt.Println("EXPECTED equal ", string(buff), testString)
		}

	}

	//fmt.Println("closing ")

	serverConfig.ln.Close()

	for _, ss := range serverSockets {
		ss.Close()
	}

	for _, ss := range clientSockets {
		ss.Close()
	}

	//time.Sleep(50 * time.Second)

}
