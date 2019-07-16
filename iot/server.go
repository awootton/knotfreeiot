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

var srvrLogThing *StringEventAccumulator

func init() {
	srvrLogThing = NewStringEventAccumulator(16)
	srvrLogThing.quiet = false
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
	connLogThing.Collect("setting CONN " + c.key.String())
	allConnMutex.Lock()
	allTheConnections[c.key] = c
	allConnMutex.Unlock()
	// start reading
	err := c.tcpConn.SetReadBuffer(4096)
	if err != nil {
		connLogThing.Collect("server err " + err.Error())
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

			// str, err := ReadProtocolAstr(c.tcpConn)
			// if err != nil {
			// 	connLogThing.Collect("rProtA err " + str + err.Error())
			// 	return
			// }
			// connLogThing.Sum("Conn r bytes", len(str))
			// // ok, so what is the message? subscribe or publish?
			// //fmt.Println("Have Server str _a " + str)
			// // eg sAchannel

			// // CONNECT c
			// // PUBLISH p
			// // SUBSCRIBE s
			// // UNSUBSCRIBE u
			// // PING g
			// // DISCONNECT d

			// if str[0] == 's' {
			// 	// process subscribe
			// 	subTopic := str[1:]

			// 	//fmt.Println("sub CONN key " + c.Key.String())
			// 	//fmt.Println("Have subTopic " + subTopic)
			// 	// we'll fill in a sub request and 'mail' it to the sub handler
			// 	// TODO: change to proc call
			// 	subr := SubscriptionMessage{}
			// 	subr.Topic.FromString(subTopic)
			// 	//fmt.Println("Have subChan becomes " + subr.Channel.String())
			// 	subr.ConnectionID.FromHashType(&c.Key)
			// 	//fmt.Println("subscribe ConnectionID is " + subr.ConnectionID.String())
			// 	c.realTopicNames[subr.Topic] = subTopic
			// 	AddSubscription(subr)
			// 	//c.Subscriptions[subr.Topic] = true
			// } else if str[0] == 'p' {
			// 	//fmt.Println("publish CONN key " + c.Key.String())
			// 	// process publish, {"C":"channelRealName","M":"a message"}
			// 	//fmt.Println("got p publish " + str)
			// 	pub := PublishProtocolA{}
			// 	err := json.Unmarshal([]byte(str[1:]), &pub)
			// 	if err != nil {
			// 		connLogThing.Collect("server json " + err.Error())
			// 		return
			// 	}
			// 	//fmt.Println("Have PublishProtocolA " + string(&pub))
			// 	pubr := PublishMessage{}
			// 	pubr.Topic.FromString(pub.T)
			// 	pubr.ConnectionID.FromHashType(&c.Key)
			// 	pubr.Message = []byte(pub.M)
			// 	//fmt.Println("Publish topic becomes " + pubr.Topic.String())
			// 	//fmt.Println("Publish ConnectionID is " + pubr.ConnectionID.String())
			// 	AddPublish(pubr)
			// }
			// // we don't have an unsubscribe yet.

		}
	}
}
