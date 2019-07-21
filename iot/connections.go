// Copyright 2019 Alan Tracey Wootton

// Package iot manages Connections, literal tcp system sockets, using an object called Connection in the file
// connections.go
package iot

import (
	"knotfree/types"
	"math/rand"
	"net"
	"strconv"
	"sync"
)

// for every TCP socket there will be a Connection struct
// make private?

// Connection - wait
// we don't need this to be public
type Connection struct {
	key types.HashType // 128 bits
	//Subscriptions map[types.HashType]bool
	running bool
	//writesChannel chan *types.IncomingMessage // channel for receiving
	tcpConn *net.TCPConn
	// map from Topic hash to Topic string
	realTopicNames map[types.HashType]string

	protocolHandler *types.ProtocolHandler
	subcribeMgr     types.SubscriptionsIntf
}

// NewConnection is used by Server.go
// the new struct is NOT inserted into the global list.
func NewConnection(tcpConn *net.TCPConn, subscribeMgr types.SubscriptionsIntf) Connection {

	c := Connection{}
	c.tcpConn = tcpConn
	c.subcribeMgr = subscribeMgr
	c.running = true // do we need this?
	// random connection id
	randomStr := strconv.FormatInt(rand.Int63(), 16) + strconv.FormatInt(rand.Int63(), 16)
	c.key.FromString(randomStr)
	//c.writesChannel = make(chan *types.IncomingMessage, 2)
	c.realTopicNames = make(map[types.HashType]string)

	return c
}

// This is a Set of all the Connection structs that can be looked up by Key.
var allTheConnections = make(map[types.HashType]*Connection)
var allConnMutex = &sync.RWMutex{}

// ResetAllTheConnectionsMap for until we use a pointer to a connections manager.
// Clears them all out.
func ResetAllTheConnectionsMap() {
	allConnMutex.Lock()
	allTheConnections = make(map[types.HashType]*Connection)
	allConnMutex.Unlock()
}

// SetProtocolHandler is just a setter
func (c *Connection) SetProtocolHandler(protocolHandler *types.ProtocolHandler) {
	c.protocolHandler = protocolHandler
}

// GetTCPConn is spelled correctly
func (c *Connection) GetTCPConn() *net.TCPConn {
	return c.tcpConn
}

// SetRealTopicName is used by someone who shouldn't?
func (c *Connection) SetRealTopicName(h *types.HashType, s string) {
	c.realTopicNames[*h] = s
}

// GetRealTopicName is used ...?
func (c *Connection) GetRealTopicName(h *types.HashType) (string, bool) {
	str, ok := c.realTopicNames[*h]
	return str, ok
}

// GetKey is a getter
func (c *Connection) GetKey() *types.HashType {
	return &c.key
}

// ConnectionExists reports if it's still in the table.
func ConnectionExists(channelID *types.HashType) bool {
	allConnMutex.RLock()
	_, ok := allTheConnections[*channelID]
	allConnMutex.RUnlock()
	return ok
}

// RememberConnection -- and do forget later. See Close()
func RememberConnection(c *Connection) {
	allConnMutex.Lock()
	allTheConnections[c.key] = c
	allConnMutex.Unlock()
}

// QueueMessageToConnection called by subscribe. needs to access allTheConnections and the Connection.writesChannel
func QueueMessageToConnection(channelID *types.HashType, message *types.IncomingMessage) {

	connLogThing.Collect("q publish incoming " + channelID.String())
	allConnMutex.RLock()
	c, ok := allTheConnections[*channelID]
	allConnMutex.RUnlock()
	if ok == false {
		// someone is sending a message to a lost channel
		// c will be nil
		return
	}
	handler := *c.protocolHandler
	err := handler.HandleWrite(message)
	if err != nil {
		// only timeout errors or socket errors will happen
		// and they will stuff an error into the pipe that the
		// poll (in server.runTheConnection) will break on and that will close the socket
		c.running = false
	}
}

// Close will kill the connection for disconnect or timeout or error or whatever.
func (c *Connection) Close() {
	// TODO: send a bulk unsub
	// key is the conn  key - a hash of the real name
	// v is the real name, a unicode string.
	for k, v := range c.realTopicNames {
		//topic := types.HashType{}
		//topic.FromHashType(&k)
		c.subcribeMgr.SendUnsubscribeMessage(&k, c)
		connLogThing.Collect("Unsub  topic " + k.String())
		_ = v
	}
	connLogThing.Collect("CLOSE Connection")
	allConnMutex.Lock()
	delete(allTheConnections, c.key)
	allConnMutex.Unlock()
	c.tcpConn.Close()
}

// func watchForData(c *Connection) {
// 	for {
// 		msg := <-c.writesChannel
// 		handler := *c.protocolHandler
// 		err := handler.HandleWrite(msg)
// 		if err != nil {
// 			connLogThing.Collect("wFD err" + err.Error())
// 			c.running = false // not the right way really
// 			return
// 		}
// 	}
// }

var connectionsReporter = func(seconds float32) []string {
	strlist := make([]string, 0, 5)
	allConnMutex.RLock()
	size := len(allTheConnections)
	allConnMutex.RUnlock()
	strlist = append(strlist, "Conn count="+strconv.Itoa(size))
	return strlist
}

var connLogThing *types.StringEventAccumulator

func init() {
	connLogThing = types.NewStringEventAccumulator(12)
	connLogThing.SetQuiet(true)
	types.NewGenericEventAccumulator(connectionsReporter)

}
