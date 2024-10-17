// Copyright 2019,2020,2021 Alan Tracey Wootton
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

// Package iot comments. TODO: package comments for this pub/sub system.
package iot

import (
	"container/list"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/awootton/knotfreeiot/packets"
	"github.com/awootton/knotfreeiot/tokens"
)

// A 'contact' is an incoming connection.

// ContactCommander is how we acceas a contact. It's a function that takes a *ContactStruct
// We push the function into the 'commands' channel and it gets executed.
type ContactCommander struct {
	who string
	fn  func(ss *ContactStruct)
}

// ContactStruct is our idea of channel or socket which is downstream from us.
// I was trying to keep this small. LOL.
type ContactStruct struct {
	ele *list.Element

	config *ContactStructConfig
	key    HalfHash // something unique. 64 bits.

	// these are the limits, they have been sent to the subscription.
	// there is also the JWTID in the token.
	token *tokens.KnotFreeTokenPayload

	contactExpires uint32

	nextBillingTime uint32
	lastBillingTime uint32 // input and output were cleared at this time.
	input           atomic.Int64
	output          atomic.Int64

	commands      chan ContactCommander
	ClosedChannel chan interface{}
	once          sync.Once

	realReader io.Reader // usually tcpConn
	realWriter io.Writer // usually tcpConn

	LogMeVerbose bool // this just a debug thing.
}

// ContactInterface is usually supplied by a tcp connection
type ContactInterface interface {
	DoClose(err error)       // call this to close the contact
	DoClosingWork(err error) // this will get called only one time, with sole access to the contact

	IsClosed() bool

	GetKey() HalfHash

	WriteCommand(cmd ContactCommander)

	GetExpires() uint32
	// only ever call this from inside a ContactCommander
	SetExpires(when uint32)

	GetToken() *tokens.KnotFreeTokenPayload
	// only ever call this from inside a ContactCommander
	SetToken(*tokens.KnotFreeTokenPayload)

	GetConfig() *ContactStructConfig

	WriteDownstream(cmd packets.Interface) error

	WriteUpstream(cmd packets.Interface) error // called by LookupTableStruct.PushUp

	String() string // used as a default channel name in test

	Heartbeat(uint32) // periodic service ~= 10 sec

	//AddSubscription(sub *packets.Subscribe) trying to deprecate this feature
	//RemoveSubscription(sub *packets.Unsubscribe)

	Read(p []byte) (int, error)
	Write(p []byte) (int, error)

	GetRates(now uint32) (int, int, int) // fixme: have stats call a billingAccumulator on heartbeat.

	SetReader(r io.Reader)
	SetWriter(w io.Writer)

	// this will disconnect the packet to writer loop and return the channel
	// and unless you are reading the new chan packets will pile up.
	// delete meObtainControlOfWriteChannel() chan packets.Interface
}

// ContactStructConfig could be just a stack frame but I'd like to return it.
// This could be an interface that implements range and len or and the callbacks.
// Instead we have function pointers. TODO: revisit.
type ContactStructConfig struct {

	// a linked list of all the *Contacts that are open and not Close'd
	listOfCi *list.List
	listlock *sync.RWMutex // so it's thread safe

	key HashType // everyone needs to feel special and unique

	sequence uint64 // every time we factory init a Contact we increment this and we never decrement.

	lookup *LookupTableStruct // LookupTableInterface

	//  address string // eg knotfreeserver:7009

	Name string // for debug

	defaultTimeoutSeconds uint32 // in seconds

	ce *ClusterExecutive // optional
}

// AccessContactsList so we can disconnect them in test and stuff.
// be sure to always lock. Don't call close or recurse in the fn or it will deadlock.
func (config *ContactStructConfig) AccessContactsList(fn func(config *ContactStructConfig, listOfCi *list.List)) {
	config.listlock.Lock()
	defer config.listlock.Unlock()
	fn(config, config.listOfCi)
}

// GetContactsListCopy copies the list.
func (config *ContactStructConfig) GetContactsListCopy() []ContactInterface {
	contactList := make([]ContactInterface, 0, config.listOfCi.Len())
	// copy out the list of contacts.
	config.AccessContactsList(func(config *ContactStructConfig, listOfCi *list.List) {
		l := listOfCi
		e := l.Front()
		for ; e != nil; e = e.Next() {
			cc := e.Value.(ContactInterface)
			contactList = append(contactList, cc)
		}
	})
	return contactList
}

