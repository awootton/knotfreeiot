package knotfree

import (
	"net"
)

var srvrLogThing *StringEventAccumulator

func init() {
	srvrLogThing = NewStringEventAccumulator(16)
	srvrLogThing.quiet = true
}

// Server - wait for connections and spawn them
func Server() {
	ln, err := net.Listen("tcp", ":8080")
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
		go RunAConnection(&c)
	}
}
