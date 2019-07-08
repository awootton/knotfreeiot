package knotfree

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"proj1/knotfree/subscriptionmgr"
	"proj1/knotfree/types"

	"strconv"
	"time"
)

// TODO: move this out to ConnectionMgr

var allTheConnections = make(map[types.HashType]*Connection)

// xxFindConnection - jkust look it up
func xxFindConnection(key *types.HashType) (*Connection, bool) {
	c, ok := allTheConnections[*key]
	return c, ok
}

// Qmessage s
func (*Connection) Qmessage(channelID *types.HashType, message *types.IncomingMessage) bool {

	c, ok := allTheConnections[*channelID]
	if ok == false {
		return ok
	}
	c.wchan <- message

	return true
}

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

// This is called from two gr's - on purpose.
func close(c *Connection) {
	// unsubscrbe
	for k, v := range c.Subscriptions {
		unsub := subscriptionmgr.UnsubscribeMessage{}
		unsub.ConnectionID.FromHashType(&c.Key)
		unsub.Channel.FromHashType(&k)
		subscriptionmgr.AddUnsubscribe(unsub)
		_ = v
	}

	delete(allTheConnections, c.Key)
	c.conn.Close()
}

func watchForData(c *Connection) {
	defer close(c)
	for {
		msg := <-c.wchan
		fmt.Println("watchForData sending " + string(msg.Message))
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
	allTheConnections[c.Key] = c
	// start reading
	err := c.conn.SetReadBuffer(4096)
	if err != nil {
		fmt.Println("server err " + err.Error())
		return
	}
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
				// log
				return
			}
			// ok, so what is the message? subscribe or publish?
			//fmt.Println("Have Messaage " + str)
			// eg sAchannel
			if str[0] == 's' {
				// process subscribe
				subChan := str[1:]
				_ = subChan
				fmt.Println("Have subChan " + subChan)
				// we'll fill in a sub request and 'mail' it to the sub handler
				// SubscriptionMessage struct  {
				// 	Channel      HashType
				// 	//ChannelName  string
				// 	ConnectionID HashType

				subr := subscriptionmgr.SubscriptionMessage{}
				subr.Channel.FromString(subChan)
				subr.ConnectionID.FromHashType(&c.Key)
				c.realChannelNames[subr.Channel] = subChan
				subscriptionmgr.AddSubscription(subr)
			} else if str[0] == 'p' {
				// process publish, {"C":"channelRealName","M":"a message"}
				fmt.Println("got p publish " + str)
				pub := PublishProtocolA{}
				err := json.Unmarshal([]byte(str[1:]), &pub)
				if err != nil {
					fmt.Println("server json " + err.Error())
					return
				}

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

		// if &c != nil {
		// 	continue
		// }

		// err = conn.SetReadBuffer(4096)
		// if err != nil {
		// 	fmt.Println("server err " + err.Error())
		// 	conn.Close()
		// 	break
		// }
		// err = conn.SetWriteBuffer(4096)
		// if err != nil {
		// 	fmt.Println("server err " + err.Error())
		// 	conn.Close()
		// 	break
		// }

		// go func(c net.Conn) {

		// 	for {
		// 		err = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		// 		if err != nil {
		// 			fmt.Println("server err " + err.Error())
		// 			conn.Close()
		// 			break
		// 		}
		// 		fmt.Println("writing")
		// 		n, err := c.Write([]byte("Server" + time.Now().String()))

		// 		if err != nil {
		// 			fmt.Println("server write err " + err.Error())
		// 			c.Close()
		// 			break
		// 		}
		// 		_ = n
		// 		fmt.Println("server wrote " + strconv.Itoa(n))
		// 		time.Sleep(22 * time.Minute)
		// 	}

		// }(conn)

		// go func(c net.Conn) {
		// 	bytes := make([]byte, 1024)
		// 	for {
		// 		err = conn.SetReadDeadline(time.Now().Add(20 * time.Minute))
		// 		if err != nil {
		// 			fmt.Println("server err " + err.Error())
		// 			conn.Close()
		// 			break
		// 		}
		// 		n, err := conn.Read(bytes)
		// 		_ = n
		// 		//time.Sleep(1000 * time.Millisecond)
		// 		if err != nil {
		// 			fmt.Println("server dropped conn")
		// 			conn.Close()
		// 			break // and we're done
		// 		}
		// 		fmt.Println("server got:" + string(bytes))
		// 	}
		// }(conn)
	}
}
