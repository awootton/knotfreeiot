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

package str2protocol

// The following is an example of a Str2 client.

import (
	"bufio"
	"fmt"
	"knotfreeiot/iot"
	"knotfreeiot/iot/reporting"
	"strconv"
	"time"
)

var sosClientRateDelay = time.Second * 30

// StartServerDemo is to start a server of the str protocol.
// Keep track of the return value or else call Close() on it.
func StartServerDemo(subscribeMgr iot.PubsubIntf, address string) *iot.SockStructConfig {

	clientLogThing.SetQuiet(true)

	//	var subscribeMgr iot.PubsubIntf
	//	subscribeMgr = iot.NewPubsubManager(initialSize)

	config := ServerOfStrings(subscribeMgr, address) // "knotfree:"+strconv.Itoa(port))

	aReportFunc := func(seconds float32) []string {
		strlist := make([]string, 0, 1)
		count := config.Len()
		strlist = append(strlist, "str_Conn count="+strconv.FormatUint(uint64(count), 10))
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

	addr := "knotfreeserver:7374"

	lights := iot.NewSockStructConfig(nil)
	ServerOfStringsInit(lights)
	switches := iot.NewSockStructConfig(nil)
	ServerOfStringsInit(switches)

	aReportFunc := func(seconds float32) []string {
		strlist := make([]string, 0, 2)
		strlist = append(strlist, "str_Lights="+strconv.FormatUint(uint64(lights.Len()), 10))
		strlist = append(strlist, "str_Switches="+strconv.FormatUint(uint64(switches.Len()), 10))
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

func runAlight(ss *iot.SockStruct) {

	done := false
	// start the reading thread.
	go func(ss *iot.SockStruct) {
		reader := bufio.NewReader(ss.GetConn())
		for { // our reading loop
			text, err := reader.ReadString('\n')
			if err != nil {
				ss.Close(err)
				done = true
				return
			}
			str := text[0 : len(text)-1]
			//fmt.Println("Light received", str)
			cmd, payload := GetFirstWord(str)
			if cmd == "pub" {
				// just pub back to switch
				topicfrom, payload := GetFirstWord(payload)
				myID := ss.GetSequence()
				topic := "strswitch_" + strconv.FormatUint(0x000FFFFF&myID, 16)

				command := "pub " + topic + " " + payload
				ServerOfStringsWrite(ss, command)
				_ = topicfrom
			} else {
				// it's just the echo of out own pub and sub commands.
				// fmt.Println("not handled", cmd, payload)
			}
		}
	}(ss)

	myID := ss.GetSequence()
	idstr := strconv.FormatUint(0x000FFFFF&myID, 16)

	topic := "strlight_" + idstr
	ss.SetSelfAddress([]byte(topic))
	// now convert to command
	cmd := "sub " + topic
	ServerOfStringsWrite(ss, cmd) // send subscribe command to server

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
		reader := bufio.NewReader(ss.GetConn())
		for { // our reading loop
			text, err := reader.ReadString('\n')
			if err != nil {
				done = true
				ss.Close(err)
				return
			}
			str := text[0 : len(text)-1] // remove the \n
			cmd, tmp := GetFirstWord(str)
			topic, payload := GetFirstWord(tmp)
			if cmd == "pub" {
				if payload == ourCommand {
					// we got the message back from the light
					duration := time.Now().Sub(when)
					// log it in buckets:
					if duration < time.Millisecond*100 {
						clientLogThing.Collect("str happy joy") // under 100 ms
					} else if duration < time.Second {
						clientLogThing.Collect("str ok") // under one sec
					} else {
						clientLogThing.Collect("str too slow") // everything else
					}
				}
			}
			_ = topic
		}
	}(ss)

	topic := "strswitch_" + idstr
	// now convert to command
	cmd := "sub " + topic
	ServerOfStringsWrite(ss, cmd) // send subscribe command to se

	for !done {
		// our sending loop
		topic := "strlight_" + idstr
		command := "pub " + topic + " " + ourCommand
		when = time.Now()
		err := ServerOfStringsWrite(ss, command) // send pub command to server
		if err != nil {
			done = true
		}
		time.Sleep(sosClientRateDelay) // 10 * time.Second)
	}
}

var clientLogThing = reporting.NewStringEventAccumulator(16)
