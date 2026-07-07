package iot

import (
	"fmt"
	"net"
	"sync"

	"time"

	"github.com/awootton/knotfreeiot/packets"
)

// Copyright 2024-2026 Alan Tracey Wootton
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

// Can we make this one reproduce and replace myself at the owner?
// Do we pass in a 'setter StartNewServiceContactFuncType' ?? TODO: (atw) make this share most of the code with ServiceContact
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
	key2channel     map[[28]byte]chan packets.Interface
	key2channelLock sync.Mutex

	oldKeys map[[28]byte]packets.Interface
	// same lock as key2channelLock

	packetsChan chan packets.Interface
	closed      chan bool
	IsDebg      bool
	// myWriter    *myWriterType
	index int // for debugging. This is the index of this ServiceContactTcp in the list of ServiceContactTcp's that the owner doesn't have. ha ha.
}

// StartNewServiceClient creates a new ServiceContact and returns it.
// And assigns it to someone so if it croaks we can just give them a new one.

// give this one the same setter as the ServiceContact so we can replace it when it dies. This is a bit of code duplication but it's only a few lines and it's easier to read than trying to merge the two together. We can refactor later if we want to.
// Starts listening for packets on the pipe.
func StartNewServiceContactTcp(address string, token string) (*ServiceContactTcp, error) {
	sc := &ServiceContactTcp{}
	sc.Host = address
	sc.token = token
	return sc, InitNewServiceContactTcp(sc)
}

// Get is a blocking call that sends a message to the cluster and waits for the reply.
// it blocks waiting for an answer. Has a smaller timeout than SendPacket
// This is an example of client code.
func (sc *ServiceContactTcp) Get(msg packets.Interface) (packets.Interface, error) {
	returnChannel := make(chan packets.Interface) // this is where the actual answer comes through.
	doneChannelForGet := make(chan bool)          // pushed or popped. Just closed.
	// this termnates when we close done.
	// it might close done if error
	var waitGroup sync.WaitGroup
	waitGroup.Add(1)

	go sc.SendPacket(msg, returnChannel, doneChannelForGet,
		func() {
			waitGroup.Done()
		})
	waitGroup.Wait() // wait for the SendPacket to start waiting before we start the timeout. Otherwise we might timeout before it even gets there.

	select {
	case <-doneChannelForGet:
		return nil, fmt.Errorf("service-contact_tcp failed prematurely")
	case packet := <-returnChannel:
		close(doneChannelForGet)
		return packet, nil // normal ending.
	case <-time.After(5 * time.Second):
		close(doneChannelForGet)
		return nil, fmt.Errorf("service-contact_tcp timed out waiting for reply 5 sec")
	}
}

// this is the entry point. It sends a message to the cluster and waits for the reply.
// the reply will go into the returnChannel
// caller should select on the returnChannel and timeout if needed. See Get() above.
func (sc *ServiceContactTcp) SendPacket(msg packets.Interface, returnChannel chan packets.Interface, doneChannelForGet chan bool, callback func()) {

	sessionValue := GetSessionValue()
	// fmt.Println("service-contact_tcp sessionValue ", sessionValue)

	// case  on the type of msg and set the session value and reply address
	switch v := msg.(type) {
	case *packets.Send:
		v.SetOption(SessionKeyString, sessionValue[:])
		v.Source.FromString(sc.mySubscriptionName)
		CheckSendPacket(v)
	case *packets.Lookup:
		v.SetOption(SessionKeyString, sessionValue[:])
		v.Source.FromString(sc.mySubscriptionName)
		CheckLookupPacket(v)

	default:
		fmt.Printf("ERROR service-contact_tcp I don't know about type %T!\n", v)
		close(doneChannelForGet)
		return
	}

	// fmt.Println("ServiceContact_tcp send packet ", msg.Sig())

	sc.key2channelLock.Lock()
	sc.key2channel[sessionValue] = returnChannel
	sc.key2channelLock.Unlock()
	defer func() {
		sc.key2channelLock.Lock()
		delete(sc.key2channel, sessionValue)
		sc.oldKeys[sessionValue] = msg // save the original message in case we get a response on this key after the caller has already returned.
		sc.key2channelLock.Unlock()
	}()

	sc.outgoing <- msg

	// tell the caller that we're not waiting to get the reply sent.
	callback() // tell the caller that we're done sending the packet and they can start waiting for the reply.

	{ // The Receive-a-packet loop from returnChannel. caller must close chan done to exit.
		for {
			select {
			case <-doneChannelForGet:
				// normal ending. we got the reply, the session was looked up and the callback called and then
				// this was closed and then we can remove the key/value from the map. This is the normal ending.
				return
			// case <-sc.contact.ClosedChannel: Does this happen?
			// 	fmt.Println("seviceContact contact closed. This is bad")
			// 	close(sc.closed)
			case <-sc.closed:
				fmt.Println("service-contact_tcp closed. This is bad")
				return
			case <-time.After(7 * time.Second): // sooner than nginx? What was that about?
				errMsg := "SendPacket timed out waiting for reply (receiver offline)"
				fmt.Println(errMsg)
				return
			}
		}
	}
}

