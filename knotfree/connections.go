package knotfree

import (
	"encoding/json"
	"knotfree/knotfree/types"
	"math/rand"
	"net"
	"strconv"
	"sync"
	"time"
)

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

// for every TCP socket there will be a Connection struct

// This is a Set of all the Connection structs that can be looked up by Key.
var allTheConnections = make(map[types.HashType]*Connection)
var allConnMutex = &sync.Mutex{}

// Connection - wait
type Connection struct {
	Key           types.HashType // 128 bits
	Subscriptions map[types.HashType]bool
	Running       bool
	writesChannel chan *types.IncomingMessage // channel for receiving
	tcpConn       *net.TCPConn

	realChannelNames map[types.HashType]string
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
	for k, v := range c.Subscriptions {
		unsub := UnsubscribeMessage{}
		unsub.ConnectionID.FromHashType(&c.Key)
		unsub.Topic.FromHashType(&k)
		AddUnsubscribe(unsub)
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
		connLogThing.Collect("watchForD got:" + string(msg.Message))
		connLogThing.Sum("Conn w bytes", len(msg.Message))
		err := WriteProtocolA(c.tcpConn, string(msg.Message))
		if err != nil {
			// log err
			return
		}
	}
}

// RunAConnection - this is really a protoA cpnnection.
//
func RunAConnection(c *Connection) {
	defer close(c)
	c.Running = true
	// random connection id
	c.Key.FromString(strconv.FormatInt(rand.Int63(), 16) + strconv.FormatInt(rand.Int63(), 16))
	c.writesChannel = make(chan *types.IncomingMessage, 2)
	c.realChannelNames = make(map[types.HashType]string)
	c.Subscriptions = make(map[types.HashType]bool)
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
		bytes := make([]byte, 256)
		for {
			err := c.tcpConn.SetReadDeadline(time.Now().Add(20 * time.Minute))
			if err != nil {
				connLogThing.Collect("server err2 " + err.Error())
				return
			}
			str, err := ReadProtocolA(c.tcpConn, bytes)
			if err != nil {
				connLogThing.Collect("rProtocolA err " + str + err.Error())
				return
			}
			connLogThing.Sum("Conn r bytes", len(str))
			// ok, so what is the message? subscribe or publish?
			//fmt.Println("Have Server str _a " + str)
			// eg sAchannel
			if str[0] == 's' {
				// process subscribe
				subTopic := str[1:]

				//fmt.Println("sub CONN key " + c.Key.String())
				//fmt.Println("Have subTopic " + subTopic)
				// we'll fill in a sub request and 'mail' it to the sub handler
				// TODO: change to proc call
				subr := SubscriptionMessage{}
				subr.Topic.FromString(subTopic)
				//fmt.Println("Have subChan becomes " + subr.Channel.String())
				subr.ConnectionID.FromHashType(&c.Key)
				//fmt.Println("subscribe ConnectionID is " + subr.ConnectionID.String())
				c.realChannelNames[subr.Topic] = subTopic
				AddSubscription(subr)
				c.Subscriptions[subr.Topic] = true
			} else if str[0] == 'p' {
				//fmt.Println("publish CONN key " + c.Key.String())
				// process publish, {"C":"channelRealName","M":"a message"}
				//fmt.Println("got p publish " + str)
				pub := PublishProtocolA{}
				err := json.Unmarshal([]byte(str[1:]), &pub)
				if err != nil {
					connLogThing.Collect("server json " + err.Error())
					return
				}
				//fmt.Println("Have PublishProtocolA " + string(&pub))
				pubr := PublishMessage{}
				pubr.Topic.FromString(pub.T)
				pubr.ConnectionID.FromHashType(&c.Key)
				pubr.Message = []byte(pub.M)
				//fmt.Println("Publish topic becomes " + pubr.Topic.String())
				//fmt.Println("Publish ConnectionID is " + pubr.ConnectionID.String())
				AddPublish(pubr)
			}
			// we don't have an unsubscribe yet.
		}
	}
}
