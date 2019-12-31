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
	"fmt"
	"strconv"
	"time"

	"github.com/awootton/knotfreeiot/iot"
	"github.com/awootton/knotfreeiot/iot/reporting"
)

var sosClientRateDelay = time.Second * 30

// StartServerDemo is to start a server of the str protocol.
// Keep track of the return value or else call Close() on it.
func StartServerDemo(subscribeMgr iot.PubsubIntf) *iot.SockStructConfig {

	clientLogThing.SetQuiet(true)

	//	var subscribeMgr iot.PubsubIntf
	//	subscribeMgr = iot.NewPubsubManager(initialSize)

	config := ServerOfMqtt(subscribeMgr, ":1883")

	aReportFunc := func(seconds float32) []string {
		strlist := make([]string, 0, 1)
		count := config.Len()
		strlist = append(strlist, "mqtt_Conn count="+strconv.FormatUint(uint64(count), 10))
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

	addr := "knotfreeserver:1883"

	lights := iot.NewSockStructConfig(nil)
	ServerOfMqttInit(lights)
	switches := iot.NewSockStructConfig(nil)
	ServerOfMqttInit(switches)

	aReportFunc := func(seconds float32) []string {
		strlist := make([]string, 0, 2)
		strlist = append(strlist, "mqtt_Lights="+strconv.FormatUint(uint64(lights.Len()), 10))
		strlist = append(strlist, "mqtt_Switches="+strconv.FormatUint(uint64(switches.Len()), 10))
		return strlist
	}
	reporting.NewGenericEventAccumulator(aReportFunc)
	go reporting.StartRunningReports()

	fmt.Println("start making clients")

	//lights.SetCallback(runAlight)
	iot.MakeBunchOfClients(clientCount, addr, 10*time.Millisecond, lights, clientLogThing)

	fmt.Println("lights started")

	//switches.SetCallback(runAswitch)
	iot.MakeBunchOfClients(clientCount, addr, 10*time.Millisecond, switches, clientLogThing)

	fmt.Println("switches started")

	return lights, switches
}

var clientLogThing = reporting.NewStringEventAccumulator(16)
