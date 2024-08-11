package iot

import (
	"fmt"
	"net"
	"sync"

	"time"

	"github.com/awootton/knotfreeiot/packets"
)

// Copyright 2024 Alan Tracey Wootton
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

// This is similar to the ServiceContact struct in iot/service-contact.go
// except it's over TCP instead of a pipe.
// TODO: make this share most of the code with ServiceContact

type ServiceContactTcp struct {

	// The client is the client that is used to send the message to the cluster
	// contact *ContactStruct

	//  ex *Executive

	Host  string
	token string

	conn     *net.TCPConn
	outgoing chan packets.Interface

	fail  int
	count int

	// this is our return address
	mySubscriptionName string

	// a map of which call to SendPacket sent the message
	key2channel     map[string]chan packets.Interface
	key2channelLock sync.Mutex

	packetsChan chan packets.Interface
	closed      chan bool
	IsDebg      bool
	// myWriter    *myWriterType
}

// Get is a blocking call that sends a message to the cluster and waits for the reply.
// it blocks waiting for an answer. Has a smaller timeout than SendPacket
// This is an example of client code.
func (sc *ServiceContactTcp) Get(msg packets.Interface) (packets.Interface, error) {
	returnChannel := make(chan packets.Interface)
	done := make(chan bool)
	// this termnates when we close done.
	// it might close done if error
	go sc.SendPacket(msg, returnChannel, done)

	select {
	case <-done:
		return nil, fmt.Errorf("ServiceContact failed prematurely")
	case packet := <-returnChannel:
		close(done)
		return packet, nil
	case <-time.After(2 * time.Second):
		close(done)
		return nil, fmt.Errorf("ServiceContact timed out waiting for reply")
	}
}

// this is the entry point. It sends a message to the cluster and waits for the reply.
// the reply will go into the returnChannel
// caller should select on the returnChannel and timeout if needed. See Get() above.
func (sc *ServiceContactTcp) SendPacket(msg packets.Interface, returnChannel chan packets.Interface, done chan bool) {

	key := GetRandomB64String()
	// case  on the type of msg and set the sessionKey and reply address
	switch v := msg.(type) {
	case *packets.Send:
		v.SetOption("sessionKey", []byte(key))
		v.Source.FromString(sc.mySubscriptionName)
	case *packets.Lookup:
		v.SetOption("sessionKey", []byte(key))
		v.Source.FromString(sc.mySubscriptionName)
	default:
		fmt.Printf("ERROR I don't know about type %T!\n", v)
		close(done)
		return
	}

	fmt.Println("ServiceContact SendPacket ", msg.Sig())

	sc.key2channelLock.Lock()
	sc.key2channel[key] = returnChannel
	sc.key2channelLock.Unlock()
	defer func() {
		sc.key2channelLock.Lock()
		delete(sc.key2channel, key)
		sc.key2channelLock.Unlock()
	}()

	sc.outgoing <- msg

	// err := PushPacketUpFromBottom(sc.contact, msg)
	// if err != nil {
	// 	fmt.Println("ServiceContact SendPacket PushPacketUpFromBottom failed ", err)
	// 	return
	// }

	{ // The Receive-a-packet loop from returnChannel. caller must close chan done to exit.
		for {
			select {
			case <-done:
				return
			// case <-sc.contact.ClosedChannel: Does this happen?
			// 	fmt.Println("seviceContact contact closed. This is bad")
			// 	close(sc.closed)
			case <-sc.closed:
				fmt.Println("seviceContact closed. This is bad")
				return
			case <-time.After(4321 * time.Millisecond): // sooner than nginx
				errMsg := "SendPacket timed out waiting for reply (receiver offline)"
				fmt.Println(errMsg)
				return
			}
		}
	}
}