// GetCe is a getter
func (config *ContactStructConfig) GetCe() *ClusterExecutive {
	return config.ce
}

// IsGuru exposes config.lookup.isGuru
func (config *ContactStructConfig) IsGuru() bool {
	if config == nil {
		return false
	}
	if config.lookup == nil {
		return false
	}
	return config.lookup.isGuru
}

// AddContactStruct initializes a contact, and puts the new ss on the global
// list. It also increments the sequence number in SockStructConfig.
// note that you must pass the same object twice, once as a ContactStruct and once as the Interface
func AddContactStruct(ss *ContactStruct, ssi ContactInterface, config *ContactStructConfig) *ContactStruct {

	size := 16
	if config.IsGuru() {
		size = 1024
	}

	ss.config = config
	ss.commands = make(chan ContactCommander, size)
	ss.ClosedChannel = make(chan interface{})

	config.AccessContactsList(func(config *ContactStructConfig, listOfCi *list.List) {
		if ss.key == 0 {
			seq := config.sequence
			config.sequence++
			ss.key = HalfHash(seq + config.key.GetUint64())
		}
		ss.ele = listOfCi.PushBack(ssi)
	})

	now := config.GetLookup().getTime()
	ss.contactExpires = 20*60 + now // stale contacts expire in 20 min. contact timeout
	// fmt.Println("contactExpires 20 min")

	ss.nextBillingTime = now + 30 // 30 seconds to start with
	ss.lastBillingTime = now

	go func() { // loop until closed read p and write to Write impl

		for {
			var cmd ContactCommander
			select {
			case <-ss.ClosedChannel:
				if ss.LogMeVerbose {
					fmt.Println("writechan error closing now", ss.GetKey().Sig())
				}
				if config.IsGuru() {
					fmt.Println("writechan error closed Guru socket")
				}
				// stop looping, we're done
				return
			case cmd = <-ss.commands:
				//fmt.Println("WriteCommand pop")
				if !ss.IsClosed() {
					// fmt.Println("WriteCommand fn TOP", cmd.who) // for debug FIXME: use debug print or something
					cmd.fn(ss)
					// fmt.Println("WriteCommand fn DONE", cmd.who)
				}
				//  done by some commands --->  p.Write(ss)
			}
		}
	}()

	return ss
}

func (ss *ContactStruct) WriteCommand(cmd ContactCommander) {
	ss.commands <- cmd
}

// NewContactStructConfig is
func NewContactStructConfig(looker *LookupTableStruct) *ContactStructConfig {
	config := ContactStructConfig{}
	config.lookup = looker
	looker.config = &config
	var alock sync.RWMutex
	config.listlock = &alock
	config.listOfCi = list.New()
	config.key.Random()
	config.sequence = 1
	config.defaultTimeoutSeconds = 10
	return &config
}

func PushPacketUpFromBottom(ssi ContactInterface, p packets.Interface) error {
	return PushPacketUpFromBottom2(ssi, p, true)
}

// PushPacketUpFromBottom to deal with an incoming message on a bottom contact heading up.
// it expects a token before anything else.
// the packets are sent up to the looker where they are separated into buckets and dealt with.
func PushPacketUpFromBottom2(ssi ContactInterface, p packets.Interface, doSetExpires bool) error {

	var err error
	var wg sync.WaitGroup
	var config *ContactStructConfig
	var looker *LookupTableStruct
	wg.Add(1)
	ssi.WriteCommand(ContactCommander{
		who: "PushPacketUpFromBottom2",
		fn: func(ss *ContactStruct) {
			defer wg.Done()
			if ssi.IsClosed() {
				// throw the packet away
				err = errors.New("closed contact")
				return
			}
			config = ssi.GetConfig()
			if config == nil {
				fmt.Println("no way there's no config")
				return
			}
			looker = config.GetLookup()

			if doSetExpires {
				ssi.SetExpires(20*60 + config.lookup.getTime())
			}

			err := expectToken(ssi, p)
			if err != nil {
				return
			}
			got, ok := p.GetOption("debg")
			if ok && string(got) == "12345678" {
				fmt.Println("Contact PushPacketUpFromBottom con=", ssi.GetConfig().key.Sig(), " ", p.Sig())
			}
		},
	})
	wg.Wait()

	if err != nil {
		ssi.DoClose(err)
		return err
	}

	switch v := p.(type) {
	case *packets.Connect:
		// handled the first time by expectToken(ssi, p)
	case *packets.Disconnect:
		ssi.WriteDownstream(v)
		fmt.Println("contact closing on disconnect")
		ssi.DoClose(errors.New("closing on disconnect"))
	case *packets.Subscribe:
		v.Address.EnsureAddressIsBinary()

		// every sub gets a jwtid except for the stats subs
		_, ok := v.GetOption("statsmax")
		if !ok && !config.IsGuru() {
			// it's a non-billing topic.
			// later, during heartbeat, it will send messages to this address
			tok := ssi.GetToken()
			if tok != nil {
				id := tok.JWTID
				v.SetOption("jwtid", []byte(id))
			}
		}
		looker.sendSubscriptionMessage(ssi, v)
	case *packets.Unsubscribe:
		v.Address.EnsureAddressIsBinary()
		looker.sendUnsubscribeMessage(ssi, v)
	case *packets.Lookup:
		v.Address.EnsureAddressIsBinary()
		looker.sendLookupMessage(ssi, v)
	case *packets.Send:
		v.Address.EnsureAddressIsBinary()
		looker.sendPublishMessage(ssi, v)
	case *packets.Ping:
		ssi.WriteDownstream(v)

	default:
		fmt.Printf("I don't know about native type : %T!\n", v)
	}
	return nil
}

