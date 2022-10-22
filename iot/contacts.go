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
	"bytes"
	"container/list"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"sync"
	"time"

	"github.com/awootton/knotfreeiot/packets"
	"github.com/awootton/knotfreeiot/tokens"
)

// A 'contact' is an incoming connection.

// ContactStruct is our idea of channel or socket which is downstream from us.
type ContactStruct struct {
	//
	ele *list.Element

	config *ContactStructConfig
	key    HalfHash // something unique.

	// not sure about this one. At the upper levels a socket could own thousands of these.
	// and maybe the root doesn't want the real names.
	// but then how do we unsubscribe when the tcp conn fails? (don't, timeout)
	// topicToName map[HalfHash][]byte // a tree would be better?

	// these are the limits, they have been sent to the subscription.
	// there is also the JWTID in the token.
	token *tokens.KnotFreeTokenPayload

	contactExpires uint32

	nextBillingTime uint32
	lastBillingTime uint32 // input and output were cleared at this time.
	//billingAmounts  tokens.KnotFreeContactStats // in out su co = 0.0
	input  int
	output int

	realReader io.Reader
	realWriter io.Writer
}

// ContactInterface is usually supplied by a tcp connection
type ContactInterface interface {
	Close(err error)

	GetClosed() bool

	GetKey() HalfHash

	GetExpires() uint32
	SetExpires(when uint32)

	GetToken() *tokens.KnotFreeTokenPayload
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

	GetRates(now uint32) (int, int, int)

	SetReader(r io.Reader)
	SetWriter(w io.Writer)
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

	address string // eg knotfreeserver:7009

	Name string // for debug

	defaultTimeoutSeconds uint32 // in seconds

	ce *ClusterExecutive // optional
}

// AccessContactsList so we can disconnect them in test and stuff.
// be sure to always lock. Don't call close or recurse in here or it will deadlock.
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

// IsGuru exposes onfig.lookup.isGuru
func (config *ContactStructConfig) IsGuru() bool {
	if config.lookup == nil {
		return false
	}
	return config.lookup.isGuru
}

