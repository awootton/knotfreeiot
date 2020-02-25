// Copyright 2019,2020 Alan Tracey Wootton
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

package iot

import (
	"container/list"
	"crypto/sha256"
	"errors"
	"fmt"
	"reflect"
	"sync"

	"github.com/awootton/knotfreeiot/badjson"
	"github.com/awootton/knotfreeiot/packets"
	"github.com/awootton/knotfreeiot/tokens"
)

// ContactStruct is our idea of channel or socket to downstream from us.
type ContactStruct struct {
	//
	ele *list.Element

	config *ContactStructConfig
	key    HalfHash // something unique.

	// not sure about this one. At the upper levels a socket could own millons of these.
	// and maybe the root doesn't want the real names.
	// but then how do we unsubscribe when the tcp conn fails? (don't, timeout)
	topicToName map[HalfHash][]byte // a tree would be better?

	token *tokens.KnotFreePayload
}

// ContactInterface is usually supplied by a tcp connection
type ContactInterface interface {
	Close(err error)

	GetKey() HalfHash

	GetToken() *tokens.KnotFreePayload
	SetToken(*tokens.KnotFreePayload)

	GetConfig() *ContactStructConfig

	WriteDownstream(cmd packets.Interface) error

	WriteUpstream(cmd packets.Interface) error // called by LookupTableStruct.PushUp

	// the upstream write is Push (below)
	String() string // used as a default channel name in test
}

func (ss *ContactStruct) String() string {
	return fmt.Sprint("Contact" + ss.key.String())
}

// GetToken return the verified and decoded payload or else nil
func (ss *ContactStruct) GetToken() *tokens.KnotFreePayload {
	return ss.token
}

// SetToken return the verified and decoded payload or else nil
func (ss *ContactStruct) SetToken(t *tokens.KnotFreePayload) {
	ss.token = t
}

// ContactStructConfig could be just a stack frame but I'd like to return it.
// This could be an interface that implements range and len or and the callbacks.
// Instead we have function pointers. TODO: revisit.
type ContactStructConfig struct {

	// a linked list of all the *Contacts that are open and not Close'd
	list     *list.List
	listlock *sync.RWMutex // so it's thread safe

	key HashType // everyone needs to feel special and unique

	sequence uint64 // every time we factory init a Contact we increment this and we never decrement.

	lookup *LookupTableStruct // LookupTableInterface

	address string // eg knotfreeserver:7009

	Name string // for debug

	defaultTimeoutSeconds uint32 // in seconds

	ce *ClusterExecutive // optional
}

// GetContactsList so we can disconnect them in test
func (config *ContactStructConfig) GetContactsList() *list.List {
	return config.list
}

// AddContactStruct initializes a contact, and puts the new ss on the global
// list. It also increments the sequence number in SockStructConfig.
// note that you must pass the same object twice, once as a ContactStruct and once as the Interface
func AddContactStruct(ss *ContactStruct, ssi ContactInterface, config *ContactStructConfig) *ContactStruct {

	ss.config = config

	ss.topicToName = make(map[HalfHash][]byte)

	config.listlock.Lock()
	defer config.listlock.Unlock()
	if ss.key == 0 {
		seq := config.sequence
		config.sequence++
		ss.key = HalfHash(seq + config.key.GetUint64())
	}
	ss.ele = ss.config.list.PushBack(ssi)

	return ss
}

// InitUpperContactStruct because upper contacts are different
// they are not linked like the others, they are saved in a map in lookup
func InitUpperContactStruct(ss *ContactStruct, config *ContactStructConfig) *ContactStruct {

	ss.topicToName = make(map[HalfHash][]byte)
	ss.config = config

	return ss
}

// NewContactStructConfig is
func NewContactStructConfig(looker *LookupTableStruct) *ContactStructConfig {
	config := ContactStructConfig{}
	config.lookup = looker
	looker.config = &config
	var alock sync.RWMutex
	config.listlock = &alock
	config.list = list.New()
	config.key.Random()
	config.sequence = 1
	config.defaultTimeoutSeconds = 10
	return &config
}