// PushDownFromTop to deal with an incoming message going down.
// typically called by an upperChannel receiving a packet via it's tcp that it dialed.
// todo: upgrade and consolidate the address logic.
// there's no channel here and we're going straight to the lookup table.
// the dialGuru gadget calls this when it gets a packet from the guru.
func PushDownFromTop(looker *LookupTableStruct, p packets.Interface) error {

	got, ok := p.GetOption("debg")
	if ok && string(got) == "12345678" {
		fmt.Println("PushDownFromTop ", p.Sig())
	}

	switch v := p.(type) {
	case *packets.Connect:
		fmt.Println("got connect we don't need ", v)
	case *packets.Disconnect:
		fmt.Println("PushDownFromTop got disconnect from guru is this for us?  ", v) // this is really bad.
		//ignore it? ssi.Close(errors.New("got disconnect from guru"))
	case *packets.Subscribe:
		v.Address.EnsureAddressIsBinary()
		looker.sendSubscriptionMessageDown(v)
	// case *packets.Unsubscribe:
	// 	v.Address.EnsureAddressIsBinary()
	// 	looker.sendUnsubscribeMessageDown(v)
	// case *packets.Lookup:
	// 	v.Address.EnsureAddressIsBinary()
	// 	looker.sendLookupMessageDown(v)
	case *packets.Send:
		v.Address.EnsureAddressIsBinary()
		looker.sendPublishMessageDown(v)
	case *packets.Ping:
		// nothing
	default:
		fmt.Printf("PushDownFromTop donesn't know about type %T!\n", v)
	}
	return nil
}

// GetKey is because we're passing around an interface
func (ss *ContactStruct) GetKey() HalfHash {
	return ss.key
}

// GetConfig is because we're passing around an interface
func (ss *ContactStruct) GetConfig() *ContactStructConfig {
	return ss.config
}

// WriteDownstream is often overridden
// in *test* we force plain contacts on the bottom of the guru's
// they just need to write.
func (ss *ContactStruct) WriteDownstream(p packets.Interface) error {

	if ss.IsClosed() {
		return errors.New("closed contact")
	}
	got, ok := p.GetOption("debg")
	if ok && string(got) == "12345678" {
		fmt.Println("ContactStruct WriteDownstream con=", ss.GetKey().Sig(), p.Sig())
	}

	go func() {
		// fmt.Println("ContactStruct WriteDownstream 2 con=", ss.GetKey().Sig(), p.Sig())
		p.Write(ss)
	}()
	// Don't wait.
	return nil
}

// GetLookup is a getter
func (config *ContactStructConfig) GetLookup() *LookupTableStruct {
	return config.lookup
}

// Close closes the conn
// and the rest of the work too. doesn't send error or disconnect.
// needs to be overridden
func (ss *ContactStruct) DoClose(err error) {

	select {
	case <-ss.ClosedChannel:
		return
	default:
		// fall through
	}

	wg := sync.WaitGroup{}
	wg.Add(1)
	ss.commands <- ContactCommander{
		who: "DoClose",
		fn: func(ss *ContactStruct) {
			defer wg.Done()
			if ss.LogMeVerbose {
				fmt.Println("Closing special ", ss.GetKey().Sig(), " with err ", err)
			}
			ss.once.Do(func() {
				close(ss.ClosedChannel)
				ss.DoClosingWork(err)
			})
		},
	}
	wg.Wait()
}

