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

package iot

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"reflect"
	"sync"

	"time"

	"github.com/awootton/knotfreeiot/packets"
	"github.com/gorilla/websocket"
	"github.com/thei4t/libmqtt" // was mqttpacket "github.com/eclipse/paho.mqtt.golang/packets"
)

type mqttContact struct {
	tcpContact
	protoVersion   libmqtt.ProtoVersion
	writeLibPacket func(libPacket libmqtt.Packet, cc *mqttContact) error
}

type mqttWsContact struct {
	mqttContact
	wsConn           *websocket.Conn
	writebuff        bytes.Buffer
	writeAccessMutex sync.Mutex
}

func (cc *mqttWsContact) Close(err error) {
	hadConfig := cc.GetConfig() != nil
	ss := &cc.ContactStruct
	ss.Close(err) // close my parent
	if hadConfig {
		dis := packets.Disconnect{}
		dis.SetOption("error", []byte(err.Error()))
		cc.WriteDownstream(&dis)
		// if cc.netDotTCPConn != nil {
		// 	cc.netDotTCPConn.Close()
		// }
	}
}

// MakeMqttExecutive is a thing like a server, not the exec
func MakeMqttExecutive(ex *Executive, serverName string) *Executive {

	go mqttServer(ex, serverName)

	return ex
}

// a simple iot wire protocol that is mqtt based.

// mqttServer serves a   mqtt protocol
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
	//mqttName := "unknown"
	//_ = mqttName
	for ex.IAmBadError == nil { // the blocking read loop
		// SetReadDeadline
		if cc.GetToken() == nil && false { // FIXME: shorter for prod.
			err := cc.netDotTCPConn.SetDeadline(time.Now().Add(10 * time.Second))
			if err != nil {
				//connLogThing.Collect("server err2 " + err.Error())
				fmt.Println("deadline err m1", err)
				cc.Close(err)
				return // quit, close the sock, be forgotten
			}
		} else {
			err := cc.netDotTCPConn.SetDeadline(time.Now().Add(20 * time.Minute))
			if err != nil {
				//connLogThing.Collect("server err2 " + err.Error())
				fmt.Println("deadline err ,2", err)
				cc.Close(err)
				return // quit, close the sock, be forgotten
			}
		}
		//fmt.Println("waiting for packet", time.Now())
		//control, err := mqttpackets.ReadPacket(tcpConn)
		// Decode(version ProtoVersion, r BufferedReader) (Packet, error)
		//protoVersion := libmqtt.V311
		control, err := libmqtt.Decode(cc.protoVersion, cc)
		// fmt.Println("got decode packet", control, err)
		if err != nil {
			//connLogThing.Collect("se err " + err.Error())
			if err.Error() != "EOF" {
				fmt.Println("packets 1 read err", err, time.Now())
			}
			cc.Close(err)
			return
		}

		MQTTHandlePacket(cc, control)
	}
}