// StartNewServiceClient creates a new ServiceContact and returns it.
// Starts listening for packets on the pipe.
func InitNewServiceContactTcp(sc *ServiceContactTcp) error {

	sc.key2channel = make(map[[28]byte]chan packets.Interface)
	sc.oldKeys = make(map[[28]byte]packets.Interface)

	sc.mySubscriptionName = GetRandomB64String()
	println("ServiceContact_tcp mySubscriptionName ", sc.mySubscriptionName)
	// sc.ex = ex
	sc.closed = make(chan bool)

	// would it hurt if we made these huge? Superstitiously huge?
	sc.packetsChan = make(chan packets.Interface, 256*256)
	sc.outgoing = make(chan packets.Interface, 256*256)

	sc.ConnectLoopForever()

	// subscribe to the mySubscriptionName
	subs := packets.Subscribe{}
	subs.Address.FromString(sc.mySubscriptionName)
	subs.Address.EnsureAddressIsBinary()
	sc.outgoing <- &subs
	println("ServiceContact_tcp sent subscribe for ", sc.mySubscriptionName)

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
		case <-time.After(7 * time.Second):
			errMsg := "timed out waiting for suback reply 7 sec."
			fmt.Println(errMsg)
			close(sc.closed)
			return fmt.Errorf(errMsg)
		}
	}

	// to keep the contact alive by resubscribing every 10 minutes.
	go func() {
		for {
			select {
			case <-sc.closed:
				return
			case <-time.After(10 * time.Minute):
			}
			subs := packets.Subscribe{}
			subs.Address.FromString(sc.mySubscriptionName)
			subs.Address.EnsureAddressIsBinary()
			println("service-contact_tcp resubscribing ", sc.mySubscriptionName)
			sc.outgoing <- &subs
		}
	}()

	// pull packets from the packetsChan and send them to the key2channel
	// forever
	go func() {
		for {
			select {
			case <-sc.closed:
				InitNewServiceContactTcp(sc) // start over?
				return
			case p := <-sc.packetsChan:
				{
					sessionValueFoundSlice, got := p.GetOption(SessionKeyString)
					if !got {
						_, ok := p.(*packets.Subscribe)
						if ok {
							// nobody cares about the sub acks. fmt.Println("ServiceContact got suback ERROR no sessionKey in packet ", subPacket.Sig())
							continue
						}
						// these are a problem.
						// let's get the whole packet and see if we can figure out what it is.
						what := p.String() // or sig() ?
						fmt.Println("ServiceContact ERROR no session value in packet #", sc.index, " ", what)
						// weird. How does a status: "topic not found errid=bvBbhJawYXIMWsxJOWHt"
						// get back here without a session value? I don't know. But it does.
						// It seems like a simple thing. running test TestDialTCP_one_at_a_time
						continue

					}
					sessionValue := [28]byte{}
					// FIXME: atw, is there a better way? No, copilot, it's not a hack that 'just works'.
					copied := copy(sessionValue[:], sessionValueFoundSlice)
					if copied != 28 {
						fmt.Println("ServiceContact ERROR sessionValue copy length is not 28. Is today 10/31? 4/1? 4/20? It's ", copied)
					}
					// one thing that happens is that the aide sends back a send before or instead of forwarding to the guru
					// where it can be properly handled by the lookmsg and the nameservices.
					// We can TELL because the sendReply of the lookmsg is stripped of the "cmd" option. So, if we get a send with a "cmd" option, we can just throw it away and not get confused.
					isGoodReply := false
					_, ok := p.(*packets.Send)
					if ok {
						cmd, ok := p.GetOption("processed_by_lookup")
						if ok && string(cmd) == "1" {
							// we're good.
							isGoodReply = true
						}
					}
					if !isGoodReply {
						_, ok := p.(*packets.Send)
						if ok {
							cmd, ok := p.GetOption("cmd")
							if ok {
								fmt.Println("ServiceContact got a send with a cmd option. This is an echo from aide. Throwing it away. cmd=", string(cmd), " in packet #", sc.index, " ", p.Sig())
								continue
							}
						}
					}

					sc.key2channelLock.Lock()
					var key [28]byte
					copy(key[:], sessionValue[:])
					destChan, ok := sc.key2channel[key]
					sc.key2channelLock.Unlock()
					if !ok {
						// the theory is that a response on this key already happened and is screwing every thing up.
						// Let's just see about that. Who would do such a henious Thing?
						oldResponse, ok2 := sc.oldKeys[sessionValue]
						fmt.Println("ServiceContact ERROR no channel for key ", string(sessionValue[:]), " in packet #", sc.index, " ", p.Sig())
						if ok2 {
							fmt.Println("ServiceContact ERROR but we DID have an OLD response for key ", string(sessionValue[:]), " was: ", oldResponse.String(), " in #", sc.index)
							fmt.Println("ServiceContact ERROR compare to  ", p.String(), " was: ", oldResponse.String(), " in #", sc.index)
						}
						if len(sc.oldKeys) > 8192 {
							sc.oldKeys = make(map[[28]byte]packets.Interface) // clear it out if it gets too big. This is a hack to avoid memory leaks.
						}
						// also, the nameServices echo back the command. Let's see it.
						_, ok3 := p.(*packets.Send)
						if ok3 {
							cmd, ok := p.GetOption("cmd")
							if ok {
								fmt.Println("ServiceContact ERROR but we DID have a Send packet for key ", string(sessionValue[:]), " in #", sc.index, " cmd=", string(cmd))
							}
						} else {
							fmt.Println("ServiceContact ERROR I expected a send packet for key ", string(sessionValue[:]), " in #", sc.index, " but got ", p.String())
						}
						_, ok4 := p.(*packets.Send)
						if !ok4 {
							fmt.Println("ServiceContact ERROR do they normally get send or lookup? ", p.String(), " in #", sc.index, " ", p.String())
						}
					} else {
						// fmt.Println("ServiceContact HAVE  channel for key ", string(sessionValue[:]), " in packet #", sc.index, " ", p.Sig())
						destChan <- p
					}
				}
			}
		}
	}()

	return nil
}