// This will unlink and close the contact.
// Guaranteed to only happen once
// Alert:never call this direcly. Use DoClose(err) instead and it will call this correctly.
// You can override it though.
func (ss *ContactStruct) DoClosingWork(err error) {

	// fmt.Println("ContactStruct DoClosingWork con=", ss.GetKey().Sig(), err)

	_ = err
	config := ss.config
	config.listlock.Lock()
	if ss.ele != nil {
		config.listOfCi.Remove(ss.ele)
	}
	config.listlock.Unlock()
}

// GetSequence is
func (ss *ContactStruct) GetSequence() uint64 {
	return (uint64(ss.key) - ss.config.key.GetUint64()) / 13
}

// SetSequence is
// func (ss *ContactStruct) setSequence(seq uint64) {
// 	ss.key = HalfHash(ss.config.key.GetUint64() + seq*13)
// }

// Len returns the count of the contacts.
func (config *ContactStructConfig) Len() int {
	config.listlock.Lock()
	val := config.listOfCi.Len()
	config.listlock.Unlock()
	return val
}

// WriteUpstream will be overridden
// this is used by an upper contact and is overridden. See tcpUpperContact
func (ss *ContactStruct) WriteUpstream(cmd packets.Interface) error {
	fmt.Println("FIXME unused", cmd, reflect.TypeOf(cmd)) // fixme panic
	return errors.New("FIXME unused WriteUpstream")
}

// IsClosed returns if the contact was closed - in a go safe way.
func (ss *ContactStruct) IsClosed() bool {

	select {
	case <-ss.ClosedChannel:
		return true
	default:
		return false
	}
}

func (ss *ContactStruct) String() string {
	return fmt.Sprint("Contact" + ss.key.String())
}

// GetToken return the verified and decoded payload or else nil
func (ss *ContactStruct) GetToken() *tokens.KnotFreeTokenPayload {
	return ss.token
}

// SetToken set the verified and decoded payload
//
//	WARNING this is only called from inside a ContactCommander
func (ss *ContactStruct) SetToken(t *tokens.KnotFreeTokenPayload) {
	// do we need the cruft?
	// need to keep this because it's the billing topic: t.JWTID
	//var wg sync.WaitGroup
	//wg.Add(1)
	// fmt.Println("WriteCommand push set token")
	//ss.commands <- func(ss *ContactStruct)
	{
		t.URL = ""
		t.Issuer = ""
		ss.token = t
		//wg.Done()
	}
	//wg.Wait()
}

// GetExpires returns when the cc should expire
func (ss *ContactStruct) GetExpires() uint32 {
	var val uint32
	wg := sync.WaitGroup{}
	wg.Add(1)
	ss.commands <- ContactCommander{
		who: "GetExpires",
		fn: func(ss *ContactStruct) {
			val = ss.contactExpires
			wg.Done()
		},
	}
	wg.Wait()
	return val
}

// SetExpires sets when the ss will expire in unix time
func (ss *ContactStruct) SetExpires(when uint32) {
	ss.commands <- ContactCommander{
		who: "SetExpires",
		fn: func(ss *ContactStruct) {
			if when > ss.contactExpires {
				ss.contactExpires = when
			}
		},
	}
	// don't wait.
}

func (ss *ContactStruct) Read(p []byte) (int, error) {
	if ss.realReader == nil {
		panic("ss.realReader == nil")
	}
	n, err := ss.realReader.Read(p) // is Read thread safe?
	// ss.commands <- ContactCommander{
	// 	who: "Read",
	// 	fn: func(ss *ContactStruct) {
	// 		ss.input += n
	// 	},
	// }
	ss.input.Add(int64(n))
	// don't wait
	return n, err
}

func (ss *ContactStruct) Write(p []byte) (int, error) {
	if ss.realWriter == nil {
		// panic("ss.realWriter == nil")
		return 0, errors.New("ss.realWriter == nil")
	}
	// fmt.Println("contact write", string(p))
	n, err := ss.realWriter.Write(p)
	// ss.commands <- ContactCommander{ // would fill the channel
	// 	who: "Write",
	// 	fn: func(ss *ContactStruct) {
	// 		ss.output += n
	// 	},
	// }
	// don't wait.
	ss.output.Add(int64(n))
	return n, err
}