// MQTTHandlePacket is for when the packet was parsed elsewhere (like in the websocket).
func MQTTHandlePacket(cc *mqttContact, control libmqtt.Packet) {

	//fmt.Println("mqtt packet", control)
	// As much fun as it would be to make the following code into virtual methods
	// of the types involved (and I tried it) it's more annoying and harder to read
	// than just doing it all here.
	var err error
	switch mq := control.(type) {

	case *libmqtt.ConnPacket:

		fmt.Println("have mqttpackets.ConnectPacket")

		p := &packets.Connect{}
		if len(mq.Password) == 0 {
			p.SetOption("token", []byte(mq.Username))
		} else {
			p.SetOption("token", []byte(mq.Password))
		}
		// = mq.Username
		cc.protoVersion = mq.Version()
		err = PushPacketUpFromBottom(cc, p)
		if err != nil {
			str := fmt.Sprint("mqtt push connect fail", err) // needs prom counter
			err = errors.New(str)
			fmt.Println(str)
			cc.Close(err)
			return
		}
		// write an ack
		conack := &libmqtt.ConnAckPacket{}
		//conack.FixedHeader.MessageType = mqttpackets.Connack
		//err = conack.WriteTo(cc)
		err = cc.writeLibPacket(conack, cc)
		if err != nil {
			fmt.Println("mqtt conn fail", err) // needs prom counter
		}

	case *libmqtt.PublishPacket: // handle upstream publish

		//fmt.Println("mqtt client publish to", mq.TopicName)

		p := &packets.Send{}
		p.Address.FromString(mq.TopicName)
		p.Payload = mq.Payload
		if mq.Props != nil {
			p.Source.FromString(mq.Props.RespTopic)
			for k, v := range mq.Props.UserProps {
				p.SetOption(k, []byte(fmt.Sprint(v)))
			}
		}
		//p.SetOption("toself", []byte("y"))
		p.SetOption("atwtestn", []byte("4321"))
		bytes, ok := p.GetOption("lookup")
		if ok && string(bytes) == "lookup" {
			// we need to chanage this to a lookup
			pp := &packets.Lookup{}
			pp.Address.FromString(mq.TopicName)
			if mq.Props != nil {
				pp.Source.FromString(mq.Props.RespTopic)
				for k, v := range mq.Props.UserProps {
					p.SetOption(k, []byte(fmt.Sprint(v)))
				}
			}
			_ = PushPacketUpFromBottom(cc, pp)
		} else {
			_ = PushPacketUpFromBottom(cc, p)
			// // TODO: do we need to ack?
			// ack := mqttpackets.PubackPacket ... etc

		}

	case *libmqtt.SubscribePacket:

		for _, topic := range mq.Topics {

			fmt.Println("mqtt client subscribes to", topic)

			p := &packets.Subscribe{}
			p.Address.FromString(topic.Name)
			err = PushPacketUpFromBottom(cc, p)
			if err != nil {
				fmt.Println("mqtt sub fail", err) // needs prom counter
			}
		}
		// write an ack
		suback := &libmqtt.SubAckPacket{}
		suback.Codes = []byte{0}
		suback.PacketID = mq.PacketID
		err = cc.writeLibPacket(suback, cc)
		if err != nil {
			fmt.Println("mqtt conn fail", err) // needs prom counter
		}

	case *libmqtt.UnsubPacket:
		for _, topic := range mq.TopicNames {

			p := &packets.Unsubscribe{}
			p.Address.FromString(topic)
			_ = PushPacketUpFromBottom(cc, p)
		}
	default:
		if mq.Type() == libmqtt.PingReqPacket.Type() {
			cc.writeLibPacket(libmqtt.PingRespPacket, cc)
		} else {
			// client sent us junk somehow
			str := "bad mqtt type=" + reflect.TypeOf(control).String()
			err = errors.New(str)
			//	mqttLogThing.Collect(str)
			fmt.Println("unhandled mqttp packet", str)
			//ss.Close(err)
			cc.Close(err)
			return
		}
	}

}