// AddContactStruct initializes a contact, and puts the new ss on the global
// list. It also increments the sequence number in SockStructConfig.
// note that you must pass the same object twice, once as a ContactStruct and once as the Interface
func AddContactStruct(ss *ContactStruct, ssi ContactInterface, config *ContactStructConfig) *ContactStruct {

	ss.config = config

	config.AccessContactsList(func(config *ContactStructConfig, listOfCi *list.List) {
		if ss.key == 0 {
			seq := config.sequence
			config.sequence++
			ss.key = HalfHash(seq + config.key.GetUint64())
		}
		ss.ele = listOfCi.PushBack(ssi)
	})

	now := config.GetLookup().getTime()
	ss.contactExpires = 20*60 + now // stale contacts expire in 20 min.

	ss.nextBillingTime = now + 30 // 30 seconds to start with
	ss.lastBillingTime = now

	return ss
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

// PushPacketUpFromBottom to deal with an incoming message on a bottom contact heading up.
// it expects a token before anything else.
func PushPacketUpFromBottom(ssi ContactInterface, p packets.Interface) error {

	if ssi.GetClosed() {
		return errors.New("closed contact")
	}
	config := ssi.GetConfig()
	if config == nil {
		fmt.Println("no way")
	}
	looker := config.GetLookup()
	var destination *HashType

	ssi.SetExpires(20*60 + config.GetLookup().getTime())

	err := expectToken(ssi, p)
	if err != nil {
		return err
	}

	switch v := p.(type) {
	case *packets.Connect:
		// handled the first time by expectToken(ssi, p)
	case *packets.Disconnect:
		ssi.WriteDownstream(v)
		ssi.Close(errors.New("closing on disconnect"))
	case *packets.Subscribe:
		v.Address.EnsureAddressIsBinary()

		// every sub gets a jwtidAlias except for the stats subs
		_, ok := v.GetOption("statsmax")
		if !ok && !config.IsGuru() {
			// it's a non-billing topic.
			// later, during heartbeat, it will send messages to this address
			id := ssi.GetToken().JWTID
			v.SetOption("jwtidAlias", []byte(id))
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
		fmt.Printf("I don't know about type up %T!\n", v)
	}

	_ = destination
	_ = looker

	//	looker.Send(ss, p)
	return nil
}

// PushDownFromTop to deal with an incoming message going down.
// typically called by an upperChannel receiving a packet via it's tcp that it dialed.
// todo: upgrade and consolidate the address logic.
func PushDownFromTop(looker *LookupTableStruct, p packets.Interface) error {

	switch v := p.(type) {
	case *packets.Connect:
		fmt.Println("got connect we don't need ", v)
	case *packets.Disconnect:
		fmt.Println("got disconnect from guru  ", v)
		//ignore it. ssi.Close(errors.New("got disconnect from guru"))
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
		fmt.Printf("I don't know about type down %T!\n", v)
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
func (ss *ContactStruct) WriteDownstream(cmd packets.Interface) error {
	//fmt.Println("constract struct write off bottom", cmd)

	// all at once
	buff := &bytes.Buffer{}
	cmd.Write(buff)
	_, err := ss.Write(buff.Bytes())
	//	err := cmd.Write(ss)
	return err
}

// GetLookup is a getter
func (config *ContactStructConfig) GetLookup() *LookupTableStruct {
	return config.lookup
}

// Close closes the conn
// and the rest of the work too. doesn't send error or disconnect.
// needs to be overridden
func (ss *ContactStruct) Close(err error) {

	if ss.ele != nil && ss.config != nil {
		ss.config.listlock.Lock()
		ss.config.listOfCi.Remove(ss.ele)
		ss.config.listlock.Unlock()
		ss.ele = nil
	}
	ss.config = nil
}

// GetSequence is
func (ss *ContactStruct) GetSequence() uint64 {
	return (uint64(ss.key) - ss.config.key.GetUint64()) / 13
}

// SetSequence is
func (ss *ContactStruct) setSequence(seq uint64) {
	ss.key = HalfHash(ss.config.key.GetUint64() + seq*13)
}

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

// GetClosed because the contact is still referenced by looker after closed.
func (ss *ContactStruct) GetClosed() bool {
	// close always nulls the list and the config
	if ss.ele == nil || ss.config == nil {
		return true
	}
	return false
}

func (ss *ContactStruct) String() string {
	return fmt.Sprint("Contact" + ss.key.String())
}

// GetToken return the verified and decoded payload or else nil
func (ss *ContactStruct) GetToken() *tokens.KnotFreeTokenPayload {
	return ss.token
}

// SetToken return the verified and decoded payload or else nil
func (ss *ContactStruct) SetToken(t *tokens.KnotFreeTokenPayload) {
	// do we need the cruft?
	// need to keep this because it's the billing topic: t.JWTID
	t.URL = ""
	t.Issuer = ""
	ss.token = t
}

// GetExpires returns when the cc should expire
func (ss *ContactStruct) GetExpires() uint32 {
	return ss.contactExpires
}

// SetExpires sets when the ss will expire in unix time
func (ss *ContactStruct) SetExpires(when uint32) {
	if when > ss.contactExpires {
		ss.contactExpires = when
	}
}

func (ss *ContactStruct) Read(p []byte) (int, error) {
	if ss.realReader == nil {
		panic("ss.realReader == nil")
	}
	n, err := ss.realReader.Read(p)
	ss.input += n
	return n, err
}

func (ss *ContactStruct) Write(p []byte) (int, error) {
	if ss.realWriter == nil {
		panic("ss.realWriter == nil")
	}
	n, err := ss.realWriter.Write(p)
	ss.output += n
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
	ss.output++
	_ = n
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
		if ok == false || b64Token == nil {
			return makeErrorAndDisconnect(ssi, "expected token", nil)
		}
		trimmedToken, issuer, err := tokens.GetKnotFreePayload(string(b64Token))
		if err != nil {
			return makeErrorAndDisconnect(ssi, "", err)
		}
		// find the public key that matches.
		publicKeyBytes := tokens.FindPublicKey(issuer)
		if len(publicKeyBytes) != 32 {
			return makeErrorAndDisconnect(ssi, "token bad issuer", nil)
		}
		foundPayload, ok := tokens.VerifyToken([]byte(trimmedToken), []byte(publicKeyBytes))
		if !ok {
			return makeErrorAndDisconnect(ssi, "token not verified", nil)
		}
		nowsec := uint32(time.Now().Unix())
		if nowsec > foundPayload.ExpirationTime {
			return makeErrorAndDisconnect(ssi, "token expired", nil)
		}

		ssi.SetToken(foundPayload)
		{ // subscribe to token for billing
			foundPayload.KnotFreeContactStats.Subscriptions += 1 // for billing subscription
			billstr, err := json.Marshal(foundPayload.KnotFreeContactStats)
			if err != nil {
				return makeErrorAndDisconnect(ssi, "", nil)
			}
			sub := packets.Subscribe{}
			id := ssi.GetToken().JWTID
			sub.Address.FromString(id) // the billing channel real name JWTID
			sub.SetOption("statsmax", billstr)
			PushPacketUpFromBottom(ssi, &sub)
		}
		return nil
	}
	return nil
}

func makeErrorAndDisconnect(ssi ContactInterface, str string, err error) error {
	if err == nil {
		err = errors.New(str)
	}

	dis := &packets.Disconnect{}
	dis.SetOption("error", []byte(err.Error()))
	ssi.WriteDownstream(dis)
	ssi.Close(err)
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
func (ss *ContactStruct) GetRates(now uint32) (int, int, int) {
	in := ss.input
	out := ss.output
	dt := now - ss.lastBillingTime
	if dt > 4*300 { // 300 is our normal reporting interval
		dt = 0
	}
	return in, out, int(dt)
}

// SetReader allows test to monkey with the flow
func (ss *ContactStruct) SetReader(r io.Reader) {
	ss.realReader = r
}

// SetWriter used nuy helpersof_test.go
func (ss *ContactStruct) SetWriter(w io.Writer) {
	ss.realWriter = w
}

// Heartbeat is periodic service ~= 10 sec
// It's going forward stats to to the billing channel
func (ss *ContactStruct) Heartbeat(now uint32) {

	//fmt.Println("contact heartbeat ", ss.GetKey())
	if ss.token == nil {
		// it's not even started yet.
		return
	}

	if ss.nextBillingTime < now && !ss.GetConfig().IsGuru() {

		deltaTime := now - ss.lastBillingTime
		ss.lastBillingTime = now
		ss.nextBillingTime = now + 300 // 300 secs after first time

		//fmt.Println("delta t", deltaTime, ss.String())

		msg := &Stats{}
		//msg.Start = now
		msg.Input = float64(ss.input)
		ss.input -= int(msg.Input) // todo: atomic
		msg.Output = float64(ss.output)
		ss.output -= int(msg.Output)         // todo: atomic
		msg.Connections = float64(deltaTime) // means one per sec, one per min ...
		// Subscriptions handled elsewhere.
		p := &packets.Send{}
		p.Address.FromString(ss.token.JWTID)
		p.Source.FromString("billing_stats_return_address_contact")
		str, err := json.Marshal(msg)
		if err != nil {
			fmt.Println("impossible#3")
		}
		p.SetOption("stats", str)
		oldtimeout := ss.contactExpires // don't let heartbeat reset the expiration.
		//fmt.Println("contact heartbeat sending stats", p, "from", ss.config.Name)
		err = PushPacketUpFromBottom(ss, p)
		ss.contactExpires = oldtimeout
		if err != nil {
			fmt.Println("things before")
		}
	}
	if !ss.GetConfig().IsGuru() {
		if ss.contactExpires < now {
			ss.Close(errors.New("timed out in heartbeat "))
		}
	}
}

// IncOutput so test and fake bytes written
func (ss *ContactStruct) IncOutput(amt int) {
	ss.output += amt
}