func (sc *ServiceContactTcp) ConnectLoopForever() {

	go func() {

		connectCount := 0

		for { // connect loop forever

			servAddr := sc.Host // target_cluster + ":8384"
			tcpAddr, err := net.ResolveTCPAddr("tcp", servAddr)
			if err != nil {
				println("had ResolveTCPAddr failed:", err.Error())
				sc.fail++
				time.Sleep(2 * time.Second)
				continue // to connect loop
			}
			println("ConnectLoopForever Dialing ", tcpAddr, " attempt ", connectCount)
			sc.conn, err = net.DialTCP("tcp", nil, tcpAddr)
			if err != nil {
				println("dial failed:", err.Error())
				time.Sleep(2 * time.Second)
				sc.fail++
				continue // to connect loop forever
			}
			connect := &packets.Connect{}
			connect.SetOption("token", []byte(sc.token))
			// if c.LogMeVerbose {
			// 	connect.SetOption("debg", []byte("12345678"))
			// }
			err = connect.Write(sc.conn)
			if err != nil {
				println("write connect to server failed:", err.Error())
				sc.conn.Close()
				time.Sleep(12 * time.Second)
				sc.fail++
				continue // to connect loop
			}

			fmt.Println("service-contact_tcp connected and waiting..")

			// subscribe to the mySubscriptionName
			subs := packets.Subscribe{}
			subs.Address.FromString(sc.mySubscriptionName)
			subs.Address.EnsureAddressIsBinary()
			sc.outgoing <- &subs
			println("service-contact_tcp connect loop forever sent subscribe for ", sc.mySubscriptionName)

			isBroken := make(chan interface{})

			go func() {
				for {
					select {
					case <-sc.closed:
						close(isBroken)
						return
					case <-isBroken:
						println("service-contact_tcp isBroken:")
						return
					case p := <-sc.outgoing:
						println("service-contact_tcp write packet from outgoing:", p.Sig())
						err := p.Write(sc.conn)
						if err != nil {
							println("service-contact_tcp write C to server failed:", err.Error())
							close(isBroken)
						}
					}
				}
			}()

			done := false
			for !done { // read cmd loop
				select {
				case <-isBroken:
					println("read cmd loop isBroken")
					done = true //break from read loop, not select
				default:
				}
				if done {
					break
				}
				err = sc.conn.SetDeadline(time.Now().Add(30 * time.Minute))
				if err != nil {
					fmt.Println("deadline err 5", err, sc.conn.RemoteAddr())
					time.Sleep(2 * time.Second)
					sc.fail++
					done = true
					break // from read loop
				}
				p, err := packets.ReadPacket(sc.conn) // blocks
				if err != nil {
					println("ReadPacket client err:", err.Error())
					sc.conn.Close()
					sc.fail++
					time.Sleep(2 * time.Second) // try again in 2 seconds
					done = true                 // break from read loop
					break                       // from read loop
				}
				println("service-contact_tcp ReadPacket packet:", p.Sig())

				sc.packetsChan <- p
				sc.count++
			} // read loop
			connectCount++
		} // connect loop
	}()

}
