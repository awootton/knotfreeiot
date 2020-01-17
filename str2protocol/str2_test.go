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

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/awootton/knotfreeiot/iot"
	"github.com/awootton/knotfreeiot/iot/iotfakes"
)

type TestPacketStuff int

// ExampleTestPacketStuff is a test.
func ExampleTestPacketStuff() {

	examples := []int{0x200000 - 1, 10, 127, 128, 0x200, 0x4000 - 1, 0x4000, 0x432100}
	for _, val := range examples {

		var b bytes.Buffer // A Buffer needs no initialization.

		err := WriteVarLenInt(uint32(val), uint8(0), &b)
		if err != nil {
			fmt.Println(err)
		}
		got, err := ReadVarLenInt(&b)
		if err != nil {
			fmt.Println(err)
		}
		if got != val {
			fmt.Println(val, " vs ", got)
		}
	}
	{
		var b bytes.Buffer
		str := Str2{}
		str.cmd = 'A' // aka 65
		str.args = [][]byte{[]byte("aa"), []byte("B"), []byte("cccccccccc")}

		err := str.Write(&b)
		if err != nil {
			fmt.Println(err)
		}
		str2, err := ReadStr2(&b)
		if len(str2.args) != 3 {
			fmt.Println("len(str2.args) != 3")
		}
		if string(str.args[0]) != "aa" {
			fmt.Println("string(str.args[0]) != aa")
		}
		if string(str.args[1]) != "B" {
			fmt.Println("string(str.args[1]) != B")
		}
		if string(str.args[2]) != "cccccccccc" {
			fmt.Println("string(str.args[2]) != cccccccccc")
		}
	}
	fmt.Println("done")

	// Output: done

}

type ToJSON int

// ExampleToJSON is a test as well as an example.
func ExampleToJSON() {
	cmd := Send{}
	cmd.source = []byte("sourceaddr")
	cmd.address = []byte("destaddr")
	cmd.payload = []byte("some data")
	cmd.options = make(map[string][]byte)
	cmd.options["option1"] = []byte("test")
	addr := "2001:0db8:85a3:0000:0000:8a2e:0370:7334"
	hexstr := strings.ReplaceAll(addr, ":", "")
	decoded, err := hex.DecodeString(hexstr)
	if err != nil {
		fmt.Println("wrong")
	}
	cmd.options["ip"] = decoded
	decoded, err = hex.DecodeString("FFFF00000000000000ABCDEF")
	if err != nil {
		fmt.Println("wrong2")
	}
	cmd.options["z"] = decoded
	cmd.options["option2"] = []byte("На берегу пустынных волн")

	jdata, err := (&cmd).ToJSON()
	// the ToJSON iterates a map and that is not deterministic.
	fmt.Println(string(jdata))
	_ = err

	//  GetIPV6Option
	fmt.Println(cmd.GetIPV6Option())

	// {"args":[{"ascii":"destaddr"},{"ascii":""},{"ascii":"sourceaddr"},{"ascii":""},{"ascii":"some data"},{"ascii":"option1"},{"ascii":"test"},{"ascii":"ip"},{"b64":"IAENuIWjAAAAAIouA3BzNA"},{"ascii":"z"},{"b64":"//8AAAAAAAAAq83v"},{"ascii":"option2"},{"utf8":"На берегу пустынных волн"}],"cmd":"P"}
	// [32 1 13 184 133 163 0 0 0 0 138 46 3 112 115 52]

}

type TestPubSub1 int

// Just operate the pubsub without real tcp sockets.
func ExampleTestPubSub1() {

	subscribeMgr := iot.NewPubsubManager(100)
	config := iot.NewSockStructConfig(subscribeMgr)

	config.SetCallback(func(ss *iot.SockStruct) {
		fmt.Println("not using sockets in this test")
	})

	config.SetClosecb(func(ss *iot.SockStruct, err error) {
		fmt.Println("not closing sockets in this test")
	})

	queue := make(chan *Send, 100)

	// the writer just stuffs the q and we'll check that later.
	config.SetWriter(func(ss *iot.SockStruct, topic []byte, topicAlias *iot.HashType, returnAddress []byte, returnAlias *iot.HashType, payload []byte) error {

		fmt.Println("have publish", string(topic))
		cmd := new(Send)
		cmd.source = returnAddress
		cmd.address = topic
		cmd.payload = payload
		queue <- cmd

		return nil
	})

	conn1 := new(iotfakes.FakeConn)
	conn2 := new(iotfakes.FakeConn)

	//
	fauxSock := iot.NewSockStruct(net.Conn(conn1), config)
	fauxSock2 := iot.NewSockStruct(net.Conn(conn2), config)

	subscribeMgr.SendSubscriptionMessage(fauxSock, []byte("Topic1"))
	subscribeMgr.SendSubscriptionMessage(fauxSock2, []byte("Topic2"))

	subscribeMgr.SendPublishMessage(fauxSock2, []byte("Topic1"), []byte("message from 2 to 1"), []byte("Topic2"))

	mmm := <-queue
	fmt.Println(mmm)

	fauxSock.SetSelfAddress([]byte("Topic1"))
	fauxSock2.SetSelfAddress([]byte("Topic2"))

	subscribeMgr.SendPublishMessage(fauxSock2, []byte("Topic1"), []byte("message from 2 to 1 again"), []byte("Topic2xx"))
	mmm = <-queue
	fmt.Println(mmm)

	time.Sleep(time.Millisecond * 100)
	fmt.Println("done")

	// Output: have publish Topic1
	// {"args":[{"ascii":"Topic1"},{"ascii":""},{"ascii":"Topic2"},{"ascii":""},{"ascii":"message from 2 to 1"}],"cmd":"P"}
	// have publish Topic1
	// {"args":[{"ascii":"Topic1"},{"ascii":""},{"ascii":"Topic2xx"},{"ascii":""},{"ascii":"message from 2 to 1 again"}],"cmd":"P"}
	// done

}

var subscribeMgr iot.PubsubIntf
var subscribeMgrMutex sync.Mutex

func getSubscribeMgr() iot.PubsubIntf {
	subscribeMgrMutex.Lock()
	if subscribeMgr == nil {
		subscribeMgr = iot.NewPubsubManager(1000)
	}
	subscribeMgrMutex.Unlock()
	return subscribeMgr
}
