package mqttprotocol

import (
	"errors"
	"fmt"
	"knotfreeiot/iot"
	"knotfreeiot/iot/reporting"
	"reflect"

	"github.com/eclipse/paho.mqtt.golang/packets"
)

// ServerOfMqtt - use the reader arch and implement mqtt
// returns a config to keep a handle on the sockets.
func ServerOfMqtt(subscribeMgr iot.PubsubIntf, addr string) *iot.SockStructConfig {

	config := iot.NewSockStructConfig(subscribeMgr)

	ServerOfMqttInit(config)

	iot.ServeFactory(config, addr)

	return config
}

// ServerOfMqttInit is to set default callbacks.
func ServerOfMqttInit(config *iot.SockStructConfig) {

	config.SetCallback(mqttServeCallback)

	servererr := func(ss *iot.SockStruct, err error) {
		mqttLogThing.Collect("mqtt server closing")
	}
	config.SetClosecb(servererr)

	config.SetWriter(HandleTopicPayload)
}

func mqttWrite(ss *iot.SockStruct, cmd packets.ControlPacket) error {

	err := cmd.Write(ss.GetConn())

	return err
}

// mqttServeCallback is
func mqttServeCallback(ss *iot.SockStruct) {

	// implement the protocol
	connected := false
	for {

		obj, err := packets.ReadPacket(ss.GetConn()) //(ControlPacket, error)
		if err != nil {
			str := "mqttRead err=" + err.Error()
			dis := packets.DisconnectPacket{}
			mqttWrite(ss, &dis)
			mqttLogThing.Collect(str)
			err := errors.New(str)
			ss.Close(err)
			return
		}
		if connected == false {
			_, isConnPacket := obj.(*packets.ConnectPacket)
			if isConnPacket == false {
				str := "mqtt expoected control packet"
				dis := packets.DisconnectPacket{}
				mqttWrite(ss, &dis)
				mqttLogThing.Collect(str)
				err := errors.New(str)
				ss.Close(err)
				return
			}
		}
		// As much fun as it would be to make the following code into virtual methods
		// of the types involved (and I tried it) it's more annoying and harder to read
		// than just doing it all here.
		switch obj.(type) {

		case *packets.ConnectPacket:
			//con := obj.(*packets.ConnectPacket)
			fmt.Println("have packets.ConnectPacket")
			connected = true
		case *packets.PublishPacket:
			pub := obj.(*packets.PublishPacket)
			payload := pub.Payload
			topic := pub.TopicName
			ss.SendPublishMessage([]byte(topic), payload)
			// ack := packets.PubackPacket ... etc
		case *packets.SubscribePacket:
			sub := obj.(*packets.SubscribePacket)
			for _, topic := range sub.Topics {
				ss.SendSubscriptionMessage([]byte(topic))
			}
		case *packets.UnsubscribePacket:
			unsub := obj.(*packets.UnsubscribePacket)
			for _, topic := range unsub.Topics {
				ss.SendUnsubscribeMessage([]byte(topic))
			}
		case *packets.PubrecPacket:
			mqttWrite(ss, &packets.PingrespPacket{})
		case *packets.DisconnectPacket:
			// client sent us an error. close.
			str := "mqtt DisconnectPacket"
			err := errors.New(str)
			mqttLogThing.Collect(str)
			ss.Close(err)
			return
		default:
			// client sent us junk somehow
			str := "bad mqtt type=" + reflect.TypeOf(obj).String()
			err := errors.New(str)
			mqttLogThing.Collect(str)
			ss.Close(err)
			return
		}
	}
}

// HandleTopicPayload writes a publish onto the  wire.
func HandleTopicPayload(ss *iot.SockStruct, topic []byte, payload []byte, returnAddress []byte) error {

	pub := packets.PublishPacket{}
	pub.TopicName = string(topic)
	pub.Retain = false
	pub.Payload = payload

	err := mqttWrite(ss, &pub)
	return err
}

var mqttLogThing = reporting.NewStringEventAccumulator(16)