// WriteByte implements BufferedWriter for libmqtt
func (ss *ContactStruct) WriteByte(c byte) error {
	if ss.realWriter == nil {
		panic("ss.realWriter == nil")
	}
	var data [1]byte
	data[0] = c
	n, err := ss.Write(data[:])
	_ = n
	return err
}

// ReadByte implements BufferedReader for libmqtt
func (ss *ContactStruct) ReadByte() (byte, error) {
	if ss.realReader == nil {
		panic("ss.realReader == nil")
	}
	var data [1]byte
	n, err := ss.Read(data[:])
	// ss.commands <- ContactCommander{ // would fill the channel
	// 	who: "ReadByte",
	// 	fn: func(ss *ContactStruct) {
	// 		ss.output += n
	// 	},
	// } // don't wait
	ss.output.Add(int64(n))
	return data[0], err
}

func expectToken(ssi ContactInterface, p packets.Interface) error {
	if ssi.GetToken() == nil {
		// we can't do anything if we're not 'checked in'
		connectPacket, ok := p.(*packets.Connect)
		if !ok {
			return makeErrorAndDisconnect(ssi, "expected Connect packet", nil)
		}
		b64Token, ok := connectPacket.GetOption("token")
		if !ok || b64Token == nil {
			return makeErrorAndDisconnect(ssi, "expected token", nil)
		}
		comment, hasComment := connectPacket.GetOption("comment")
		if hasComment {
			fmt.Println("comment", string(comment), "from", ssi.GetKey().Sig())
		}
		trimmedToken, issuer, err := tokens.GetKnotFreePayload(string(b64Token))
		if err != nil {
			return makeErrorAndDisconnect(ssi, "", err)
		}
		// find the public key that matches.
		publicKeyBytes := tokens.FindPublicKey(issuer)
		if len(publicKeyBytes) != 32 {
			return makeErrorAndDisconnect(ssi, "token bad issuer "+issuer, nil)
		}
		foundPayload, ok := tokens.VerifyToken([]byte(trimmedToken), []byte(publicKeyBytes))
		if !ok {
			return makeErrorAndDisconnect(ssi, "token not verified", nil)
		}
		nowsec := ssi.GetConfig().GetCe().timegetter() // uint32(time.Now().Unix())
		if nowsec > foundPayload.ExpirationTime {
			return makeErrorAndDisconnect(ssi, "token expired", nil)
		}

		ssi.SetToken(foundPayload) // we're already in the contact loop thread
		{                          // subscribe to token for billing
			foundPayload.KnotFreeContactStats.Subscriptions += 1 // for billing subscription
			billstr, err := json.Marshal(foundPayload.KnotFreeContactStats)
			if err != nil {
				return makeErrorAndDisconnect(ssi, "", nil)
			}
			sub := packets.Subscribe{}
			id := ssi.GetToken().JWTID
			sub.Address.FromString(id) // the billing channel real name JWTID
			// fmt.Println("contact subscribing to ", ssi.GetToken().JWTID)
			sub.SetOption("statsmax", billstr)
			sub.SetOption("noack", []byte("1"))
			go PushPacketUpFromBottom(ssi, &sub)
		}
		return nil
	}
	return nil
}

func makeErrorAndDisconnect(ssi ContactInterface, str string, err error) error {
	if err == nil {
		err = errors.New(str)
	}
	go func() { // must not block.
		dis := &packets.Disconnect{}
		dis.SetOption("error", []byte(err.Error()))
		ssi.WriteDownstream(dis)
		fmt.Println("contacts makeErrorAndDisconnect", str, err)
		ssi.DoClose(err)
	}()
	return err
}

// HasError literally means does this packet have an "error" option
// returns a Disconnect if the p has an error
func HasError(p packets.Interface) *packets.Disconnect {

	errmsg, ok := p.GetOption("error")
	if ok {
		dis := packets.Disconnect{}
		dis.SetOption("error", errmsg)
		return &dis
	}
	return nil
}

