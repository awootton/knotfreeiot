// Copyright 2019 Alan Tracey Wootton

// Package iot provides   pub/sub
package iot

import (
	"fmt"
	"knotfree/protocolaa" // FIXME: get rid of this
	"knotfree/types"
	"net"
	"strings"

	"time"
)

// Server - wait for connections and spawn them
// runs forever
// TODO: handlerFactory as argument.
func Server(subscribeMgr types.SubscriptionsIntf) {
	fmt.Println("Server starting")
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
		go handleConnection(tmpconn, subscribeMgr) //,handler types.ProtocolHandler)
	}
}

// RunAConnection - FIXME: this is really a protoA connection.
//
func handleConnection(tmpconn net.Conn, subscribeMgr types.SubscriptionsIntf) {

	srvrLogThing.Collect("Conn Accept")

	c := NewConnection(tmpconn.(*net.TCPConn), subscribeMgr)

	// FIXME: pass a factory
	// not always aa
	handler := protocolaa.NewServerHandler(c, subscribeMgr)
	c.SetProtocolHandler(handler)

	defer c.Close()

	connLogThing.Collect("new connection")

	allConnMutex.Lock()
	allTheConnections[*c.GetKey()] = c
	allConnMutex.Unlock()
	// start reading
	err := c.GetTCPConn().SetReadBuffer(4096)
	if err != nil {
		connLogThing.Collect("server err " + err.Error())
		return
	}
	err = c.GetTCPConn().SetWriteBuffer(4096)
	if err != nil {
		connLogThing.Collect("cserver " + err.Error())
		return
	}

	// we might just for over the range of the handler input channel?
	for true { // c.running {
		// SetReadDeadline
		err := c.GetTCPConn().SetDeadline(time.Now().Add(20 * time.Minute))
		if err != nil {
			connLogThing.Collect("server err2 " + err.Error())
			return // quit, close the sock, be forgotten
		}
		err = handler.Serve()
		if err != nil {
			connLogThing.Collect("se err " + err.Error())
			return // quit, close the sock, be forgotten
		}
	}
}

// isClosedConnError reports whether err is an error from use of a closed
// network connection.
func isClosedConnError(err error) bool {
	if err == nil {
		return false
	}

	// TODO: remove this string search and be more like the Windows
	// case below. That might involve modifying the standard library
	// to return better error types.
	str := err.Error()
	if strings.Contains(str, "use of closed network connection") {
		return true
	}

	// TODO(bradfitz): x/tools/cmd/bundle doesn't really support
	// build tags, so I can't make an http2_windows.go file with
	// Windows-specific stuff. Fix that and move this, once we
	// have a way to bundle this into std's net/http somehow.
	// if runtime.GOOS == "windows" {
	// 	if oe, ok := err.(*net.OpError); ok && oe.Op == "read" {
	// 		if se, ok := oe.Err.(*os.SyscallError); ok && se.Syscall == "wsarecv" {
	// 			const WSAECONNABORTED = 10053
	// 			const WSAECONNRESET = 10054
	// 			if n := errno(se.Err); n == WSAECONNRESET || n == WSAECONNABORTED {
	// 				return true
	// 			}
	// 		}
	// 	}
	// }
	return false
}

var srvrLogThing *types.StringEventAccumulator

func init() {
	srvrLogThing = types.NewStringEventAccumulator(16)
	srvrLogThing.SetQuiet(true)
}

// SocketSetup sets common options
func SocketSetup(conn net.Conn) error {
	tcpConn := conn.(*net.TCPConn)
	err := tcpConn.SetReadBuffer(4096)
	if err != nil {
		srvrLogThing.Collect("SS err1 " + err.Error())
		return err
	}
	err = tcpConn.SetWriteBuffer(4096)
	if err != nil {
		srvrLogThing.Collect("SS err2 " + err.Error())
		return err
	}
	err = tcpConn.SetNoDelay(true)
	if err != nil {
		srvrLogThing.Collect("SS err3 " + err.Error())
		return err
	}
	// SetReadDeadline and SetWriteDeadline
	err = tcpConn.SetDeadline(time.Now().Add(20 * time.Minute))
	if err != nil {
		srvrLogThing.Collect("cl err4 " + err.Error())
		return err
	}
	return nil
}
