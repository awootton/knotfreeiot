package knotfree

import (
	"knotfree/knotfree/types"
	"math/rand"
	"net"
	"strconv"
	"sync"
	"time"
)

// TODO: get all of protocolAa out of here.

// for every TCP socket there will be a Connection struct

// Connection - wait
type Connection struct {
	Key types.HashType // 128 bits
	//Subscriptions map[types.HashType]bool
	Running       bool
	writesChannel chan *types.IncomingMessage // channel for receiving
	tcpConn       *net.TCPConn
	// map from Topic hash to Topic string
	realTopicNames map[types.HashType]string

	protocolHandler *ProtocolHandler
}

// ProtocolHandler for handling read and write
type ProtocolHandler interface {
	Serve() error
	HandleWrite(*types.IncomingMessage) error // from writesChannel
}

// QueueMessageToConnection function. needs to access allTheConnections and the Connection.writesChannel
func QueueMessageToConnection(channelID *types.HashType, message *types.IncomingMessage) bool {

	//fmt.Println("QueueMessageToConnection with " + string(message.Message))
	connLogThing.Collect("qlook4 CONN " + channelID.String())
	allConnMutex.Lock()
	c, ok := allTheConnections[*channelID]
	allConnMutex.Unlock()
	if ok == false {
		return ok
	}
	c.writesChannel <- message

	return true
}

// This is called from two gr's - on purpose.
//
func close(c *Connection) {
	// unsubscrbe
	for k, v := range c.realTopicNames {
		topic := types.HashType{}
		topic.FromHashType(&k)
		SendUnsubscribeMessage(&topic, &c.Key)
		connLogThing.Collect("Unsub  topic " + k.String())
		_ = v
	}
	connLogThing.Collect("deleting CONN " + c.Key.String())
	allConnMutex.Lock()
	delete(allTheConnections, c.Key)
	allConnMutex.Unlock()
	c.tcpConn.Close()
}

func watchForData(c *Connection) {
	defer close(c)
	//fmt.Println("starting watchForData ")
	for {
		msg := <-c.writesChannel
		connLogThing.Collect("watchForD got:" + string(*msg.Message))
		connLogThing.Sum("Conn w bytes", len(*msg.Message))
		err := WriteProtocolAaStr(c.tcpConn, string(*msg.Message))
		if err != nil {
			// log err
			return
		}
	}
}

// RunAConnection - this is really a protoA connection.
//
func RunAConnection(c *Connection) {

	handler := ProtocolAaServerHandler{}
	handler.c = c

	defer close(c)
	c.Running = true
	// random connection id
	randomStr := strconv.FormatInt(rand.Int63(), 16) + strconv.FormatInt(rand.Int63(), 16)
	c.Key.FromString(randomStr)
	c.writesChannel = make(chan *types.IncomingMessage, 2)
	c.realTopicNames = make(map[types.HashType]string)
	//c.Subscriptions = make(map[types.HashType]bool)
	connLogThing.Collect("setting CONN " + c.Key.String())
	allConnMutex.Lock()
	allTheConnections[c.Key] = c
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
	for c.Running {

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
