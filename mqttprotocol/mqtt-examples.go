package mqttprotocol

import (
	"fmt"
	"knotfreeiot/iot"
	"knotfreeiot/iot/reporting"
	"strconv"
	"time"
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
