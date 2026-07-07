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

const serviceDebugSession1 = false

// SessionKeyString is the one and only key used for p.SetOption(SessionKeyString,val) and p.GetOption(SessionKeyString) for the sessionKey option.
// It is a constant that is used to identify the sessionKey option in packets.
// I never again expect to see "sessionKey" used anywhere else in the code. Only on line 18 of this file!
const SessionKeyString = "QyJx_SessionKey" // only ever use this has the key for the sessionKey option.
// I hope there's a static string pool and these compare instantly.

// This is a constant that is used to as the sessionKey value in packets.
func GetSessionValue() [28]byte {
	return GetRandomStringOf28()
}

const version = "0.0.3"

// ServiceContact is a client for sending messages up into the cluster and the return the value back to the caller

type StartNewServiceContactFuncType func(*ServiceContact, error)

type ServiceContact struct {

	// The client is the client that is used to send the message to the cluster
	contact *ContactStruct

	ex *Executive

	// this is 0our return address
	mySubscriptionName string

	// a map of which call to SendPacket sent the message
	key2channel     map[[28]byte]chan packets.Interface
	key2channelLock sync.Mutex

	oldKeys map[[28]byte]packets.Interface
	// same lock as key2channelLock

	packetsChan          chan packets.Interface
	serviceContactClosed chan bool
	IsDebg               bool
	myWriter             *myWriterType

	setter StartNewServiceContactFuncType

	startTime    time.Time
	timeoutCount int
	index        int // the index in the array of service-contacts. This is just for debugging and logging purposes.
}

// StartAndInitServiceContact creates a Contact that is able to send and receive packets.
// Starts listening for packets on the pipe. It's supposed to stay alive forever but it has
// an amazing ability to die in mysterious ways. So when it dies it calls the setter function that was passed in, and the setter function is supposed to create a new ServiceContact and assign it to the executive so we can replace it when it dies. This is a bit of a hack but it's better than crashing the whole process when it dies.
// then it assigns the contact to the caller so we can replace it when it dies.
// I supress the log at executive startup because it will be very noisy in the tests. But if you want to see the log, set supressLogPlease to false.
func StartAndInitServiceContact(ex *Executive, setter StartNewServiceContactFuncType, index int, supressLogPlease bool) {
	sc := &ServiceContact{}
	sc.setter = setter
	sc.ex = ex
	sc.startTime = time.Now()
	sc.timeoutCount = 0
	sc.index = index
	err := initNewServiceContact(sc, supressLogPlease)
	sc.setter(sc, err) // assign it to the executive so we can replace it when it dies.
}

// localStartNewServiceContact is something to put brekpoints on.
// this happens on some kind of fail. What?
func localStartNewServiceContact(ex *Executive, setter StartNewServiceContactFuncType, index int) {
	fmt.Println("localStartNewServiceContact called with index #", index)
	StartAndInitServiceContact(ex, setter, index, false)
}

// Get is a blocking call that sends a message to the cluster and waits for the reply.
// it blocks waiting for an answer. Has a smaller timeout than SendPacket
// This is an example of client code.
func (sc *ServiceContact) GetPacketReply(msg packets.Interface) (packets.Interface, error) {

	return sc.GetPacketReplyLonger(msg, time.Duration(17*time.Second))
}

