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

package tiers

// atw fix me this is unfinished

import (
	"fmt"
	"sync"

	"github.com/awootton/knotfreeiot/iot"
	"github.com/awootton/knotfreeiot/iot/reporting"
	"github.com/awootton/knotfreeiot/strprotocol"

	"strconv"

	"github.com/awootton/knotfreeiot/iot/gobprotocol"
)

// TwoByTwoTest is
func TwoByTwoTest() {

	// aaa and bbb will be downstream
	// ccc and ddd will be upstream

	aaasub := getSubscribeMgr()
	bbbsub := getSubscribeMgr()
	cccsub := getSubscribeMgr()
	dddsub := getSubscribeMgr()

	aaa := strprotocol.StartServerDemo(aaasub, "localhost:7374")
	bbb := strprotocol.StartServerDemo(bbbsub, "localhost:7375")

	gcc := gobprotocol.ServerOfGob(cccsub, "localhost:2000")
	gdd := gobprotocol.ServerOfGob(dddsub, "localhost:2001")

	// two clients from aaa to ccc and ddd
	// two clients from bbb to ccc and ddd

	aaaC := makeClient()
	// /	aaaC.addClient()

	_ = aaaC

	// and we need a splitter to direct traffic to ccc and ddd

	_ = aaasub
	_ = bbbsub

	_ = aaa
	_ = bbb

	_ = gcc
	_ = gdd
	var splitterA = func(t iot.HashType) *iot.SockStruct {
		return nil
	}
	aaasub.SetUpstreamSelector(splitterA)
	var splitterB = func(t iot.HashType) *iot.SockStruct {
		return nil
	}
	bbbsub.SetUpstreamSelector(splitterB)

}

func addClient(config *iot.SockStructConfig, addr string) {

}

func makeClient() *iot.SockStructConfig {

	pods := iot.NewSockStructConfig(nil)
	gobprotocol.ServerOfGobInit(pods)
	_ = pods

	aReportFunc := func(seconds float32) []string {
		strlist := make([]string, 0, 2)
		strlist = append(strlist, "len bob servers"+strconv.FormatUint(uint64(pods.Len()), 10))
		return strlist
	}
	reporting.NewGenericEventAccumulator(aReportFunc)
	go reporting.StartRunningReports()

	fmt.Println("gob start making clients")

	//	pods.SetCallback(runbucket)

	return nil
}

var subscribeMgr iot.PubsubIntf
var subscribeMgrMutex sync.Mutex

func getSubscribeMgr() iot.PubsubIntf {
	subscribeMgrMutex.Lock()
	if subscribeMgr == nil {
		subscribeMgr = iot.NewPubsubManager(100 * 1000)
	}
	subscribeMgrMutex.Unlock()
	return subscribeMgr
}