// Push to deal with an incoming message on a bottom contact heading up.
// todo: upgrade and consolidate the address logic.
func Push(ssi ContactInterface, p packets.Interface) error {

	config := ssi.GetConfig()
	looker := config.GetLookup()
	var destination *HashType

	if ssi.GetToken() == nil {
		// we can't do the rest if we're not 'checked in'
		connectPacket, ok := p.(*packets.Connect)
		if !ok {
			err := errors.New("expected Connect packet")
			dis := packets.Disconnect{}
			dis.SetOption("error", []byte(err.Error()))
			ssi.WriteDownstream(&dis) // is this redundant?
			ssi.Close(err)
			return err
		}
		b64Token, ok := connectPacket.GetOption("token")
		if ok == false || b64Token == nil {
			err := errors.New("expected token in Connect packet")
			dis := packets.Disconnect{}
			dis.SetOption("error", []byte(err.Error()))
			ssi.WriteDownstream(&dis) // is this redundant?
			ssi.Close(err)
			return err
		}
		payload, hashb64, err := tokens.GetKnotFreePayload(string(b64Token))
		if err != nil {
			dis := packets.Disconnect{}
			dis.SetOption("error", []byte(err.Error()))
			ssi.WriteDownstream(&dis) // is this redundant?
			ssi.Close(err)
			return err
		}
		issuer := payload.Issuer
		// find the public key that matches.
		publicKeyBytes := tokens.FindPublicKey(issuer)
		if len(publicKeyBytes) != 32 {
			err := errors.New("bad issuer")
			dis := packets.Disconnect{}
			dis.SetOption("error", []byte(err.Error()))
			ssi.WriteDownstream(&dis) // is this redundant?
			ssi.Close(err)
			return err
		}
		foundPayload, ok := tokens.VerifyTicket(b64Token, []byte(publicKeyBytes))
		if !ok {
			err := errors.New("not verified")
			dis := packets.Disconnect{}
			dis.SetOption("error", []byte(err.Error()))
			ssi.WriteDownstream(&dis) // is this redundant?
			ssi.Close(err)
			return err
		}
		ssi.SetToken(foundPayload)
		// subscribe to token hash
		sub := packets.Subscribe{}
		sub.Address = []byte(hashb64)
		Push(ssi, &sub)
		return nil
	}

	switch v := p.(type) {
	case *packets.Connect:
		// fmt.Println(v)
	case *packets.Disconnect:
		//fmt.Println(v)
		ssi.WriteDownstream(v)
		ssi.Close(errors.New("got disconnect"))
	case *packets.Subscribe:
		//fmt.Println(v)
		if len(v.AddressAlias) < 24 {
			v.AddressAlias = make([]byte, 24)
			sh := sha256.New()
			sh.Write(v.Address)
			v.AddressAlias = sh.Sum(nil)
		}
		looker.sendSubscriptionMessage(ssi, v)
	case *packets.Unsubscribe:
		if len(v.AddressAlias) < 24 {
			v.AddressAlias = make([]byte, 24)
			sh := sha256.New()
			sh.Write(v.Address)
			v.AddressAlias = sh.Sum(nil)
		}
		looker.sendUnsubscribeMessage(ssi, v)
	case *packets.Lookup:
		//fmt.Println(v)
		if len(v.AddressAlias) < 24 {
			v.AddressAlias = make([]byte, 24)
			sh := sha256.New()
			sh.Write(v.Address)
			v.AddressAlias = sh.Sum(nil)
		}
		looker.sendLookupMessage(ssi, v)
	case *packets.Send:
		//fmt.Println(v)
		if len(v.AddressAlias) < 24 {
			v.AddressAlias = make([]byte, 24)
			sh := sha256.New()
			sh.Write(v.Address)
			v.AddressAlias = sh.Sum(nil)
		}
		looker.sendPublishMessage(ssi, v)

	default:
		fmt.Printf("I don't know about type %T!\n", v)
	}

	_ = destination
	_ = looker

	//	looker.Send(ss, p)
	return nil
}

