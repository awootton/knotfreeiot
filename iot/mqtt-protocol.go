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
	"errors"
	"fmt"
	"net"
	"reflect"
	"time"

	"github.com/awootton/knotfreeiot/packets"
	mqttpackets "github.com/eclipse/paho.mqtt.golang/packets"
)

type mqttContact struct {
	tcpContact
}

// MakeMqttExecutive is a thing like a server, not the exec
func MakeMqttExecutive(ex *Executive, serverName string) *Executive {

	go mqttServer(ex, serverName)

	return ex

}

// a simple iot wire protocol that is mqtt based.

func (cc *mqttContact) WriteDownstream(p packets.Interface) error {

	// need packet switch here. like push

	switch v := p.(type) {
	case *packets.Connect:
		fmt.Println("cant happen")
	case *packets.Disconnect:
		mq := &mqttpackets.DisconnectPacket{}
		mq.MessageType = mqttpackets.Disconnect
		return mq.Write(cc.tcpConn)
	case *packets.Subscribe:
		fmt.Println("cant happen3")
		mq := &mqttpackets.SubscribePacket{}
		mq.MessageType = mqttpackets.Subscribe
		mq.Topics = []string{string(v.Address)}
		err := mq.Write(cc.tcpConn)
		return err

	case *packets.Unsubscribe:
		fmt.Println("cant happen4")

	case *packets.Lookup:

	case *packets.Send:

		mq := &mqttpackets.PublishPacket{}
		mq.MessageType = mqttpackets.Publish
		mq.TopicName = string(v.Address)
		mq.Retain = false
		mq.Payload = v.Payload
		//fmt.Println("mqtt pay", string(mq.Payload))
		err := mq.Write(cc.tcpConn)
		return err

	default:
		fmt.Printf("I don't know about type %T!\n", v)
	}

	return nil
}

// mqttServer serves a line oriented mqtt protocol
func mqttServer(ex *Executive, name string) {
	fmt.Println("mqtt service starting ", name)
	ln, err := net.Listen("tcp", name)
	if err != nil {
		// handle error
		//srvrLogThing.Collect(err.Error())
		fmt.Println("server didnt' stary ", err)
		return
	}
	for ex.IAmBadError == nil {
		//fmt.Println("Server listening")
		tmpconn, err := ln.Accept()
		if err != nil {
			//	srvrLogThing.Collect(err.Error())
			fmt.Println("accetp err ", err)
			continue
		}
		go mqttConnection(tmpconn.(*net.TCPConn), ex) //,handler types.ProtocolHandler)
	}
}

func mqttConnection(tcpConn *net.TCPConn, ex *Executive) {

	//srvrLogThing.Collect("Conn Accept")

	cc := localMakeMqttContact(ex.Config, tcpConn)
	defer cc.Close(nil)

	// connLogThing.Collect("new connection")

	err := SocketSetup(tcpConn)
	if err != nil {
		//connLogThing.Collect("server err " + err.Error())
		fmt.Println("setup err", err)
		return
	}
	mqttName := "unknown"
	for ex.IAmBadError == nil {
		// SetReadDeadline
		err := cc.tcpConn.SetDeadline(time.Now().Add(20 * time.Minute))
		if err != nil {
			//connLogThing.Collect("server err2 " + err.Error())
			fmt.Println("deadline err", err)
			cc.Close(err)
			return // quit, close the sock, be forgotten
		}
		//fmt.Println("waiting for packet")
		//str, err := lineReader.ReadString('\n')
		control, err := mqttpackets.ReadPacket(tcpConn)
		if err != nil {
			//connLogThing.Collect("se err " + err.Error())
			if err.Error() != "EOF" {
				fmt.Println("packets 1 read err", err)
			}
			cc.Close(err)
			return
		}

		fmt.Println("mqtt packet", control)
		// As much fun as it would be to make the following code into virtual methods
		// of the types involved (and I tried it) it's more annoying and harder to read
		// than just doing it all here.
		switch mq := control.(type) {

		case *mqttpackets.ConnectPacket:
			fmt.Println("have mqttpackets.ConnectPacket")
			p := &packets.Connect{}
			p.SetOption("token", mq.Password)
			mqttName = mq.Username
			// TODO: validate things.
			err = Push(cc, p)
			if err != nil {
				fmt.Println("mqtt push connect fail", err)
			}
			// write an ack
			conack := &mqttpackets.ConnackPacket{}
			conack.FixedHeader.MessageType = mqttpackets.Connack
			err = conack.Write(tcpConn)
			if err != nil {
				fmt.Println("errrr", err)
			}

		case *mqttpackets.PublishPacket: // handle upstream publish
			p := &packets.Send{}
			p.Address = []byte(mq.TopicName)
			p.Source = []byte(mqttName)
			p.Payload = mq.Payload
			p.SetOption("toself", []byte("y"))
			_ = Push(cc, p)
			// // TODO: do we need to ack?
			// ack := mqttpackets.PubackPacket ... etc
		case *mqttpackets.SubscribePacket:

			for _, topic := range mq.Topics {

				p := &packets.Subscribe{}
				p.Address = []byte(topic)
				_ = Push(cc, p)

			}
		case *mqttpackets.UnsubscribePacket:
			for _, topic := range mq.Topics {

				p := &packets.Unsubscribe{}
				p.Address = []byte(topic)
				_ = Push(cc, p)

			}
		case *mqttpackets.PingreqPacket:
			p := &mqttpackets.PingrespPacket{}
			p.MessageType = mqttpackets.Pingresp
			p.Write(tcpConn)

		case *mqttpackets.DisconnectPacket:
			fmt.Println("client sent us an error", mq)
			// client sent us an error. close.
			// str := "mqtt DisconnectPacket"
			// err := errors.New(str)
			// mqttLogThing.Collect(str)
			//ss.Close(err)
			// return
		default:
			// client sent us junk somehow
			str := "bad mqtt type=" + reflect.TypeOf(control).String()
			err = errors.New(str)
			//	mqttLogThing.Collect(str)
			//ss.Close(err)
			cc.Close(err)
			return
		}
	}

}

// localMakeMqttContact is a factory
func localMakeMqttContact(config *ContactStructConfig, tcpConn *net.TCPConn) *mqttContact {
	contact1 := mqttContact{}
	AddContactStruct(&contact1.ContactStruct, &contact1, config)
	contact1.tcpConn = tcpConn
	return &contact1
}