func (cc *mqttContact) WriteDownstream(p packets.Interface) error {

	// need packet switch here. like push

	switch v := p.(type) {
	case *packets.Connect:
		fmt.Println("cant happen")
	case *packets.Disconnect:
		mq := &libmqtt.DisconnPacket{}
		mq.Props = &libmqtt.DisconnProps{}
		//	mq.MessageType = mqttpackets.Disconnect
		estr, ok := v.GetOption("error")
		if ok {
			mq.Props.Reason = string(estr)
		}
		//mq.ProtoVersion = cc.protoVersion
		return cc.writeLibPacket(mq, cc) // mq.WriteTo(cc)
	case *packets.Subscribe:
		fmt.Println("cant happen3")
		mq := &libmqtt.SubscribePacket{}
		//mq.MessageType = mqttpackets.Subscribe
		//mq.Topics = []string{string(v.Address)}
		//	mq.ProtoVersion = cc.protoVersion
		err := cc.writeLibPacket(mq, cc) //mq.WriteTo(cc)
		return err

	case *packets.Unsubscribe:
		fmt.Println("cant happen4")

	case *packets.Lookup:
		// what form does a lookup take in mqtt ?
		mq := &libmqtt.PublishPacket{}
		mq.Payload = []byte("had lookup") //v.??
		mq.TopicName = v.Address.String()
		if len(mq.TopicName) == 0 {
			mq.TopicName = "fixme_need_topic always" // fixme:
		}
		if cc.protoVersion == 5 {
			mq.Props = &libmqtt.PublishProps{}

			mq.Props.UserProps = make(map[string][]string)
			// if v.SourceAlias != nil { // fixme: must always be something.
			// 	mq.Props.RespTopic = string(v.SourceAlias)
			// } else {
			// 	mq.Props.RespTopic = "xxTEST/TIMEefghijk"
			// }
			mq.Props.RespTopic = v.Source.String()

			keys, values := v.GetOptionKeys()
			for i, key := range keys {
				mq.Props.UserProps.Add(key, string(values[i]))
			}
			mq.Props.UserProps.Add("atw", "test1")
			mq.Props.UserProps.Add("lookup", "lookup")
		}

		err := cc.writeLibPacket(mq, cc)
		return err

	case *packets.Send:

		mq := &libmqtt.PublishPacket{}
		mq.Payload = v.Payload
		mq.TopicName = v.Address.String()
		if len(mq.TopicName) == 0 {
			mq.TopicName = "fixme_need_topic" // fixme:
		}

		if cc.protoVersion == 5 {
			mq.Props = &libmqtt.PublishProps{}

			mq.Props.UserProps = make(map[string][]string)
			// if v.SourceAlias != nil { // fixme: must always be something.
			// 	mq.Props.RespTopic = string(v.SourceAlias)
			// } else {
			// 	mq.Props.RespTopic = "xxTEST/TIMEefghijk"
			// }
			mq.Props.RespTopic = v.Source.String()

			keys, values := v.GetOptionKeys()
			for i, key := range keys {
				mq.Props.UserProps.Add(key, string(values[i]))
			}
			mq.Props.UserProps.Add("atw", "test1")
		}

		//mq.ProtoVersion = cc.protoVersion
		err := cc.writeLibPacket(mq, cc) // mq.WriteTo(cc)

		// since there's no message in mqtt disconnect, send the pub first.
		u := HasError(v)
		if u != nil {
			mq := &libmqtt.DisconnPacket{}
			mq.Props = &libmqtt.DisconnProps{}
			//	mq.MessageType = mqttpackets.Disconnect
			// no place for message
			//mq.ProtoVersion = cc.protoVersion
			cc.writeLibPacket(mq, cc) // mq.WriteTo(cc)
			err = errors.New(string(v.Payload))
		}
		return err
	case *packets.Ping:
		// should not really happen here.
	default:
		fmt.Printf("I don't know about type mqtt %T!\n", v)
	}

	return nil
}

// localMakeMqttContact is a factory
func localMakeMqttContact(config *ContactStructConfig, tcpConn *net.TCPConn) *mqttContact {
	contact1 := &mqttContact{}
	AddContactStruct(&contact1.ContactStruct, contact1, config)
	contact1.netDotTCPConn = tcpConn
	contact1.realReader = tcpConn
	contact1.realWriter = tcpConn

	writer := func(mq libmqtt.Packet, cc *mqttContact) error {
		mq.SetVersion(cc.protoVersion)
		err := mq.WriteTo(cc)
		return err
	}
	contact1.writeLibPacket = writer

	return contact1
}

// func (cc *mqttWsContact) WriteDownstream(p packets.Interface) error {
// 	// var buff bytes.Buffer
// 	// cc.realWriter = &buff
// 	super := cc.mqttContact

