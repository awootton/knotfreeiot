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

// connectObj - wait
// we don't need this to be public
type connectObj struct {
	key types.HashType // 128 bits
	//Subscriptions map[types.HashType]bool
	running bool // do we need this?
	//writesChannel chan *types.IncomingMessage // channel for receiving
	tcpConn *net.TCPConn
	// map from Topic hash to Topic string
	realTopicNames map[types.HashType]string

	// todo: these will all be the same so it's a waste
	protocolHandler types.ProtocolHandlerIntf
	subcribeMgr     types.SubscriptionsIntf
}

// NewConnection is used by Server.go
// the new struct is NOT inserted into the global list.
func NewConnection(tcpConn *net.TCPConn, subscribeMgr types.SubscriptionsIntf) types.ConnectionIntf {

	c := connectObj{}
	c.tcpConn = tcpConn
	c.subcribeMgr = subscribeMgr
	c.running = true // do we need this?
	// random connection id
	randomStr := strconv.FormatInt(rand.Int63(), 16) + strconv.FormatInt(rand.Int63(), 16)
	c.key.FromString(randomStr)
	//c.writesChannel = make(chan *types.IncomingMessage, 2)
	c.realTopicNames = make(map[types.HashType]string)

	return &c
}

// This is a Set of all the Connection structs that can be looked up by Key.
var allTheConnections = make(map[types.HashType]types.ConnectionIntf)
var allConnMutex = &sync.RWMutex{}

// InitAllTheConnectionsGlobal is something I'm only using during initialization
// func InitAllTheConnectionsGlobal(size int) {
// 	ResetAllTheConnectionsMap()
// 	allConnMutex.Lock()
// 	allTheConnections = make(map[types.HashType]types.ConnectionIntf, size)
// 	allConnMutex.Unlock()
// }

// ResetAllTheConnectionsMap is something I'm only using during initialization.
// Clears them all out.
func ResetAllTheConnectionsMap(size int) {
	allConnMutex.Lock()
	allTheConnections = make(map[types.HashType]types.ConnectionIntf, size)
	allConnMutex.Unlock()
}

// GetAllConnectionsSize  is
func GetAllConnectionsSize() int {
	allConnMutex.Lock()
	n := len(allTheConnections)
	allConnMutex.Unlock()
	return n
}

// SetProtocolHandler is just a setter
func (c *connectObj) SetProtocolHandler(protocolHandler types.ProtocolHandlerIntf) {
	c.protocolHandler = protocolHandler
}

// GetProtocolHandler is just a getter
func (c *connectObj) GetProtocolHandler() types.ProtocolHandlerIntf {
	return c.protocolHandler
}

// GetTCPConn is spelled correctly
func (c *connectObj) GetTCPConn() *net.TCPConn {
	return c.tcpConn
}

// SetTCPConn is spelled correctly
func (c *connectObj) SetTCPConn(t *net.TCPConn) {
	c.tcpConn = t
}

// SetRealTopicName is used by someone who shouldn't?
func (c *connectObj) SetRealTopicName(h *types.HashType, s string) {
	c.realTopicNames[*h] = s
}

// GetRealTopicName is used ...?
func (c *connectObj) GetRealTopicName(h *types.HashType) (string, bool) {
	str, ok := c.realTopicNames[*h]
	return str, ok
}

// GetKey is a getter
func (c *connectObj) GetKey() *types.HashType {
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
func RememberConnection(c types.ConnectionIntf) {
	allConnMutex.Lock()
	allTheConnections[*c.GetKey()] = c
	allConnMutex.Unlock()
}

// QueueMessageToConnection called by subscribe. needs to access allTheConnections and the Connection.writesChannel
func QueueMessageToConnection(channelID *types.HashType, message *types.IncomingMessage) {

	connLogThing.Collect("QueueMessageToConnection") // + channelID.String())
	allConnMutex.RLock()
	c, ok := allTheConnections[*channelID]
	allConnMutex.RUnlock()
	if ok == false {
		// someone is sending a message to a lost channel
		// c will be nil
		connLogThing.CollectOnce("someone is sending a message to a lost channel")
		return
	}
	handler := c.GetProtocolHandler()
	//connLogThing.Collect("HandleWrite")
	err := handler.HandleWrite(message)
	if err != nil {
		// only timeout errors or socket errors will happen
		// and they will stuff an error into the pipe that the
		// poll (in server.runTheConnection) will break on and that will close the socket

		// c.running = false FIXME: can we do without this please?
	}
}

// Close will kill the connection for disconnect or timeout or error or whatever.
func (c *connectObj) Close() {
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
	if c.tcpConn != nil {
		c.tcpConn.Close()
	}
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
