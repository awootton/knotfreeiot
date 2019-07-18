// Package iot provides   pub/sub
package iot

import (
	"knotfree/protocolaa"
	"knotfree/types"
	"math/rand"
	"net"
	"strings"

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
	// not always aa
	handler := protocolaa.NewServerHandler(c, GetSubscriptionsMgr())
	c.SetProtocolHandler(&handler)

	c.running = true
	// random connection id
	randomStr := strconv.FormatInt(rand.Int63(), 16) + strconv.FormatInt(rand.Int63(), 16)
	c.key.FromString(randomStr)
	//c.writesChannel = make(chan *types.IncomingMessage, 2)
	c.realTopicNames = make(map[types.HashType]string)

	defer c.Close()

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

	//go watchForData(c)
	// bytes, _ := json.Marshal(c)
	// fmt.Println("connection struct " + string(bytes))
	for c.running {

		err := c.tcpConn.SetReadDeadline(time.Now().Add(20 * time.Minute))
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