// 	fmt.Println("ws writing ", p)

// 	err := super.WriteDownstream(p)
// 	if err != nil {
// 		cc.Close(err)
// 		return err
// 	}
// 	data := buff.Bytes()
// 	if len(data) > 0 {
// 		mt := websocket.BinaryMessage
// 		err = cc.wsConn.WriteMessage(mt, data)
// 		if err != nil {
// 			cc.Close(err)
// 			return err
// 		}
// 	}
// 	return nil
// }

// WebSocketLoop loops
func WebSocketLoop(wsConn *websocket.Conn, config *ContactStructConfig) {

	cc := &mqttWsContact{}
	cc.wsConn = wsConn
	AddContactStruct(&cc.ContactStruct, cc, config)
	cc.netDotTCPConn = nil
	cc.realReader = nil // set below.
	cc.realWriter = &cc.writebuff
	cc.writeLibPacket = func(mq libmqtt.Packet, ccx *mqttContact) error {

		mq.SetVersion(cc.protoVersion)

		//fmt.Println("writeLibPacket has version %n ", cc.protoVersion)

		cc.writebuff.Reset()
		err := mq.WriteTo(cc)
		if err != nil {
			cc.Close(err)
			return err
		}
		data := cc.writebuff.Bytes()
		if len(data) > 0 {
			mt := websocket.BinaryMessage
			//fmt.Println("collecting data len = %n ", len(data))
			// we need to get a lock here, it's rare but sometimes
			// we get a panic
			cc.writeAccessMutex.Lock()
			err = cc.wsConn.WriteMessage(mt, data)
			cc.writeAccessMutex.Unlock()
			if err != nil {
				cc.Close(err)
				return err
			}
		}
		return nil
	}

	var wsBuffer bytes.Buffer

	var readBuffer bytes.Buffer

	cc.realReader = &readBuffer

	defer wsConn.Close()
	for {
	top:
		//fmt.Println("waiting for mqtt ws packet")
		mt, message, err := wsConn.ReadMessage()
		if err != nil {
			fmt.Println("mqtt ws read err", err) // eg. websocket: close 1000 (normal)
			break
		}
		_ = mt
		wsBuffer.Write(message)
		// this REALLY stinks. They should only send WHOLE packets.
		// Or, we should hijack the tcp and wire it up directly.

		currentBytes := wsBuffer.Bytes()
		//fmt.Println("new currentBytes len = %n ", len(cc.writebuff.Bytes()))
		ok, plen := IsWholeMqttPacket(currentBytes)
		if !ok {
			goto top
		}
		extra := currentBytes[plen:]

		packetData := currentBytes[0:plen]
		//fmt.Println("got ws decoded packet", hex.EncodeToString(packetData))

		readBuffer.Reset()
		readBuffer.Write(packetData)

		wsBuffer.Reset()
		wsBuffer.Write(extra)

		cc.realReader = &readBuffer
		control, err := libmqtt.Decode(cc.protoVersion, cc)
		if err != nil {
			if err.Error() != "EOF" {
				fmt.Println("libmqtt.Decode err", control, err)
			}
			cc.Close(err)
			break // return
		}

		MQTTHandlePacket(&cc.mqttContact, control)
	}
	//fmt.Println("returned from loop ")
}

// IsWholeMqttPacket returns true if the data is an mqtt packet and returns the length used.
func IsWholeMqttPacket(data []byte) (bool, int) {

	i := 0
	if len(data) < 2 {
		return false, 0
	}
	i++ // pass the command
	length := 0
	shift := 0
	for { // get a variable size lenght
		tmp := data[i]
		i++
		if i >= len(data) {
			return false, 0
		}
		length = length + (int(tmp)&0x7F)<<shift
		shift += 7
		if tmp&0x80 == 0 {
			break
		}
	}
	// do we have all the data?
	i += length
	if len(data) < i {
		return false, 0
	}
	return true, i
}
