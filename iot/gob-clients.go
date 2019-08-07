package iot

import (
	"fmt"

	"knotfreeiot/iot/reporting"
	"time"
)

// StartServerDemo is to start a server of the Aa protocol.
// Keep track of the return value or else call Close() on it.
// func StartServerDemo(initialSize int) *SockStructConfig {

// 	clientLogThing.SetQuiet(true)

// 	var subscribeMgr PubsubIntf
// 	subscribeMgr = NewPubsubManager(initialSize)

// 	config := SockStructConfig{}
// 	// aReportFunc := func(seconds float32) []string {
// 	// 	strlist := make([]string, 0, 1)
// 	// 	count := config.Len()
// 	// 	strlist = append(strlist, "aa_Conn count="+strconv.FormatUint(uint64(count), 10))
// 	// 	return strlist
// 	// }
// 	// reporting.NewGenericEventAccumulator(aReportFunc)
// 	// go reporting.StartRunningReports()

// 	return &config
// }

// StartClientsDemo return two sets of client sockets.
func StartClientsDemo(clientCount int) (*SockStructConfig, *SockStructConfig) {

	if clientCount > 10 {
		clientLogThing.SetQuiet(true)
	}

	addr := "knotfreeserver:6161"

	lights := NewSockStructConfig(nil)
	//	ServerOfAaInit(lights)
	_ = lights
	switches := NewSockStructConfig(nil)
	//	ServerOfAaInit(switches)
	_ = switches

	aReportFunc := func(seconds float32) []string {
		strlist := make([]string, 0, 2)
		//	strlist = append(strlist, "aa_Lights="+strconv.FormatUint(uint64(lights.Len()), 10))
		//	strlist = append(strlist, "aa_Switches="+strconv.FormatUint(uint64(switches.Len()), 10))
		return strlist
	}
	reporting.NewGenericEventAccumulator(aReportFunc)
	go reporting.StartRunningReports()

	fmt.Println("gob start making clients")

	//lights.SetCallback(runAlight)
	MakeBunchOfClients(clientCount, addr, 10*time.Millisecond, lights, clientLogThing)

	fmt.Println("gob  lights started")

	//	switches.SetCallback(runAswitch)
	MakeBunchOfClients(clientCount, addr, 10*time.Millisecond, switches, clientLogThing)

	fmt.Println("gob switches started")

	return lights, switches
}

var clientLogThing = reporting.NewStringEventAccumulator(16)
