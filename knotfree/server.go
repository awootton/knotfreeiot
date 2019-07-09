package knotfree

import (
	"encoding/json"
	"fmt"

	"knotfree/knotfree/types"
	"math/rand"
	"net"

	"strconv"
	"time"
)

// TODO: move this out to ConnectionMgr

var allTheConnections = make(map[types.HashType]*Connection)

// Connection - wait
type Connection struct {
	Key types.HashType // 128 bits
	//Subscriptions []string `json:"subs,omitempty"`
	Subscriptions map[types.HashType]bool
	Running       bool
	wchan         chan *types.IncomingMessage // channel for receiving
	conn          *net.TCPConn

	realChannelNames map[types.HashType]string
}

// QueueMessageToConnection function. needs to access allTheConnections and Connection wchan
func QueueMessageToConnection(channelID *types.HashType, message *types.IncomingMessage) bool {

	//fmt.Println("QueueMessageToConnection with " + string(message.Message))
	//fmt.Println("looking for CONN " + channelID.String())
	c, ok := allTheConnections[*channelID]
	if ok == false {
		return ok
	}
	c.wchan <- message

	return true
}

// This is called from two gr's - on purpose.
func close(c *Connection) {
	// unsubscrbe
	for k, v := range c.Subscriptions {
		unsub := UnsubscribeMessage{}
		unsub.ConnectionID.FromHashType(&c.Key)
		unsub.Channel.FromHashType(&k)
		AddUnsubscribe(unsub)
		_ = v
	}

	fmt.Println("deleting CONN " + c.Key.String())
	delete(allTheConnections, c.Key)
	c.conn.Close()
}

func watchForData(c *Connection) {
	defer close(c)
	//fmt.Println("starting watchForData ")
	for {
		msg := <-c.wchan
		//fmt.Println("watchForData " + string(msg.Message))
		err := WriteProtocolA(c.conn, string(msg.Message))
		if err != nil {
			// log err
			return
		}
	}
}

func run(c *Connection) {
	defer close(c)
	c.Running = true
	// random connection id
	c.Key.FromString(strconv.FormatInt(rand.Int63(), 16) + strconv.FormatInt(rand.Int63(), 16))
	c.wchan = make(chan *types.IncomingMessage, 2)
	c.realChannelNames = make(map[types.HashType]string)
	fmt.Println("setting CONN " + c.Key.String())
	allTheConnections[c.Key] = c
	// start reading
	err := c.conn.SetReadBuffer(4096)
	if err != nil {
		fmt.Println("server err " + err.Error())
		return
	}

	go watchForData(c)
	// bytes, _ := json.Marshal(c)
	// fmt.Println("connection struct " + string(bytes))
	for c.Running {
		bytes := make([]byte, 256)
		for {
			err := c.conn.SetReadDeadline(time.Now().Add(20 * time.Minute))
			if err != nil {
				fmt.Println("server err " + err.Error())
				return
			}
			str, err := ReadProtocolA(c.conn, bytes)
			if err != nil {
				fmt.Println("ReadProtocolA err " + str + err.Error())
				return
			}
			// ok, so what is the message? subscribe or publish?
			//fmt.Println("Have Server str _a " + str)
			// eg sAchannel
			if str[0] == 's' {
				// process subscribe
				subChan := str[1:]
				_ = subChan
				//fmt.Println("sub CONN key " + c.Key.String())
				//fmt.Println("Have subChan " + subChan)
				// we'll fill in a sub request and 'mail' it to the sub handler
				// TODO: change to proc call
				subr := SubscriptionMessage{}
				subr.Channel.FromString(subChan)
				//fmt.Println("Have subChan becomes " + subr.Channel.String())
				subr.ConnectionID.FromHashType(&c.Key)
				//fmt.Println("subscribe ConnectionID is " + subr.ConnectionID.String())
				c.realChannelNames[subr.Channel] = subChan
				AddSubscription(subr)
			} else if str[0] == 'p' {
				//fmt.Println("publish CONN key " + c.Key.String())
				// process publish, {"C":"channelRealName","M":"a message"}
				//fmt.Println("got p publish " + str)
				pub := PublishProtocolA{}
				err := json.Unmarshal([]byte(str[1:]), &pub)
				if err != nil {
					fmt.Println("server json " + err.Error())
					return
				}
				//fmt.Println("Have PublishProtocolA " + string(&pub))
				pubr := PublishMessage{}
				pubr.Channel.FromString(pub.C)
				pubr.ConnectionID.FromHashType(&c.Key)
				pubr.Message = []byte(pub.M)
				//fmt.Println("Publish channel becomes " + pubr.Channel.String())
				//fmt.Println("Publish ConnectionID is " + pubr.ConnectionID.String())
				AddPublish(pubr)
			}
		}
	}
}

// Server - wait for connections and spawn them
func Server() {
	ln, err := net.Listen("tcp", ":8080")
	if err != nil {
		// handle error
		fmt.Println(err.Error())
		return
	}
	for {
		tmpconn, err := ln.Accept()
		if err != nil {
			fmt.Println(err.Error())
			continue
		}
		c := Connection{conn: tmpconn.(*net.TCPConn)}
		go run(&c)

	}
}
