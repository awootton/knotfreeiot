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
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"time"

	"knotfreeiot/iot"
	"knotfreeiot/iot/reporting"
)

// UpstreamConn will reference a pod!
// they will all have increasing sequence numbers.
// we'll keep them in order and ... help. todo. fixme.
type UpstreamConn struct {
	sequence uint32
	address  string
	//port     int

	ss *iot.SockStruct
}

// StartClient connects to a server and uses it as an upstream provider.
func StartClient(currentList []UpstreamConn) *iot.SockStructConfig {

	//	addr := "knotfreeserver:6161"

	pods := iot.NewSockStructConfig(nil)
	ServerOfGobInit(pods)
	_ = pods

	aReportFunc := func(seconds float32) []string {
		strlist := make([]string, 0, 2)
		strlist = append(strlist, "len bob servers"+strconv.FormatUint(uint64(pods.Len()), 10))
		return strlist
	}
	reporting.NewGenericEventAccumulator(aReportFunc)
	go reporting.StartRunningReports()

	fmt.Println("gob start making clients")

	pods.SetCallback(runbucket)
	// 	MakeBunchOfClients(clientCount, addr, 10*time.Millisecond, lights, clientLogThing)

	// two clients from aaa to ccc and ddd
	// two clients from bbb to ccc and ddd

	fmt.Println("gob started")

	return pods
}

func runbucket(ss *iot.SockStruct) {

	done := false
	// implement the protocol
	// start the reading thread.
	go func(ss *iot.SockStruct) {
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
					_ = item
					//	ss.SendPublishMessage(item.topic, item.payload, item.returnAddress)
				}
			case *SubscribeMessage:
				sub := obj.(*SubscribeMessage)
				for _, item := range *sub {
					_ = item
					//	ss.SendSubscriptionMessage(item.topic)
				}
			case *UnsubscribeMessage:
				sub := obj.(*UnsubscribeMessage)
				for _, item := range *sub {
					_ = item
					//	ss.SendSubscriptionMessage(item.topic)
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
	}(ss)

	myID := ss.GetSequence()
	idstr := strconv.FormatUint(0x000FFFFF&myID, 16)
	topic := "aalight_" + idstr
	ss.SetSelfAddress([]byte(topic))
	//	s := subscribe{[]byte(topic)}

	hash := iot.HashType{}
	hash.FromBytes([]byte(topic))
	//fmt.Println("sub to ", hash.GetA()&0x0FFFF)

	//	a Write(ss, &s)

	for !done {
		// our sending loop, but the light doesn't send.
		time.Sleep(time.Second)
	}

}

var clientLogThing = reporting.NewStringEventAccumulator(16)
