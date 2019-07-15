// Package iot manages Connections, literal tcp system sockets, using an object called Connection in the file
// connections.go
package iot

import (
	"knotfree/types"
	"net"
	"strconv"
	"sync"
)

// for every TCP socket there will be a Connection struct

// Connection - wait
// we don't need this to be public
type Connection struct {
	key types.HashType // 128 bits
	//Subscriptions map[types.HashType]bool
	running       bool
	writesChannel chan *types.IncomingMessage // channel for receiving
	tcpConn       *net.TCPConn
	// map from Topic hash to Topic string
	realTopicNames map[types.HashType]string

	protocolHandler *types.ProtocolHandler
}

// SetProtocolHandler is just a setter
func (c *Connection) SetProtocolHandler(protocolHandler *types.ProtocolHandler) {
	c.protocolHandler = protocolHandler
}

// GetTCPConn is spelled correctly
func (c *Connection) GetTCPConn() *net.TCPConn {
	return c.tcpConn
}

// SetRealTopicName is
func (c *Connection) SetRealTopicName(h *types.HashType, s string) {
	c.realTopicNames[*h] = s
}

// GetRealTopicName is
func (c *Connection) GetRealTopicName(h *types.HashType) (string, bool) {
	str, ok := c.realTopicNames[*h]
	return str, ok
}

// GetKey is
func (c *Connection) GetKey() *types.HashType {
	return &c.key
}

// QueueMessageToConnection called by subscribe. needs to access allTheConnections and the Connection.writesChannel
func QueueMessageToConnection(channelID *types.HashType, message *types.IncomingMessage) bool {

	//fmt.Println("QueueMessageToConnection with " + string(message.Message))
	connLogThing.Collect("qlook4 CONN " + channelID.String())
	allConnMutex.Lock()
	c, ok := allTheConnections[*channelID]
	allConnMutex.Unlock()
	if ok == false {
		return ok
	}
	// FIXME: just call the handler directly. Watch for blocking.
	c.writesChannel <- message

	return true
}

// Close will kill the connection for disconnect or timeout or error or whatever.
func (c *Connection) Close() {
	// unsubscrbe
	for k, v := range c.realTopicNames {
		topic := types.HashType{}
		topic.FromHashType(&k)
		GetSubscriptionsMgr().SendUnsubscribeMessage(&topic, &c.key)
		connLogThing.Collect("Unsub  topic " + k.String())
		_ = v
	}
	connLogThing.Collect("deleting CONN " + c.key.String())
	allConnMutex.Lock()
	delete(allTheConnections, c.key)
	allConnMutex.Unlock()
	c.tcpConn.Close()
}

func watchForData(c *Connection) {
	defer c.Close()
	//fmt.Println("starting watchForData ")
	for {
		msg := <-c.writesChannel
		handler := *c.protocolHandler
		err := handler.HandleWrite(msg)
		if err != nil {
			c.Close()
		}
	}
}

// This is a Set of all the Connection structs that can be looked up by Key.
var allTheConnections = make(map[types.HashType]*Connection)
var allConnMutex = &sync.Mutex{}

type connectionsEventsReporter struct {
}

func (collector *connectionsEventsReporter) report(seconds float32) []string {
	strlist := make([]string, 0, 5)
	allConnMutex.Lock()
	size := len(allTheConnections)
	allConnMutex.Unlock()
	strlist = append(strlist, "Conn count="+strconv.Itoa(size))
	return strlist
}

var connLogThing *StringEventAccumulator

func init() {
	connLogThing = NewStringEventAccumulator(12)
	connLogThing.quiet = true
	AddReporter(&connectionsEventsReporter{})
}
