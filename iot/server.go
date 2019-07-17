// Package iot provides   pub/sub
package iot

import (
	"knotfree/protocolaa"
	"knotfree/types"
	"math/rand"
	"net"

	"strconv"
	"time"
)

var srvrLogThing *types.StringEventAccumulator

func init() {
	srvrLogThing = types.NewStringEventAccumulator(16)
	srvrLogThing.SetQuiet(true)
}

// Server - wait for connections and spawn them
func Server() {
	ln, err := net.Listen("tcp", ":6161")
	if err != nil {
		// handle error
		srvrLogThing.Collect(err.Error())
		return
	}
	for {
		tmpconn, err := ln.Accept()
		if err != nil {
			srvrLogThing.Collect(err.Error())
			continue
		}
		srvrLogThing.Collect("Conn Accept")
		c := Connection{tcpConn: tmpconn.(*net.TCPConn)}
		go runTheConnection(&c) //,handler types.ProtocolHandler)
	}
}

// RunAConnection - FIXME: this is really a protoA connection.
//
func runTheConnection(c *Connection) {

	// FIXME: pass a factory
	handler := protocolaa.NewServerHandler(c, GetSubscriptionsMgr())
	c.SetProtocolHandler(&handler)

	defer c.Close()
	c.running = true
	// random connection id
	randomStr := strconv.FormatInt(rand.Int63(), 16) + strconv.FormatInt(rand.Int63(), 16)
	c.key.FromString(randomStr)
	c.writesChannel = make(chan *types.IncomingMessage, 2)
	c.realTopicNames = make(map[types.HashType]string)
	//c.Subscriptions = make(map[types.HashType]bool)
	connLogThing.Collect("new connection")
	allConnMutex.Lock()
	allTheConnections[c.key] = c
	allConnMutex.Unlock()
	// start reading
	err := c.tcpConn.SetReadBuffer(4096)
	if err != nil {
		connLogThing.Collect("server err " + err.Error())
		return
	}
	err = c.tcpConn.SetWriteBuffer(4096)
	if err != nil {
		connLogThing.Collect("cserver " + err.Error())
		return
	}

	go watchForData(c)
	// bytes, _ := json.Marshal(c)
	// fmt.Println("connection struct " + string(bytes))
	for c.running {

		for {
			err := c.tcpConn.SetReadDeadline(time.Now().Add(20 * time.Minute))
			if err != nil {
				connLogThing.Collect("server err2 " + err.Error())
				return
			}

			err = handler.Serve()
			if err != nil {
				connLogThing.Collect("handler err " + err.Error())
				return
			}
		}
	}
}
