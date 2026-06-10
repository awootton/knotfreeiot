package iot

import (
	"fmt"
	"sync"

	"time"

	"github.com/awootton/knotfreeiot/packets"
	"github.com/awootton/knotfreeiot/tokens"

	"github.com/djherbis/buffer"
	"github.com/djherbis/nio/v3"
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

// ServiceContact is a client for sending messages up into the cluster and the return the value back to the caller

type ServiceContact struct {

	// The client is the client that is used to send the message to the cluster
	contact *ContactStruct

	ex *Executive

	// this is 0our return address
	mySubscriptionName string

	// a map of which call to SendPacket sent the message
	key2channel     map[string]chan packets.Interface
	key2channelLock sync.Mutex

	packetsChan chan packets.Interface
	closed      chan bool
	IsDebg      bool
	myWriter    *myWriterType
}

// Get is a blocking call that sends a message to the cluster and waits for the reply.
// it blocks waiting for an answer. Has a smaller timeout than SendPacket
// This is an example of client code.
func (sc *ServiceContact) GetPacketReply(msg packets.Interface) (packets.Interface, error) {

	return sc.GetPacketReplyLonger(msg, time.Duration(5*time.Second))
}

func (sc *ServiceContact) GetPacketReplyLonger(msg packets.Interface, timeout time.Duration) (packets.Interface, error) {
	returnChannel := make(chan packets.Interface)
	done := make(chan bool)
	// this terminates when we close done.
	// it might close done if error
	go sc.SendPacket(msg, returnChannel, done)

	select {
	case <-done:
		return nil, fmt.Errorf("ServiceContact failed prematurely")
	case packet := <-returnChannel:
		close(done)
		return packet, nil
	case <-time.After(timeout):
		close(done)
		return nil, fmt.Errorf("ServiceContact timed out waiting for reply")
	}
}

// Get is a blocking call that sends a message to the cluster and waits for the reply.
// it blocks waiting for an answer. Has a smaller timeout than SendPacket
// This is an example of client code.
// func (sc *ServiceContactTcp) Get(msg packets.Interface) (packets.Interface, error) {
// 	returnChannel := make(chan packets.Interface)
// 	done := make(chan bool)
// 	// this termnates when we close done.
// 	// it might close done if error
// 	go sc.SendPacket(msg, returnChannel, done)

// 	select {
// 	case <-done:
// 		return nil, fmt.Errorf("ServiceContact failed prematurely")
// 	case packet := <-returnChannel:
// 		close(done)
// 		return packet, nil
// 	case <-time.After(2 * time.Second):
// 		close(done)
// 		return nil, fmt.Errorf("ServiceContact timed out waiting for reply 2 sec")
// 	}
// }

// GetPacketGroupReplyLonger watches for an 'ind' or index key and
// collects packets until an 'of' key says we're done
// TODO:we have no tests for this.
func (sc *ServiceContact) GetPacketGroupReplyLonger(msg packets.Interface, timeout time.Duration) ([]packets.Interface, error) {
	results := make([]packets.Interface, 0, 1)
	returnChannel := make(chan packets.Interface)
	done := make(chan bool)
	// this terminates when we close done.
	// it might close done if error
	go sc.SendPacket(msg, returnChannel, done)

	for {
		select {
		case <-done:
			return nil, fmt.Errorf("ServiceContact failed prematurely")
		case packet := <-returnChannel:
			results = append(results, packet)
			_, ok := packet.GetOption("ind")
			if !ok {
				close(done)
				return results, nil
			}
			// todo: make sure placement in array matches 'ind'
			_, ok = packet.GetOption("of")
			// TODO: make sure size of array marches 'of/
			if ok {
				close(done)
				return results, nil
			}

		case <-time.After(timeout):
			close(done)
			return nil, fmt.Errorf("ServiceContact timed out waiting for reply")
		}
	}
}

// this is the entry point. It sends a message to the cluster and waits for the reply.
// the reply will go into the returnChannel
// caller should select on the returnChannel and timeout if needed. See Get() above.
func (sc *ServiceContact) SendPacket(msg packets.Interface, returnChannel chan packets.Interface, done chan bool) {

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

	// fmt.Println("ServiceContact SendPacket ", msg.Sig())

	sc.key2channelLock.Lock()
	sc.key2channel[key] = returnChannel
	sc.key2channelLock.Unlock()
	defer func() {
		sc.key2channelLock.Lock()
		delete(sc.key2channel, key)
		sc.key2channelLock.Unlock()
	}()

	err := PushPacketUpFromBottom(sc.contact, msg)
	if err != nil {
		fmt.Println("ServiceContact SendPacket PushPacketUpFromBottom failed ", err)
		return
	}
	timeout := time.Duration(5 * time.Second)
	if DEBUG {
		timeout = time.Duration(999 * time.Second)
	}
	{ // The Receive-a-packet loop from returnChannel. caller must close chan done to exit.
		for {
			select {
			case <-done:
				return
			case <-sc.contact.ClosedChannel: // Does this happen?
				fmt.Println("error seviceContact contact closed. This is bad")
				// we have to start over
				InitNewServiceContact(sc) // leaks?
				return
			// 	close(sc.closed)
			case <-sc.closed:
				fmt.Println("seviceContact closed. This is bad")
				return
			case <-time.After(timeout): // sooner than nginx
				errMsg := "SendPacket timed out waiting for reply (receiver offline)"
				fmt.Println(errMsg)
				return
			}
		}
	}
}

// StartNewServiceContact returns a Contact that is able to send and receive packets.
// Starts listening for packets on the pipe.
func StartNewServiceContact(ex *Executive) (*ServiceContact, error) {
	sc := &ServiceContact{}
	sc.ex = ex
	return sc, InitNewServiceContact(sc)
}

// InitNewServiceContact- a Contact that is able to send and receive packets.
// Starts listening for packets on the pipe.
func InitNewServiceContact(sc *ServiceContact) error {

	sc.key2channel = make(map[string]chan packets.Interface)
	sc.mySubscriptionName = GetRandomB64String()
	sc.closed = make(chan bool)

	packetsChan := make(chan packets.Interface, 100)
	contact := &ContactStruct{}
	contact.LogMeVerbose = true // todo: remove later
	sc.contact = contact
	// hook the real writer
	myWriter := &myWriterType{}
	myWriter.packets = packetsChan
	sc.packetsChan = packetsChan
	sc.myWriter = myWriter
	contact.contactExpires += 60 * 60 * 24 * 365 * 10 // in 10 years

	// Allocate a strict 128KB memory buffer
	buf := buffer.New(64 * 1024)
	// Returns a buffered pipe reader and writer
	myWriter.myPipeReader, myWriter.myPipeWriter = nio.Pipe(buf)
	// myWriter.myPipeReader, myWriter.myPipeWriter = io.Pipe() // this is the pipe that the packets will come in on

	sc.startReadTheWriterPipe() // reads packets and puts them on the packetsChan forever.

	contact.SetWriter(myWriter) // myWriter)
	// I'm going to need a much bigger buffer for this to run an api
	AddContactStructSized(contact, contact, sc.ex.Config, 1024*64)

	connect := packets.Connect{}
	connect.SetOption("token", []byte(tokens.GetImpromptuGiantToken()))
	err := PushPacketUpFromBottom(contact, &connect)
	_ = err

	// subscribe to the mySubscriptionName
	subs := packets.Subscribe{}
	subs.Address.FromString(sc.mySubscriptionName)
	subs.Address.EnsureAddressIsBinary()
	err = PushPacketUpFromBottom(contact, &subs)
	_ = err

	// now we have to wait for the suback to come back
	fmt.Println("ServiceContact waiting for suback.")
	haveSuback := false
	for !haveSuback {
		select {
		case <-contact.ClosedChannel:
			haveSuback = true
		case packet := <-packetsChan:
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
					fmt.Println("http handler have suback  ", subcmd.Sig())
					// }
					haveSuback = true
				}
			}
			// we have to wait for the suback to come back
		case <-time.After(20 * time.Second):
			errMsg := "timed out waiting for suback reply "
			fmt.Println(errMsg)
			// and then we crash and panic and die because sc.closed is already closed.
			// close(sc.closed)
			select {
			case <-sc.closed:
				// already closed. do nothing.
			default:
				close(sc.closed)
			}
			return fmt.Errorf(errMsg)
		}
	}

	fmt.Println("ServiceContact started.")

	// to keep the contact alive by resubscribing every 10 minutes.
	go func() {
		for {
			select {
			case <-sc.closed:
				return
			case <-time.After(10 * 60 * time.Second):
			}
			fmt.Println("ServiceContact keep alive.")
			subs := packets.Subscribe{}
			subs.Address.FromString(sc.mySubscriptionName)
			subs.Address.EnsureAddressIsBinary()
			err := PushPacketUpFromBottom(contact, &subs)
			_ = err
		}
	}()

	// pull packets from the packetsChan and send them to the key2channel
	// forever
	go func() {
		for {
			select {
			case <-sc.closed:
				fmt.Println("ServiceContact closed. we're dead as a doornail")
				InitNewServiceContact(sc)
				return // we're dead as a doornail
			case p := <-packetsChan:
				{
					sessionKey, got := p.GetOption("sessionKey")
					if !got {
						// this happens. fmt.Println("ERROR no sessionKey in packet ", p.Sig())
						continue
					}
					sc.key2channelLock.Lock()
					destChan, ok := sc.key2channel[string(sessionKey)]
					sc.key2channelLock.Unlock()
					if !ok {
						// fmt.Println("ERROR no channel for key ", string(sessionKey))
					} else {
						destChan <- p
					}
				}
			}
		}
	}()

	return nil
}

func (sc *ServiceContact) startReadTheWriterPipe() {
	go func() {
		for {
			select {
			case <-sc.contact.ClosedChannel:
				fmt.Println(" handler contact closed")
				InitNewServiceContact(sc)
				return
			default:

				packet, err := packets.ReadPacket(sc.myWriter)
				if err != nil || packet == nil {
					// the buffer only had a partial packet. Why?
					fmt.Println("ERROR packet read fail ", err)
					sc.contact.DoClose(err)
					// can we start over? wtf.
					InitNewServiceContact(sc)
					return
				}
				if sc.IsDebg {
					fmt.Println("http subdomain handler got packet ", packet.String())
				}
				sc.packetsChan <- packet
			}
		}
	}()
}
