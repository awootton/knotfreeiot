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

package aaprotocol

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"time"

	"github.com/awootton/knotfreeiot/iot/reporting"

	"github.com/awootton/knotfreeiot/iot"
)

var aaClientRateDelay = time.Second * 30

// StartServerDemo is to start a server of the Aa protocol.
// Keep track of the return value or else call Close() on it.
func StartServerDemo(subscribeMgr iot.PubsubIntf) *iot.SockStructConfig {

	clientLogThing.SetQuiet(true)

	//	var subscribeMgr iot.PubsubIntf
	//	subscribeMgr = iot.NewPubsubManager(initialSize)

	config := ServerOfAa(subscribeMgr, ":6161")

	aReportFunc := func(seconds float32) []string {
		strlist := make([]string, 0, 1)
		count := config.Len()
		strlist = append(strlist, "aa_Conn count="+strconv.FormatUint(uint64(count), 10))
		return strlist
	}
	reporting.NewGenericEventAccumulator(aReportFunc)
	go reporting.StartRunningReports()

	return config
}

// StartClientsDemo return two sets of client sockets.
func StartClientsDemo(clientCount int) (*iot.SockStructConfig, *iot.SockStructConfig) {

	if clientCount > 10 {
		clientLogThing.SetQuiet(true)
	}

	addr := "knotfreeserver:6161"

	lights := iot.NewSockStructConfig(nil)
	ServerOfAaInit(lights)
	switches := iot.NewSockStructConfig(nil)
	ServerOfAaInit(switches)

	aReportFunc := func(seconds float32) []string {
		strlist := make([]string, 0, 2)
		strlist = append(strlist, "aa_Lights="+strconv.FormatUint(uint64(lights.Len()), 10))
		strlist = append(strlist, "aa_Switches="+strconv.FormatUint(uint64(switches.Len()), 10))
		return strlist
	}
	reporting.NewGenericEventAccumulator(aReportFunc)
	go reporting.StartRunningReports()

	fmt.Println("start making clients")

	lights.SetCallback(runAlight)
	iot.MakeBunchOfClients(clientCount, addr, 10*time.Millisecond, lights, clientLogThing)

	fmt.Println("lights started")

	switches.SetCallback(runAswitch)
	iot.MakeBunchOfClients(clientCount, addr, 10*time.Millisecond, switches, clientLogThing)

	fmt.Println("switches started")

	return lights, switches
}

// This is a client doing a very basic thing.
func runAlight(ss *iot.SockStruct) {

	done := false
	// start the reading thread.
	go func(ss *iot.SockStruct) {
		cmdReader := aaNewReader(ss.GetConn())
		//var recentTopic []byte
		for { // our reading loop
			obj, err := cmdReader.aaRead()
			if err != nil {
				ss.Close(err)
				done = true
				return
			}
			switch obj.(type) {
			case *setTopic:
				//recentTopic = obj.(*setTopic).msg
			case *publish:
				payload := obj.(*publish).msg
				// just pub/echo back to our switch
				myID := ss.GetSequence()
				// FIXME: we should use the returnAddress
				topic := "aaswitch_" + strconv.FormatUint(0x000FFFFF&myID, 16)
				//fmt.Println("pub to " + string(topic))
				hash := iot.HashType{}
				hash.FromBytes([]byte(topic))
				//fmt.Println("pub to ", hash.GetA()&0x0FFFF)
				returnAddr := []byte("none") //ss.GetSelfAddress()
				err := HandleTopicPayload(ss, []byte(topic), nil, returnAddr, nil, payload)
				if err != nil {
					ss.Close(err)
					done = true
					return
				}
			case *subscribe:
				// why would the server send us a subscribe?
			case *unsubscribe:
				// why would the server send us an unsubscribe?
			case *ping:
				// if we pinged then this is it coming back.
			case *aaError:
				// Server sent us an error. Then it closed us
				str := "got aaError=" + string(obj.(*aaError).msg)
				aaLogThing.Collect(str)
				ss.Close(err)
				return
			default:
				// Server sent us junk somehow
				str := "bad aa type=" + reflect.TypeOf(obj).String()
				err := errors.New(str)
				aaLogThing.Collect(str)
				ss.Close(err)
				return
			}
		}
	}(ss)

	myID := ss.GetSequence()
	idstr := strconv.FormatUint(0x000FFFFF&myID, 16)
	topic := "aalight_" + idstr
	ss.SetSelfAddress([]byte(topic))
	s := subscribe{[]byte(topic)}

	hash := iot.HashType{}
	hash.FromBytes([]byte(topic))
	//fmt.Println("sub to ", hash.GetA()&0x0FFFF)

	aaWrite(ss, &s)

	for !done {
		// our sending loop, but the light doesn't send.
		time.Sleep(time.Second)
	}
}

func runAswitch(ss *iot.SockStruct) {

	myID := ss.GetSequence()
	idstr := strconv.FormatUint(0x000FFFFF&myID, 16)

	ourCommand := "Hello From Switch_" + idstr

	done := false
	when := time.Now()

	// start the reading thread.
	go func(ss *iot.SockStruct) {
		cmdReader := aaNewReader(ss.GetConn())
		//var recentTopic []byte
		for { // our reading loop
			obj, err := cmdReader.aaRead()
			if err != nil {
				done = true
				ss.Close(err)
				return
			}
			switch obj.(type) {
			case *setTopic:
				//recentTopic = obj.(*setTopic).msg
			case *publish:
				payload := obj.(*publish).msg
				if string(payload) == ourCommand {
					// we got the message back from the light
					duration := time.Now().Sub(when)
					// log it in buckets:
					if duration < time.Millisecond*100 {
						clientLogThing.Collect("aa happy joy") // under 100 ms
					} else if duration < time.Second {
						clientLogThing.Collect("aa ok") // under one sec
					} else {
						clientLogThing.Collect("aa too slow") // everything else
					}
				}
			case *subscribe:
				// why would the server send us a subscribe?
			case *unsubscribe:
				// why would the server send us an unsubscribe?
			case *ping:
				// if we pinged then this is it coming back.
			case *aaError:
				// Server sent us an error. Then it closed us
				str := "got aaError=" + string(obj.(*aaError).msg)
				aaLogThing.Collect(str)
				ss.Close(err)
				return
			default:
				// Server sent us junk somehow
				str := "bad aa type=" + reflect.TypeOf(obj).String()
				err := errors.New(str)
				aaLogThing.Collect(str)
				ss.Close(err)
				return
			}
		}
	}(ss)

	topic := "aaswitch_" + idstr
	// now convert to command
	//fmt.Println("ssub to " + string(topic))
	ss.SetSelfAddress([]byte(topic))
	cmd := subscribe{[]byte(topic)}

	hash := iot.HashType{}
	hash.FromBytes([]byte(topic))
	//fmt.Println("ssub to ", hash.GetA()&0x0FFFF)

	aaWrite(ss, &cmd)

	for !done {
		// our sending loop
		topic := "aalight_" + idstr
		when = time.Now()
		//fmt.Println("spub to " + string(topic))

		hash := iot.HashType{}
		hash.FromBytes([]byte(topic))
		//fmt.Println("spub to ", hash.GetA()&0x0FFFF)
		returnAddr := []byte("none") //ss.GetSelfAddress()
		err := HandleTopicPayload(ss, []byte(topic), nil, returnAddr, nil, []byte(ourCommand))

		if err != nil {
			done = true
		}
		time.Sleep(aaClientRateDelay) // 10 * time.Second)
	}
}

var clientLogThing = reporting.NewStringEventAccumulator(16)
