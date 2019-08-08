package iot

import (
	"encoding/gob"
	"errors"
	"fmt"
	"knotfreeiot/iot/reporting"
	"reflect"
)

type subMessage struct {
	topic HashType
}

// ServerOfGob - use the reader arch and use it to implement sub-servers and master-servers
// returns a config to keep a handle to the sockets.
func ServerOfGob(subscribeMgr PubsubIntf, addr string) *SockStructConfig {

	config := NewSockStructConfig(subscribeMgr)

	ServerOfGobInit(config)

	ServeFactory(config, addr)

	return config
}

// ServerOfGobInit is to set default callbacks.
func ServerOfGobInit(config *SockStructConfig) {

	setupGobTypes()

	//config.SetCallback(aaServeCallback)

	servererr := func(ss *SockStruct, err error) {
		gobLogThing.Collect("gob server closing")
	}
	config.SetClosecb(servererr)

	config.SetWriter(HandleTopicPayload)
}

// GobIntf is
type GobIntf interface {
	Write(ss *SockStruct) error
}

// ConnectMessage is
type ConnectMessage struct {
	token string
	name  string
}

func (m *ConnectMessage) Write(ss *SockStruct) error {
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

func (m *PublishMessage) Write(ss *SockStruct) error {
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

func (m *SubscribeMessage) Write(ss *SockStruct) error {
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

func (m *UnsubscribeMessage) Write(ss *SockStruct) error {
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
func xxgobWrite(ss *SockStruct, obj GobIntf) error {
	err := obj.Write(ss)
	return err
}

// HandleTopicPayload writes a publish onto the  wire.
// It's also the callback the pubsub uses.
// we don't have a command with two arguments.
func HandleTopicPayload(ss *SockStruct, topic []byte, payload []byte, returnAddress []byte) error {

	pub := NewPublishMessage(1)
	item := pub[0]
	item.topic = topic
	item.payload = payload
	item.returnAddress = returnAddress

	err := pub.Write(ss)

	return err
}

// ReadGob is
func ReadGob(ss *SockStruct) (GobIntf, error) {
	dec := gob.NewDecoder(ss.GetConn())
	var result GobIntf
	err := dec.Decode(&result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// gobServeCallback is
func gobServeCallback(ss *SockStruct) {

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
		// of the types involved (and I tried it) it's more annoying and harder to read
		// than just doing it all here.
		switch obj.(type) {

		case *ConnectMessage:
			fmt.Println("have ConnectMessage")
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
			str := "bad gob type=" + reflect.TypeOf(obj).String()
			err := errors.New(str)
			gobLogThing.Collect(str)
			ss.Close(err)
			return
		}
	}
}

var gobLogThing = reporting.NewStringEventAccumulator(16)
