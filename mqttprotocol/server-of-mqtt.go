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

package mqttprotocol

import (
	"errors"
	"fmt"
	"reflect"

	"knotfreeiot/iot/reporting"

	"knotfreeiot/iot"

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

// mqttServeCallback is what handles a socket after an incomeing mqtt conection is made.
// it expectes a ConnectPacket and then loops forever dealing with mqtt messages.
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
			//conpack := obj.(*packets.ConnectPacket)
			fmt.Println("have packets.ConnectPacket")
			connected = true
		case *packets.PublishPacket: // handle upstream publish
			pub := obj.(*packets.PublishPacket)
			payload := pub.Payload
			topic := pub.TopicName
			ss.SendPublishMessage([]byte(topic), payload, []byte("unknown"))
			// TODO: do we need to ack?
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
		case *packets.PingreqPacket:
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

// HandleTopicPayload writes a publish downstream.
func HandleTopicPayload(ss *iot.SockStruct, topic []byte, payload []byte, returnAddress []byte) error {

	pub := packets.PublishPacket{}
	pub.TopicName = string(topic)
	pub.Retain = false
	pub.Payload = payload

	err := mqttWrite(ss, &pub)
	return err
}

var mqttLogThing = reporting.NewStringEventAccumulator(16)