// GetRates to peek into in, out, dt := cc.GetRates(now)
// fixme: have stats call a billingAccumulator on heartbeat.
func (ss *ContactStruct) GetRates(now uint32) (int, int, int) {

	var in int64
	var out int64
	var dt uint32
	var wg sync.WaitGroup
	wg.Add(1)
	ss.commands <- ContactCommander{
		who: "GetRates",
		fn: func(ss *ContactStruct) {
			ss.input.Store(in)
			ss.output.Store(out)
			dt = now - ss.lastBillingTime
			if dt > 4*300 { // ? is our normal reporting interval
				dt = 0
			}
			wg.Done()
		},
	}
	wg.Wait()
	return int(in), int(out), int(dt) // return uint64
}

// SetReader allows test to monkey with the flow
func (ss *ContactStruct) SetReader(r io.Reader) {
	ss.realReader = r
}

// SetWriter used by helpersof_test.go
func (ss *ContactStruct) SetWriter(w io.Writer) {
	ss.realWriter = w
}

func (ss *ContactStruct) sendBillingInfo(now uint32) {

	if ss.IsClosed() {
		return
	}
	var config *ContactStructConfig
	// var tok *tokens.KnotFreeTokenPayload
	msg := &Stats{}
	var deltaTime uint32
	var wg sync.WaitGroup
	wg.Add(1)
	ss.commands <- ContactCommander{
		who: "sendBillingInfp",
		fn: func(ss *ContactStruct) {

			config = ss.config
			// tok = ss.token
			deltaTime := ss.nextBillingTime - ss.lastBillingTime
			ss.lastBillingTime = ss.nextBillingTime
			ss.nextBillingTime += 60 // 60 secs after first time
			var tmp int64
			ss.input.Store(tmp)
			msg.Input = float64(tmp)
			ss.input.Add(-tmp)
			ss.output.Store(tmp)
			msg.Output = float64(tmp)
			ss.output.Add(-tmp)
			msg.Connections = float64(deltaTime) // means one per sec, one per min ... one
			wg.Done()
		},
	}
	wg.Wait()
	go func() {
		// also send to exec
		config.lookup.ex.Billing.AddUsage(&msg.KnotFreeContactStats, now, int(deltaTime))

		// Subscriptions handled elsewhere.
		p := &packets.Send{}
		// fmt.Println("contact publishing to ", ss.token.JWTID)
		p.Address.FromString(ss.token.JWTID)
		p.Source.FromString("billing_stats_return_address_contact")
		str, err := json.Marshal(msg)
		if err != nil {
			fmt.Println("impossible#3")
		}
		p.SetOption("add-stats", str)
		p.SetOption("stats-deltat", []byte(strconv.FormatInt(int64(deltaTime), 10)))

		//fmt.Println("contact heartbeat sending stats", p, "from", ss.config.Name)

		// don't bill a billing subscripton for the guru.

		if !config.IsGuru() {
			doSetExpires := false
			err = PushPacketUpFromBottom2(ss, p, doSetExpires)
		}
		if err != nil {
			fmt.Println("things before")
		}
	}()
	// don't wait
}

// Heartbeat is periodic service ~= 10 sec
// It's going to forward stats to to the billing channel
// 50/60 times it'll do nothing.
func (ss *ContactStruct) Heartbeat(now uint32) {

	var config *ContactStructConfig
	var nextBillingTime uint32
	var token *tokens.KnotFreeTokenPayload
	var expires uint32
	var wg sync.WaitGroup
	wg.Add(1)
	ss.commands <- ContactCommander{
		who: "Heartbeat",
		fn: func(ss *ContactStruct) {
			config = ss.config
			nextBillingTime = ss.nextBillingTime
			token = ss.token
			expires = ss.contactExpires
			wg.Done()
		},
	}
	wg.Wait()
	if token == nil {
		// it's not even started yet.
		return
	}
	// Guru clients don't bill. ? FIXME:
	if nextBillingTime < now { // && !ss.GetConfig().IsGuru() {
		ss.sendBillingInfo(now)
	}
	if !config.IsGuru() {
		if expires < now {
			fmt.Println("contact timed out in heartbeat")
			ss.DoClose(errors.New("timed out in heartbeat "))
		}
	}
}

// IncOutput so test and fake bytes written
func (ss *ContactStruct) IncOutput(amt int) {
	// ss.commands <- ContactCommander{
	// 	who: "IncOutput",
	// 	fn: func(ss *ContactStruct) {
	// 		ss.output += amt
	// 	},
	// }
	ss.output.Add(int64(amt))
}