func (sc *ServiceContact) GetPacketReplyLonger(msg packets.Interface, timeout time.Duration) (packets.Interface, error) {
	returnChannel := make(chan packets.Interface)
	doneChannelForGet := make(chan bool)
	// this terminates when we close done.
	// it might close done if error

	var waitGroup sync.WaitGroup
	waitGroup.Add(1)

	go sc.SendPacket(msg, returnChannel, doneChannelForGet, func() {
		waitGroup.Done()
	})
	waitGroup.Wait() // wait for the SendPacket to start waiting before we start the timeout. Otherwise we might timeout before it even gets there.

	// don't start waiting until this thing has got the packet up to the cluster. Otherwise we might timeout before it even gets there.

	// and then start waiting.
	select {
	case <-doneChannelForGet:
		return nil, fmt.Errorf("ServiceContact failed prematurely")
	case packet := <-returnChannel:

		// this is basically the answer.
		if serviceDebugSession1 {
			fmt.Println("ServiceContact got reply #", sc.index, " packet", packet.Sig(), " for message ", sc.GetMessageText(msg))
		}
		close(doneChannelForGet)
		// reset on sucess so we don't start a new ServiceContact after a timeout if the next call is successful.
		sc.timeoutCount = 0
		sc.startTime = time.Now()
		return packet, nil

	case <-time.After(timeout):
		// why are we here when we just got the answer?

		// how many of these before we start a new ServiceContact? I don't know.
		// we can get these when the DB is slow.
		sc.timeoutCount++
		// how do we even know the ServiceContact is the problem?
		// bail if 3 in 17 sec. Totally arbitrary.
		// delta := time.Since(sc.startTime)
		// if sc.timeoutCount > 3 && delta < time.Duration(17*time.Second) {
		// 	fmt.Println("ServiceContact timed out waiting for reply too many times. Starting new ServiceContact.")
		// 	StartNewServiceContact(sc.ex, sc.setter)
		// }

		// or, screw it. Something smells.
		// ALWAYS start a new ServiceContact on timeout. We can always reset the timeout count and start time on success, so we don't have to worry about false positives.
		// fmt.Println("ServiceContact timed out waiting for reply. Starting new ServiceContact. #", sc.index, " timeout count ", sc.timeoutCount)
		// StartNewServiceContact(sc.ex, sc.setter, sc.index)
		// who writes this kind of shitty code anyway? Someday I'll know what's causing yhese timeouts and I can fix it and remove this hack. But for now, this is better than crashing the whole process when it happens.

		// do this at the other timeout case.

		close(doneChannelForGet) // stop waiting here
		select {
		case <-sc.serviceContactClosed: // already closed. do nothing.
			break
		default:
			fmt.Println("ServiceContact timed out. Starting new ServiceContact. #", sc.index, " v", version)
			// close(sc.serviceContactClosed) // will kill the entire ServiceContact and start a new one.
		}
		// This is a bit of a hack.
		waitingForMessage := sc.GetMessageText(msg)
		return nil, fmt.Errorf("ServiceContact timed out in GetPacketReplyLonger # %d after %v v%s. Waiting for message: %s", sc.index, timeout, version, waitingForMessage)
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
// This is antique shit that used to be used by the old sub-domain-server
// must die
func (sc *ServiceContact) XXXXXXxxMustDieGetPacketGroupReplyLonger(msg packets.Interface, timeout time.Duration) ([]packets.Interface, error) {
	results := make([]packets.Interface, 0, 1)
	returnChannel := make(chan packets.Interface)
	doneChannelForGet := make(chan bool)
	// this terminates when we close done.
	// it might close done if error
	var waitGroup sync.WaitGroup
	waitGroup.Add(1)
	go sc.SendPacket(msg, returnChannel, doneChannelForGet, func() {
		waitGroup.Done()
	})
	waitGroup.Wait() // wait for the SendPacket to start waiting before we start the timeout. Otherwise we might timeout before it even gets there.

	for {
		select {
		case <-doneChannelForGet:
			return nil, fmt.Errorf("ServiceContact failed prematurely")
		case packet := <-returnChannel:
			results = append(results, packet)
			_, ok := packet.GetOption("ind")
			if !ok {
				close(doneChannelForGet)
				return results, nil
			}
			// todo: make sure placement in array matches 'ind'
			_, ok = packet.GetOption("of")
			// TODO: make sure size of array marches 'of/
			if ok {
				close(doneChannelForGet)
				return results, nil
			}

		case <-time.After(timeout):
			close(doneChannelForGet)
			return nil, fmt.Errorf("ServiceContact MustDieGetPacketGroupReplyLonger timed out after %v v %v", timeout, version)
		}
	}
}

// GetMessageText gets the text of of a message sent. For debugging
func (sc *ServiceContact) GetMessageText(msg packets.Interface) string {

	messageText := []byte{}
	switch v := msg.(type) {
	case *packets.Send:
		messageText = v.Payload
	case *packets.Lookup:
		messageText, _ = v.GetOption("cmd")
		messageText = append([]byte("lookup:"), messageText...)
	default:
		messageText = []byte("this packet:" + msg.String())
	}
	return string(messageText)
}

// this is the entry point. It sends a message to the cluster and waits for the reply.
// the reply will go into the returnChannel
// caller should select on the returnChannel and timeout if needed. See Get() above.
func (sc *ServiceContact) SendPacket(msg packets.Interface, returnChannel chan packets.Interface, doneChannelForGet chan bool, callback func()) {

	sessionValue := GetSessionValue() // this is the key that will be used to match the reply to the request. It is a random string of 28 bytes. It is used as the key in the key2channel map.
	// fmt.Println("ServiceContact SendPacket ", msg.Sig(), " with sessionKey ", sessionKey, " index ", sc.index)
	// case  on the type of msg and set the sessionKey and reply address
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
		fmt.Printf("ERROR I don't know about type %T!\n", v)
		close(doneChannelForGet)
		return
	}
	// fmt.Println("ServiceContact SendPacket ", msg.Sig())
	sc.key2channelLock.Lock()
	sc.key2channel[sessionValue] = returnChannel
	sc.key2channelLock.Unlock()
	defer func() {
		sc.key2channelLock.Lock()
		// fmt.Println("ServiceContact removing sessionValue, " index ", sc.index)
		delete(sc.key2channel, sessionValue)
		sc.oldKeys[sessionValue] = msg // save the original message in case we get a response on this key after the caller has already returned.
		// Looking for bugs. Need permethrin on my code.
		sc.key2channelLock.Unlock()
	}()

	// how long does this take?
	// the clock should not start until we get back from doing this.
	err := PushPacketUpFromBottom(sc.contact, msg)

	callback() // tell the caller that we're done sending the packet and they can start waiting for the reply.

	if err != nil {
		fmt.Println("ServiceContact SendPacket PushPacketUpFromBottom failed ", err)
		return
	}
	timeout := time.Duration(5 * time.Second) // why wait so long? 5 * time.Second) If it's broken it's broken.
	if DEBUG {
		// timeout = time.Duration(999 * time.Second)
	}
	// we wait 5 whole seconds and then burn the whole house down.
	{ // The Receive-a-packet loop from returnChannel. caller must close chan done to exit.
		for {
			select {
			case <-doneChannelForGet:
				return
			case <-sc.contact.ClosedChannel:
				fmt.Println("ServiceContact error CONTACT closed. This is bad. Why? Who closed my contact? index=#", sc.index)
				// we have to start over
				// localStartNewServiceContact(sc.ex, sc.setter, sc.index)
				select {
				case <-sc.serviceContactClosed:
					// already closed. do nothing.
					break
				default:
					fmt.Println("ServiceContact error CONTACT closed. Starting new ServiceContact. index=#", sc.index)
					close(sc.serviceContactClosed)
					break
				}
			case <-sc.serviceContactClosed: // we'll do this on i/o errors and such. Means we're dead.
				fmt.Println("ServiceContact closed. This is bad. Why? #", sc.index)
				localStartNewServiceContact(sc.ex, sc.setter, sc.index)
				fmt.Println("ServiceContact closed. Starting new ServiceContact. #", sc.index)
				return
			case <-time.After(timeout):
				errMsg := fmt.Sprintf("ServiceContact , Receive-a-packet loop, timed out # %d v%s", sc.index, version)
				//localStartNewServiceContact(sc.ex, sc.setter, sc.index)
				fmt.Println(errMsg)
				return
			}
		}
	}
}

// initNewServiceContact- a Contact that is able to send and receive packets.
// Starts listening for packets on the pipe.
func initNewServiceContact(sc *ServiceContact, supressLogPlease bool) error {

	if !supressLogPlease {
		fmt.Println("ServiceContact initNewServiceContact #", sc.index)
	}

	sc.key2channel = make(map[[28]byte]chan packets.Interface)
	sc.oldKeys = make(map[[28]byte]packets.Interface)
	sc.mySubscriptionName = GetRandomB64String()
	sc.serviceContactClosed = make(chan bool)

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

	// Allocate a strict 64k memory buffer
	buf := buffer.New(2 * 64 * 1024)
	// Returns a buffered pipe reader and writer
	// how can this not be perfect? The buffer will never grow beyond 64k, and the pipe will never block because the buffer is big enough to hold a whole packet. And we can read and write in parallel without blocking each other. It's perfect for our use case.??
	myWriter.myPipeReader, myWriter.myPipeWriter = nio.Pipe(buf)

	//  this was the old one.
	// myWriter.myPipeReader, myWriter.myPipeWriter = io.Pipe() // this is the pipe that the packets will come in on

	// don't we want to start this after the writers are set up? I think so.

	// sc.startReadTheWriterPipe() // reads packets and puts them on the packetsChan forever.

	contact.SetWriter(myWriter) // myWriter)
	// I'm going to need a much bigger buffer for this to run an api
	AddContactStructSized(contact, contact, sc.ex.Config, 1024*64)

	sc.startReadTheWriterPipe() // reads packets and puts them on the packetsChan forever.

	connect := packets.Connect{}
	token := tokens.GetImpromptuGiantToken()
	connect.SetOption("token", []byte(token))
	err := PushPacketUpFromBottom(contact, &connect)
	_ = err

	// subscribe to the mySubscriptionName
	subs := packets.Subscribe{}
	subs.Address.FromString(sc.mySubscriptionName)
	subs.Address.EnsureAddressIsBinary()
	err = PushPacketUpFromBottom(contact, &subs)
	_ = err

	mySubscriptionNameBinary := subs.Address

	// now we have to wait for the suback to come back
	// Don't do anything else. The aide may not be connected ?
	if !supressLogPlease {
		fmt.Println("ServiceContact #", sc.index, " waiting for suback.")
	}
	haveSuback := false
	for !haveSuback {
		select {
		case <-contact.ClosedChannel:
			haveSuback = true
		case packet := <-packetsChan:
			// see if it's a suback
			// fmt.Println("waiting for suback on gotDataChan.TheChan got ", cmd.Sig())
			if packet == nil {
				fmt.Println("ServiceContact ERROR nil packet waiting for suback. #", sc.index, "Never happens.")
			} else {
				subcmd, ok := packet.(*packets.Subscribe)
				_ = subcmd
				if !ok {
					fmt.Println("ServiceContact ERROR wrong packet waiting for suback #", sc.index, " ")
				} else {
					// if isDebg {
					if !supressLogPlease {
						fmt.Println("ServiceContact have suback #", sc.index, " sig=", subcmd.Sig())
					}
					// is this OUR suback?
					subcmd.Address.EnsureAddressIsBinary()
					if subcmd.Address.String() != mySubscriptionNameBinary.String() { // that would be weird.
						fmt.Println("ServiceContact ERROR suback for wrong subscription #", sc.index, " expected ", mySubscriptionNameBinary.String(), " got ", subcmd.Address.String())
						// fail now. Start over.
						close(sc.serviceContactClosed)
						select {
						case <-sc.serviceContactClosed:
							// already closed. do nothing.
						default:
							close(sc.serviceContactClosed)
						}
					}
					haveSuback = true
				}
			}
			// we have to wait for the suback to come back, and not for 23 sec: atw 7/2/26
		case <-time.After(3 * time.Second):
			errMsg := fmt.Sprintf("ServiceContact timed out waiting for suback reply # %d v%s", sc.index, version)
			fmt.Println(errMsg)
			// and then we crash and panic and die because sc.closed is already closed.
			// close(sc.closed) no, silly, just close the serviceContactClosed channel and let the other goroutines handle the rest.
			select {
			case <-sc.serviceContactClosed:
				// already closed. do nothing.
			default:
				close(sc.serviceContactClosed)
			}
			return fmt.Errorf(errMsg)
		}
	}

	if !supressLogPlease {
		fmt.Println("ServiceContact started #", sc.index, " with subscription name ", sc.mySubscriptionName, " v", version)
	}

	// to keep the contact alive by resubscribing every 10 minutes.
	go func() {
		for {
			select {
			case <-sc.serviceContactClosed:
				return
			case <-time.After(10 * time.Minute): // I don't know what to say.
			}
			// fmt.Println("ServiceContact keep alive #", sc.index)
			subs := packets.Subscribe{}
			subs.Address.FromString(sc.mySubscriptionName)
			subs.Address.EnsureAddressIsBinary()
			err := PushPacketUpFromBottom(contact, &subs)
			_ = err
		}
	}()

	// pull packets from the packetsChan and send them to the key2channel
	// forever. If someone closes it, we're done.
	go func() {
		for {
			select {
			case <-sc.serviceContactClosed:
				// this one
				// fmt.Println("ServiceContact #", sc.index, " closed. we're dead as a doornail. Starting new ServiceContact")
				sc.contact.DoClose(fmt.Errorf("ServiceContact closed"))
				// we have to start over
				localStartNewServiceContact(sc.ex, sc.setter, sc.index)
				fmt.Println("ServiceContact #", sc.index, " closed. Started new ServiceContact. v", version)
				return // we're dead as a doornail
			case p := <-packetsChan:
				{
					sessionValueFoundSlice, got := p.GetOption(SessionKeyString)
					if !got {
						// this happens. why, again? Very weird. We MUST get rid of these. It's gaslighting me.
						_, ok := p.(*packets.Subscribe)
						if ok {
							// nobody cares about the sub acks. fmt.Println("ServiceContact got suback ERROR no sessionKey in packet ", subPacket.Sig())
							continue
						}
						// these are a problem.
						// let's get the whole packet and see if we can figure out what it is.
						what := p.String() // or sig() ?
						fmt.Println("ServiceContact ERROR no session value in packet #", sc.index, " ", what, " v", version)
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
					destChan, ok := sc.key2channel[sessionValue]
					sc.key2channelLock.Unlock()
					if !ok {
						// the theory is that a response on this key already happened and is screwing every thing up.
						// Let's just see about that. Who would do such a henious Thing?
						oldResponse, ok2 := sc.oldKeys[sessionValue]
						fmt.Println("ServiceContact ERROR no channel for key ", string(sessionValue[:]), " in packet #", sc.index, " ", p.Sig(), " v", version)
						if ok2 {
							fmt.Println("ServiceContact ERROR but we DID have an OLD response for key ", string(sessionValue[:]), " was: ", oldResponse.String(), " in #", sc.index, " v", version)
							fmt.Println("ServiceContact ERROR compare to  ", p.String(), " was: ", oldResponse.String(), " in #", sc.index, " v", version)
						}
						if len(sc.oldKeys) > 8192 {
							sc.oldKeys = make(map[[28]byte]packets.Interface) // clear it out if it gets too big. This is a hack to avoid memory leaks.
						}
						// also, the nameServices echo back the command. Let's see it.
						_, ok3 := p.(*packets.Send)
						if ok3 {
							cmd, ok := p.GetOption("cmd")
							if ok {
								fmt.Println("ServiceContact ERROR but we DID have a Send packet for key ", string(sessionValue[:]), " in #", sc.index, " cmd=", string(cmd), " v", version)
							}
						} else {
							fmt.Println("ServiceContact ERROR I expected a send packet for key ", string(sessionValue[:]), " in #", sc.index, " but got ", p.String(), " v", version)
						}
						_, ok4 := p.(*packets.Send)
						if !ok4 {
							fmt.Println("ServiceContact ERROR do they normally get send or lookup? ", p.String(), " in #", sc.index, " ", p.String(), " v", version)
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

// startReadTheWriterPipe will read packets off the stream and put them in the channel. Simple.
func (sc *ServiceContact) startReadTheWriterPipe() {
	go func() {
		for {
			select {
			case <-sc.contact.ClosedChannel:
				fmt.Println("ServiceContact handler contact closed #", sc.index, " v", version)
				// localStartNewServiceContact(sc.ex, sc.setter, sc.index)
				fmt.Println("ServiceContact handler contact closed #", sc.index, " v", version)
				return
			default:

				packet, err := packets.ReadPacket(sc.myWriter)
				if err != nil || packet == nil {
					// the buffer only had a partial packet. Why? Why? Why? Why? Why?
					// I suspect that two people are writing to the socket at the same time and the buffer is getting confused. no?
					// But that's not supposed to happen because EVERYTHING is supposed to go in a channel first.
					// and then there's just ONE reader of the socket (this one).
					sc.contact.DoClose(err)

					// let the DoClose do this
					// localStartNewServiceContact(sc.ex, sc.setter, sc.index)
					fmt.Println("ServiceContact ERROR packet read fail (too few packets for Send probably) #", sc.index, " ", err, " v", version)

					return
				}
				if sc.IsDebg {
					fmt.Println("ServiceContact got packet #", sc.index, " ", packet.String(), " v", version)
				}
				sc.packetsChan <- packet
			}
		}
	}()
}

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