// StartNewServiceClient creates a new ServiceContact and returns it.
// Starts listening for packets on the pipe.
func StartNewServiceContactTcp(address string, token string) (*ServiceContactTcp, error) {

	sc := &ServiceContactTcp{}
	sc.Host = address
	sc.token = token

	sc.key2channel = make(map[string]chan packets.Interface)
	sc.mySubscriptionName = GetRandomB64String()
	// sc.ex = ex
	sc.closed = make(chan bool)

	sc.packetsChan = make(chan packets.Interface, 100)
	sc.outgoing = make(chan packets.Interface, 100)

	// contact := &ContactStruct{}
	// sc.contact = contact
	// hook the real writer
	// myWriter := &myWriterType{}
	// myWriter.packets = packetsChan
	//sc.packetsChan = packetsChan
	// sc.myWriter = myWriter
	//contact.contactExpires += 60 * 60 * 24 * 365 * 10 // in 10 years

	// myWriter.myPipeReader, myWriter.myPipeWriter = io.Pipe() // this is the pipe that the packets will come in on

	// sc.startReadTheWriterPipe() // reads packets and puts them on the packetsChan forever.

	sc.ConnectLoopForever()

	// contact.SetWriter(myWriter) // myWriter)
	// AddContactStruct(contact, contact, ex.Config)

	// connect := packets.Connect{}
	// connect.SetOption("token", []byte(tokens.GetImpromptuGiantToken()))
	// err := PushPacketUpFromBottom(contact, &connect)
	// _ = err

	// subscribe to the mySubscriptionName
	subs := packets.Subscribe{}
	subs.Address.FromString(sc.mySubscriptionName)
	subs.Address.EnsureAddressIsBinary()
	sc.outgoing <- &subs
	// err = PushPacketUpFromBottom(contact, &subs)
	// _ = err

	// now we have to wait for the suback to come back
	haveSuback := false
	for !haveSuback {
		select {
		// case <-contact.ClosedChannel:
		// 	haveSuback = true
		case packet := <-sc.packetsChan:
			// see if it's a suback
			// fmt.Println("waiting for suback on gotDataChan.TheChan got ", cmd.Sig())
			if packet == nil {
				fmt.Println("ERROR nil packet waiting for suback. Never happens.")
			} else {
				subcmd, ok := packet.(*packets.Subscribe)
				_ = subcmd
				if !ok {
					fmt.Println("ERROR wrong packet waiting for suback  ")
				} else {
					// if isDebg {
					// 	fmt.Println("http handler have suback  ", subcmd.Sig())
					// }
					haveSuback = true
				}
			}
			// we have to wait for the suback to come back
		case <-time.After(4 * time.Second):
			errMsg := "timed out waiting for suback reply "
			fmt.Println(errMsg)
			close(sc.closed)
			return nil, fmt.Errorf(errMsg)
		}
	}

	// to keep the contact alive by resubscribing every 10 minutes.
	go func() {
		for {
			select {
			case <-sc.closed:
				return
			case <-time.After(10 * 60 * time.Second):
			}
			subs := packets.Subscribe{}
			subs.Address.FromString(sc.mySubscriptionName)
			subs.Address.EnsureAddressIsBinary()
			//err := PushPacketUpFromBottom(contact, &subs)
			//_ = err
			sc.outgoing <- &subs
		}
	}()

	// pull packets from the packetsChan and send them to the key2channel
	// forever
	go func() {
		for {
			select {
			case <-sc.closed:
				return // we're dead as a doornail
			case p := <-sc.packetsChan:
				{
					sessionKey, got := p.GetOption("sessionKey")
					if !got {
						fmt.Println("ERROR no sessionKey in packet ", p.Sig())
						continue
					}
					sc.key2channelLock.Lock()
					destChan, ok := sc.key2channel[string(sessionKey)]
					sc.key2channelLock.Unlock()
					if !ok {
						fmt.Println("ERROR no channel for key ", string(sessionKey))
					} else {
						destChan <- p
					}
				}
			}
		}
	}()

	return sc, nil
}

// func (sc *ServiceContactTcp) startReadTheWriterPipe() {
// 	go func() {
// 		for {
// 			select {
// 			// case <-sc.contact.ClosedChannel:
// 			// 	fmt.Println(" handler contact closed")
// 			// 	return
// 			default:

// 				packet, err := packets.ReadPacket(sc.myWriter)
// 				if err != nil || packet == nil {
// 					// the buffer only had a partial packet
// 					fmt.Println("ERROR packet read fail ", err)
// 					sc.contact.DoClose(err)
// 					return
// 				}
// 				if sc.IsDebg {
// 					fmt.Println("http subdomain handler got packet ", packet.String())
// 				}
// 				sc.packetsChan <- packet
// 			}
// 		}
// 	}()
// }

func (sc *ServiceContactTcp) ConnectLoopForever() {

	go func() {

		connectCount := 0

		for { // connect loop forever

			servAddr := sc.Host // target_cluster + ":8384"
			tcpAddr, err := net.ResolveTCPAddr("tcp", servAddr)
			if err != nil {
				println("had ResolveTCPAddr failed:", err.Error())
				sc.fail++
				time.Sleep(10 * time.Second)
				continue // to connect loop
			}
			println("ConnectLoopForever Dialing ")
			sc.conn, err = net.DialTCP("tcp", nil, tcpAddr)
			if err != nil {
				println("dial failed:", err.Error())
				time.Sleep(10 * time.Second)
				sc.fail++
				continue // to connect loop forevek
			}
			connect := &packets.Connect{}
			connect.SetOption("token", []byte(sc.token))
			// if c.LogMeVerbose {
			// 	connect.SetOption("debg", []byte("12345678"))
			// }
			err = connect.Write(sc.conn)
			if err != nil {
				println("write C to server failed:", err.Error())
				sc.conn.Close()
				time.Sleep(10 * time.Second)
				sc.fail++
				continue // to connect loop
			}

			fmt.Println("connected and waiting..")

			isBroken := make(chan interface{})

			go func() {
				for {
					select {
					case <-sc.closed:
						close(isBroken)
						return
					case <-isBroken:
						return
					case p := <-sc.outgoing:
						err := p.Write(sc.conn)
						if err != nil {
							println("write C to server failed:", err.Error())
							close(isBroken)
						}
					}
				}
			}()

			for { // read cmd loop
				done := false
				select {
				case <-isBroken:
					done = true //break from read loop, not select
				default:
				}
				if done {
					break
				}
				p, err := packets.ReadPacket(sc.conn) // blocks
				if err != nil {
					println("ReadPacket client err:", err.Error())
					sc.conn.Close()
					sc.fail++
					// quitSubscribeLoop <- true
					time.Sleep(10 * time.Second)
					// close(isBroken)
					break // from read loop
				}
				sc.packetsChan <- p
				sc.count++
			} // read loop
			connectCount++
		} // connect loop
	}()

}
