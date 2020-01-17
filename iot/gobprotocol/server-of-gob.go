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

package gobprotocol

import (
	"encoding/gob"
	"errors"
	"reflect"

	"github.com/awootton/knotfreeiot/iot"
	"github.com/awootton/knotfreeiot/iot/reporting"
)

// atw DELETE ME not using this later

type subMessage struct {
	topic iot.HashType
}

// ServerOfGob - use the reader arch and use it to implement sub-servers and master-servers
// returns a config to keep a handle to the sockets.
func ServerOfGob(subscribeMgr iot.PubsubIntf, addr string) *iot.SockStructConfig {

	config := iot.NewSockStructConfig(subscribeMgr)

	ServerOfGobInit(config)

	iot.ServeFactory(config, addr)

	return config
}

// ServerOfGobInit is to set default callbacks.
func ServerOfGobInit(config *iot.SockStructConfig) {

	setupGobTypes()

	//config.SetCallback(aaServeCallback)

	servererr := func(ss *iot.SockStruct, err error) {
		gobLogThing.Collect("gob server closing")
	}
	config.SetClosecb(servererr)

	config.SetWriter(HandleTopicPayload)
}

// GobIntf is
type GobIntf interface {
	Write(ss *iot.SockStruct) error
}

// ConnectMessage is
type ConnectMessage struct {
	token string
	name  string
}

func (m *ConnectMessage) Write(ss *iot.SockStruct) error {
	enc := gob.NewEncoder(ss.GetConn())
	err := enc.Encode(m)
	return err
}

// PublishItem is
type PublishItem struct {
	topic         []byte
	payload       []byte
	returnAddress []byte
}

// PublishMessage is an array
type PublishMessage []PublishItem

func (m *PublishMessage) Write(ss *iot.SockStruct) error {
	enc := gob.NewEncoder(ss.GetConn())
	err := enc.Encode(m)
	return err
}

// NewPublishMessage is to force the type onto the array
func NewPublishMessage(size int) PublishMessage {
	pub := make([]PublishItem, size)
	return pub
}

// SubscribeItem is
type SubscribeItem struct {
	topic []byte
}

// SubscribeMessage is
type SubscribeMessage []SubscribeItem

func (m *SubscribeMessage) Write(ss *iot.SockStruct) error {
	enc := gob.NewEncoder(ss.GetConn())
	err := enc.Encode(m)
	return err
}

// UnsubscribeItem is
type UnsubscribeItem struct {
	topic []byte
}

// UnsubscribeMessage is
type UnsubscribeMessage []UnsubscribeItem

func (m *UnsubscribeMessage) Write(ss *iot.SockStruct) error {
	enc := gob.NewEncoder(ss.GetConn())
	err := enc.Encode(m)
	return err
}

func setupGobTypes() {
	gob.Register(ConnectMessage{})
	gob.Register(PublishMessage{})
	gob.Register(SubscribeMessage{})
	gob.Register(UnsubscribeMessage{})

}

// gobWrite is redundant
func xxgobWrite(ss *iot.SockStruct, obj GobIntf) error {
	err := obj.Write(ss)
	return err
}

// HandleTopicPayload writes a publish onto the  wire.
// It's also the callback the pubsub uses.
// we don't have a command with two arguments.
func HandleTopicPayload(ss *iot.SockStruct, topic []byte, topicAlias *iot.HashType, returnAddress []byte, returnAlias *iot.HashType, payload []byte) error {

	pub := NewPublishMessage(1)
	item := pub[0]
	item.topic = topic
	item.payload = payload
	item.returnAddress = returnAddress

	err := pub.Write(ss)

	return err
}

// ReadGob is
func ReadGob(ss *iot.SockStruct) (GobIntf, error) {
	dec := gob.NewDecoder(ss.GetConn())
	var result GobIntf
	err := dec.Decode(&result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// gobServeCallback is
func gobServeCallback(ss *iot.SockStruct) {

	// implement the protocol
	connected := false
	for {
		obj, err := ReadGob(ss) //(GobIntf, error)
		if err != nil {
			str := "gobRead err=" + err.Error()
			//	dis := packets.DisconnectPacket{}
			//	mqttWrite(ss, &dis)
			gobLogThing.Collect(str)
			err := errors.New(str)
			ss.Close(err)
			return
		}
		if connected == false {
			_, isConnPacket := obj.(*ConnectMessage)
			if isConnPacket == false {
				str := "gob expected hello packet"
				//dis := packets.DisconnectPacket{}
				//mqttWrite(ss, &dis)
				gobLogThing.Collect(str)
				err := errors.New(str)
				ss.Close(err)
				return
			}
		}
		// As much fun as it would be to make the following code into virtual methods
		// (and I tried it) it's more annoying and harder to read
		// than just doing it all here.
		switch obj.(type) {

		case *ConnectMessage:
			//fmt.Println("have ConnectMessage")
			gobLogThing.Collect("gob ConnectMessage")
			connected = true
		case *PublishMessage:
			pub := obj.(*PublishMessage)
			for _, item := range *pub {
				ss.SendPublishMessage(item.topic, item.payload, item.returnAddress)
			}
		case *SubscribeMessage:
			sub := obj.(*SubscribeMessage)
			for _, item := range *sub {
				ss.SendSubscriptionMessage(item.topic)
			}
		case *UnsubscribeMessage:
			sub := obj.(*UnsubscribeMessage)
			for _, item := range *sub {
				ss.SendSubscriptionMessage(item.topic)
			}
		default:
			// client sent us junk somehow
			str := "gob type=" + reflect.TypeOf(obj).String()
			err := errors.New(str)
			gobLogThing.Collect(str)
			ss.Close(err)
			return
		}
	}
}

var gobLogThing = reporting.NewStringEventAccumulator(16)
