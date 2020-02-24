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

package packets_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/awootton/knotfreeiot/packets"
)

func TestSend(t *testing.T) {

	got := ""
	want := ""

	cmd := packets.Send{}
	cmd.Source = []byte(`source`)
	cmd.Address = []byte("dest")
	cmd.Payload = []byte("some_data")

	var bb bytes.Buffer
	err := (&cmd).Write(&bb)
	_ = err

	got = hex.EncodeToString(bb.Bytes())
	want = `5005040006000964657374736f75726365736f6d655f64617461` // P followed by 5 strings, two are zero len.
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	uni, err := packets.ReadUniversal(&bb)

	bytes, err := packets.UniversalToJSON(uni)
	got = string(bytes)
	want = `[P,dest,,source,,some_data]`
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	got = cmd.String()
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	bb.Reset()
	err = (&cmd).Write(&bb)
	_ = err

	pack, err := packets.ReadPacket(&bb)
	bytes, err = pack.ToJSON()
	got = string(bytes)
	want = `[P,dest,,source,,some_data]`
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

}

func TestSub(t *testing.T) {

	got := "a"
	want := "b"

	cmd := packets.Subscribe{}
	cmd.Address = []byte("destination address")

	var bb bytes.Buffer
	err := (&cmd).Write(&bb)
	_ = err
	got = hex.EncodeToString(bb.Bytes())
	want = `5302130064657374696e6174696f6e2061646472657373`
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	uni, err := packets.ReadUniversal(&bb)

	bytes, err := packets.UniversalToJSON(uni)
	got = string(bytes)
	want = `[S,"destination address",]`
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	got = cmd.String()
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	bb.Reset()
	err = (&cmd).Write(&bb)
	_ = err

	pack, err := packets.ReadPacket(&bb)
	bytes, err = pack.ToJSON()
	got = string(bytes)
	want = `[S,"destination address",]`
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

}