// PushDown to deal with an incoming message going down.
// typically called by an upper Contact receiving a packet via tcp.
// todo: upgrade and consolidate the address logic.
func PushDown(ssi ContactInterface, p packets.Interface) error {

	config := ssi.GetConfig()
	looker := config.GetLookup()
	var destination *HashType

	switch v := p.(type) {
	case *packets.Connect:
		fmt.Println("got connect we don't need ", v)
	case *packets.Disconnect:
		fmt.Println("got disconnect  ", v)
		ssi.Close(errors.New("got disconnect"))
	case *packets.Subscribe:
		if len(v.AddressAlias) < 24 {
			v.AddressAlias = make([]byte, 24)
			sh := sha256.New()
			sh.Write(v.Address)
			v.AddressAlias = sh.Sum(nil)
		}
		looker.sendSubscriptionMessageDown(ssi, v)
	case *packets.Unsubscribe:
		if len(v.AddressAlias) < 24 {
			v.AddressAlias = make([]byte, 24)
			sh := sha256.New()
			sh.Write(v.Address)
			v.AddressAlias = sh.Sum(nil)
		}
		looker.sendUnsubscribeMessageDown(ssi, v)
	case *packets.Lookup:
		if len(v.AddressAlias) < 24 {
			v.AddressAlias = make([]byte, 24)
			sh := sha256.New()
			sh.Write(v.Address)
			v.AddressAlias = sh.Sum(nil)
		}
		looker.sendLookupMessageDown(ssi, v)
	case *packets.Send:
		if len(v.AddressAlias) < 24 {
			v.AddressAlias = make([]byte, 24)
			sh := sha256.New()
			sh.Write(v.Address)
			v.AddressAlias = sh.Sum(nil)
		}
		looker.sendPublishMessageDown(ssi, v)

	default:
		fmt.Printf("I don't know about type %T!\n", v)
	}

	_ = destination
	_ = looker

	//	looker.Send(ss, p)
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

// WriteDownstream needs to be overridden
func (ss *ContactStruct) WriteDownstream(cmd packets.Interface) error {
	panic("WriteDownstream needs to be overridden")
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
		ss.config.list.Remove(ss.ele)
		ss.config.listlock.Unlock()
		ss.ele = nil
	}
	if ss.topicToName != nil {
		for key, realName := range ss.topicToName {
			p := new(packets.Unsubscribe)
			p.Address = realName
			ss.config.lookup.sendUnsubscribeMessage(ss, p)
			_ = key
		}
		ss.topicToName = nil
	}
	//ss.key = 0
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

// Len is an obvious wrapper
func (config *ContactStructConfig) Len() int {
	config.listlock.Lock()
	val := config.list.Len()
	config.listlock.Unlock()
	return val
}

// WriteUpstream will be overridden
func (ss *ContactStruct) WriteUpstream(cmd packets.Interface) error {
	fmt.Println("FIXME unused", cmd, reflect.TypeOf(cmd)) // fixme panic
	return errors.New("FIXME unused WriteUpstream")
}

// ContactFactory is for exec
type ContactFactory func(config *ContactStructConfig) ContactInterface

// ContactAttach for when the contact exists and we want to attach it to the config
type ContactAttach func(cc ContactInterface, config *ContactStructConfig)

// Text2Packet turns badjson into a packet
func Text2Packet(text string) (packets.Interface, error) {
	// parse the text
	segment, err := badjson.Chop(text)
	if err != nil {
		fmt.Println("SendText badjson err", err)
		return nil, err
	}
	uni := packets.Universal{}
	uni.Args = make([][]byte, 64) // much too big
	tmp := segment.Raw()          // will not be quoted
	uni.Cmd = packets.CommandType(tmp[0])
	segment = segment.Next()

	// traverse the result
	i := 0
	for s := segment; s != nil; s = s.Next() {
		stmp := s.Raw()
		uni.Args[i] = []byte(stmp)
		i++
		if i > 10 {
			break
		}
	}
	p, err := packets.FillPacket(&uni)
	if err != nil {
		//fmt.Println("problem with packet", err)
	}
	return p, err
}