func TestUnSub(t *testing.T) {

	got := "a"
	want := "b"

	cmd := packets.Unsubscribe{}
	cmd.Address = []byte("destination address")

	var bb bytes.Buffer
	err := (&cmd).Write(&bb)
	_ = err
	got = hex.EncodeToString(bb.Bytes())
	want = `5502130064657374696e6174696f6e2061646472657373`
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	uni, err := packets.ReadUniversal(&bb)

	bytes, err := packets.UniversalToJSON(uni) // ([]byte, error)
	got = string(bytes)
	want = `[U,"destination address",]`
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	got = cmd.String()
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	bb.Reset()
	err = (&cmd).Write(&bb)
	_ = err

	pack, err := packets.ReadPacket(&bb)
	bytes, err = pack.ToJSON()
	got = string(bytes)
	want = `[U,"destination address",]`
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestConnect(t *testing.T) {

	got := "a"
	want := "b"

	cmd := packets.Connect{}
	cmd.SetOption("key1", []byte("value1"))
	cmd.SetOption("key2", []byte("value2"))

	var bb bytes.Buffer
	err := (&cmd).Write(&bb)
	_ = err
	got = hex.EncodeToString(bb.Bytes())
	want = `4304040604066b65793176616c7565316b65793276616c756532` //
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	uni, err := packets.ReadUniversal(&bb)

	bytes, err := packets.UniversalToJSON(uni) // ([]byte, error)
	got = string(bytes)
	want = `[C,key1,value1,key2,value2]`
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	got = cmd.String()
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	bb.Reset()
	err = (&cmd).Write(&bb)
	_ = err

	pack, err := packets.ReadPacket(&bb)
	bytes, err = pack.ToJSON()
	got = string(bytes)
	want = `[C,key1,value1,key2,value2]`
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestDis(t *testing.T) {

	got := "a"
	want := "b"

	cmd := packets.Disconnect{}
	cmd.SetOption("key1", []byte("value1"))
	cmd.SetOption("key2", []byte("value2"))

	var bb bytes.Buffer
	err := (&cmd).Write(&bb)
	_ = err
	got = hex.EncodeToString(bb.Bytes())
	want = `4404040604066b65793176616c7565316b65793276616c756532` //
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	uni, err := packets.ReadUniversal(&bb)

	bytes, err := packets.UniversalToJSON(uni) // ([]byte, error)
	got = string(bytes)
	want = `[D,key1,value1,key2,value2]`
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	got = cmd.String()
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	bb.Reset()
	err = (&cmd).Write(&bb)
	_ = err

	pack, err := packets.ReadPacket(&bb)
	bytes, err = pack.ToJSON()
	got = string(bytes)
	want = `[D,key1,value1,key2,value2]`
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestLookup(t *testing.T) {

	got := "a"
	want := "b"

	cmd := packets.Lookup{}
	cmd.Address = []byte("look me up")
	cmd.Source = []byte("reply to me")

	var bb bytes.Buffer
	err := (&cmd).Write(&bb)
	_ = err
	got = hex.EncodeToString(bb.Bytes())
	want = `4c040a000b006c6f6f6b206d652075707265706c7920746f206d65` //
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	uni, err := packets.ReadUniversal(&bb)

	bytes, err := packets.UniversalToJSON(uni) // ([]byte, error)
	got = string(bytes)
	want = `[L,"look me up",,"reply to me",]`
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	got = cmd.String()
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	bb.Reset()
	err = (&cmd).Write(&bb)
	_ = err

	pack, err := packets.ReadPacket(&bb)
	bytes, err = pack.ToJSON()
	got = string(bytes)
	want = `[L,"look me up",,"reply to me",]`
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

type TestPacketStuff int

// ExampleTestPacketStuff is a test.
func ExampleTestPacketStuff() {

	examples := []int{0x200000 - 1, 10, 127, 128, 0x200, 0x4000 - 1, 0x4000, 0x432100}
	for _, val := range examples {

		var b bytes.Buffer // A Buffer needs no initialization.

		err := packets.WriteVarLenInt(uint32(val), uint8(0), &b)
		if err != nil {
			fmt.Println(err)
		}
		got, err := packets.ReadVarLenInt(&b)
		if err != nil {
			fmt.Println(err)
		}
		if got != val {
			fmt.Println(val, " vs ", got)
		}
	}
	{
		var b bytes.Buffer
		str := packets.Universal{}
		str.Cmd = 'A' // aka 65
		str.Args = [][]byte{[]byte("aa"), []byte("B"), []byte("cccccccccc")}

		err := str.Write(&b)
		if err != nil {
			fmt.Println(err)
		}
		str2, err := packets.ReadUniversal(&b)
		if len(str2.Args) != 3 {
			fmt.Println("len(str2.Args) != 3")
		}
		if string(str.Args[0]) != "aa" {
			fmt.Println("string(str.Args[0]) != aa")
		}
		if string(str.Args[1]) != "B" {
			fmt.Println("string(str.Args[1]) != B")
		}
		if string(str.Args[2]) != "cccccccccc" {
			fmt.Println("string(str.Args[2]) != cccccccccc")
		}
	}
	fmt.Println("done")

	// Output: done

}

type ToJSON int

// ExampleToJSON is a test as well as an example.
func ExampleToJSON() {
	cmd := packets.Send{}
	cmd.Source = []byte("sourceaddr")
	cmd.Address = []byte("destaddr")
	cmd.Payload = []byte("some data")
	cmd.SetOption("option1", []byte("test"))

	addr := "2001:0db8:85a3:0000:0000:8a2e:0370:7334"
	hexstr := strings.ReplaceAll(addr, ":", "")
	decoded, err := hex.DecodeString(hexstr)
	if err != nil {
		fmt.Println("wrong")
	}
	cmd.SetOption("IPv6", decoded)
	decoded, err = hex.DecodeString("FFFF00000000000000ABCDEF")
	if err != nil {
		fmt.Println("wrong2")
	}
	cmd.SetOption("z", decoded)
	cmd.SetOption("option2", []byte("На берегу пустынных волн"))

	jdata, err := (&cmd).ToJSON()
	fmt.Println(string(jdata))
	_ = err

	//  GetIPV6Option
	fmt.Println(cmd.GetIPV6Option())

	// Output: [P,destaddr,,sourceaddr,,"some data",IPv6,=IAENuIWjAAAAAIouA3BzNA,option1,test,option2,"На берегу пустынных волн",z,=//8AAAAAAAAAq83v]
	// [32 1 13 184 133 163 0 0 0 0 138 46 3 112 115 52]

}

func Test1(t *testing.T) {

	got := "a"
	want := "b"

	cmd := packets.Send{}
	cmd.Source = []byte(`source address "with" quotes`)
	cmd.SourceAlias = StandardAliasHash(cmd.Source)
	cmd.Address = []byte("$ the dest addr")
	cmd.AddressAlias = StandardAliasHash(cmd.Address)
	cmd.Address = []byte("")
	cmd.Payload = []byte("some_data")

	jdata, err := (&cmd).ToJSON()
	fmt.Println(string(jdata))
	_ = err

	got = string(jdata)
	want = `[P,,=/+7JC9UZPjNIsTRSthIW0CsYb6Hsx+mAv8rkKyddhcI,"source address \"with\" quotes",=Rf8QTqVX7vNRYxBqCWOXqKnAiDLa9unhyKQ6rTZLnG0,some_data]`
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func StandardAliasHash(longName []byte) []byte {
	h := sha256.New()
	h.Write(longName)
	return h.Sum(nil)
}

func TestForZombies(t *testing.T) {

	cmd := packets.Send{}
	val, ok := cmd.GetOption("key9")

	got := "notok"
	want := "notok"
	if ok == true {
		got = "ok"
	}
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	got = string(val)
	want = ""
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	cmd.SetOption("key1", []byte("val1"))
	val, ok = cmd.GetOption("key9")

	got = "notok"
	want = "notok"
	if ok == true {
		got = "ok"
	}
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	got = string(val)
	want = ""
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	val = cmd.GetIPV6Option()
	got = string(val)
	want = ""
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	var bb bytes.Buffer
	bb.Reset()
	bb.WriteString("")
	uni, err := packets.ReadUniversal(&bb)
	_ = uni
	_ = err
	got = err.Error()
	want = "EOF"
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	bb.Reset()
	bb.WriteString("P") // not enough data
	uni, err = packets.ReadUniversal(&bb)
	_ = uni
	_ = err
	got = err.Error()
	want = "EOF"
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	bb.Reset()
	bb.WriteString("P")
	bb.WriteByte(0x05) // still not enough
	uni, err = packets.ReadUniversal(&bb)
	_ = uni
	_ = err
	got = err.Error()
	want = "EOF"
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	bb.Reset()
	bytes, err := hex.DecodeString(`5005040006`)
	bb.Write(bytes) // still not enough
	uni, err = packets.ReadUniversal(&bb)
	_ = uni
	_ = err
	got = err.Error()
	want = "EOF"
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	bb.Reset()
	bytes, err = hex.DecodeString(`5085040006`)
	bb.Write(bytes) // too many args
	uni, err = packets.ReadUniversal(&bb)
	_ = uni
	_ = err
	got = err.Error()
	want = "Too many strings"
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	bb.Reset()
	bytes, err = hex.DecodeString(`500504000600`)
	bb.Write(bytes) // too few string lengths
	uni, err = packets.ReadUniversal(&bb)
	_ = uni
	_ = err
	got = err.Error()
	want = "EOF"
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	bb.Reset()
	bytes, err = hex.DecodeString(`500504000600`)
	bb.Write(bytes) // too few string lengths
	aPacket, err := packets.ReadPacket(&bb)
	_ = aPacket
	got = err.Error()
	want = "EOF"
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

}

func TestForZombies2(t *testing.T) {

	got := "notok"
	want := "notok"
	var bb bytes.Buffer

	bb.Reset()
	bytes, err := hex.DecodeString(`5005040006000964657374736f75726365736f6d655f646174`)
	bb.Write(bytes) // short by one byte
	uni, err := packets.ReadUniversal(&bb)
	_ = uni
	_ = err
	got = err.Error()
	want = "Too few bytes18 19"
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	bb.Reset()
	bytes, err = hex.DecodeString(`500504000600`)
	bb.Write(bytes) // too few string lengths
	aPacket, err := packets.ReadPacket(&bb)
	_ = aPacket
	got = err.Error()
	want = "EOF"
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	cmd := packets.Send{}
	cmd.Source = []byte(`source`)
	cmd.Address = []byte("dest")
	cmd.Payload = []byte("some_data")
	for i := 0; i < 65; i++ {
		val := "SomeTextSomeTextSomeTextSomeText"
		val = val + val
		val = val + val
		val = val + val
		cmd.SetOption("a very long key name"+strconv.Itoa(i), []byte(val))
	}
	bb.Reset()
	err = (&cmd).Write(&bb)
	got = err.Error()
	want = "Too many args"
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	cmd = packets.Send{}
	cmd.Source = []byte(`source`)
	cmd.Address = []byte("dest")
	cmd.Payload = []byte("some_data")
	for i := 0; i < 60; i++ {
		val := "SomeTextSomeTextSomeTextSomeText"
		val = val + val
		val = val + val
		val = val + val
		cmd.SetOption("a very long key name"+strconv.Itoa(i), []byte(val))
	}
	bb.Reset()
	err = (&cmd).Write(&bb)
	fmt.Println("buffer size", bb.Len())
	aPacket, err = packets.ReadPacket(&bb)
	_ = aPacket
	got = err.Error()
	want = "Packet too long for this reality"
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

}

//type TestPubSub1 int

// Just operate the pubsub without real tcp sockets.
// func ExampleTestPubSub1() {

// 	subscribeMgr := iot.NewPubsubManager(100)
// 	config := iot.NewSockStructConfig(subscribeMgr)

// 	config.SetCallback(func(ss *iot.SockStruct) {
// 		fmt.Println("not using sockets in this test")
// 	})

// 	config.SetClosecb(func(ss *iot.SockStruct, err error) {
// 		fmt.Println("not closing sockets in this test")
// 	})

// 	queue := make(chan *Send, 100)

// 	// the writer just stuffs the q and we'll check that later.
// 	config.SetWriter(func(ss *iot.SockStruct, topic []byte, topicAlias *iot.HashType, returnAddress []byte, returnAlias *iot.HashType, payload []byte) error {

// 		fmt.Println("have publish", string(topic))
// 		cmd := new(Send)
// 		cmd.Source = returnAddress
// 		cmd.Address = topic
// 		cmd.Payload = payload
// 		queue <- cmd

// 		return nil
// 	})

// 	conn1 := new(iotfakes.FakeConn)
// 	conn2 := new(iotfakes.FakeConn)

// 	//
// 	fauxSock := iot.NewSockStruct(net.Conn(conn1), config)
// 	fauxSock2 := iot.NewSockStruct(net.Conn(conn2), config)

// 	subscribeMgr.SendSubscriptionMessage(fauxSock, []byte("Topic1"))
// 	subscribeMgr.SendSubscriptionMessage(fauxSock2, []byte("Topic2"))

// 	subscribeMgr.SendPublishMessage(fauxSock2, []byte("Topic1"), []byte("message from 2 to 1"), []byte("Topic2"))

// 	mmm := <-queue
// 	fmt.Println(mmm)

// 	fauxSock.SetSelfAddress([]byte("Topic1"))
// 	fauxSock2.SetSelfAddress([]byte("Topic2"))

// 	subscribeMgr.SendPublishMessage(fauxSock2, []byte("Topic1"), []byte("message from 2 to 1 again"), []byte("Topic2xx"))
// 	mmm = <-queue
// 	fmt.Println(mmm)

// 	time.Sleep(time.Millisecond * 100)
// 	fmt.Println("done")

// 	// Output: have publish Topic1
// 	// {"args":[{"ascii":"Topic1"},{"ascii":""},{"ascii":"Topic2"},{"ascii":""},{"ascii":"message from 2 to 1"}],"cmd":"P"}
// 	// have publish Topic1
// 	// {"args":[{"ascii":"Topic1"},{"ascii":""},{"ascii":"Topic2xx"},{"ascii":""},{"ascii":"message from 2 to 1 again"}],"cmd":"P"}
// 	// done

// }

// var subscribeMgr iot.PubsubIntf
// var subscribeMgrMutex sync.Mutex

// func getSubscribeMgr() iot.PubsubIntf {
// 	subscribeMgrMutex.Lock()
// 	if subscribeMgr == nil {
// 		subscribeMgr = iot.NewPubsubManager(1000)
// 	}
// 	subscribeMgrMutex.Unlock()
// 	return subscribeMgr
// }
