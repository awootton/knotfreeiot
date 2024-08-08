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

package iot_test

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"testing"

	"github.com/awootton/knotfreeiot/iot"
	"github.com/awootton/knotfreeiot/packets"
	"github.com/awootton/knotfreeiot/tokens"

	_ "net/http/pprof"
)

func TestTwoLevel(t *testing.T) {

	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	tokens.LoadPublicKeys()

	atoken := tokens.GetTest32xToken()
	atokenStruct := tokens.ParseTokenNoVerify(atoken)

	got := ""
	want := ""
	ok := true
	var err error
	localtime := starttime
	getTime := func() uint32 {
		return localtime
	}
	// null ce
	var ce *iot.ClusterExecutive

	// set up
	guru0 := iot.NewExecutive(100, "guru0", getTime, true, ce)
	iot.GuruNameToConfigMap["guru0"] = guru0

	aide1 := iot.NewExecutive(100, "aide1", getTime, false, ce)
	aide2 := iot.NewExecutive(100, "aide2", getTime, false, ce)

	// we have to tell aides to connect to guru
	names := []string{"guru0"}
	aide1.Looker.SetUpstreamNames(names, names)
	aide2.Looker.SetUpstreamNames(names, names)
	WaitForActions(guru0)
	WaitForActions(aide1)
	WaitForActions(aide2)
	WaitForActions(guru0)
	// make a contact
	contact1 := makeTestContact(aide1.Config, "")

	contact2 := makeTestContact(aide2.Config, "")

	// note that they are in *different* lookups so normally they could not communicate but here we have a guru.

	// connect
	connect := packets.Connect{}
	connect.SetOption("token", atoken)
	iot.PushPacketUpFromBottom(contact1, &connect)
	iot.PushPacketUpFromBottom(contact2, &connect)

	// subscribe
	subs := packets.Subscribe{}
	subs.Address.FromString("contact1 address")
	err = iot.PushPacketUpFromBottom(contact1, &subs)

	WaitForActions(guru0)
	WaitForActions(aide1)
	WaitForActions(aide2)
	WaitForActions(guru0)

	got, _ = contact1.(*testContact).popResultAsString()
	got = strings.Replace(got, atokenStruct.JWTID, "xxxx", 1)
	want = "[S,=ygRnE97Kfx0usxBqx5cygy4enA1eojeR,jwtid,xxxx,pub2self,0]" //"no message received"
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	// val := readCounter(iot.TopicsAdded)
	// got = fmt.Sprint("topics collected ", val)
	// _ = got
	count, fract := guru0.GetSubsCount()
	_ = fract
	got = fmt.Sprint("topics collected ", count)
	want = "topics collected 1"
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	sendmessage := packets.Send{}
	sendmessage.Address.FromString("contact1 address")
	sendmessage.Source.FromString("contact2 address")
	sendmessage.Payload = []byte("can you hear me now?")

	iot.PushPacketUpFromBottom(contact2, &sendmessage)

	WaitForActions(guru0)
	WaitForActions(aide1)
	WaitForActions(aide2)
	WaitForActions(guru0)

	got, _ = contact1.(*testContact).popResultAsString()
	want = `[P,=ygRnE97Kfx0usxBqx5cygy4enA1eojeR,"contact2 address","can you hear me now?"]`
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	sendmessage2 := packets.Send{}
	sendmessage2.Address.FromString("contact1 address")
	sendmessage2.Source.FromString("contact2 address")
	sendmessage2.Payload = []byte("how about now?")

	iot.PushPacketUpFromBottom(contact2, &sendmessage2)

	WaitForActions(guru0) // FIXME: use IterateAndWait
	WaitForActions(aide1)
	WaitForActions(aide2)
	WaitForActions(guru0)

	got, _ = contact1.(*testContact).popResultAsString()
	want = `[P,=ygRnE97Kfx0usxBqx5cygy4enA1eojeR,"contact2 address","how about now?"]`
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	_ = got
	_ = want
	_ = err
	_ = ok

	WaitForActions(guru0)
	WaitForActions(aide1)
	WaitForActions(aide2)
	WaitForActions(guru0)

}

func TestSend(t *testing.T) {

	tokens.LoadPublicKeys()

	got := ""
	want := ""
	ok := true
	var err error
	localtime := starttime
	getTime := func() uint32 {
		return localtime
	}
	var ce *iot.ClusterExecutive

	// set up
	guru := iot.NewExecutive(100, "guru", getTime, true, ce)

	// make a contact
	contact1 := makeTestContact(guru.Config, "")

	contact2 := makeTestContact(guru.Config, "")

	connect := packets.Connect{}
	connect.SetOption("token", []byte(tokens.GetTest32xToken()))
	iot.PushPacketUpFromBottom(contact1, &connect)
	iot.PushPacketUpFromBottom(contact2, &connect)

	// subscribe
	subs := packets.Subscribe{}
	subs.Address.FromString("contact1_address")
	err = iot.PushPacketUpFromBottom(contact1, &subs)
	_ = err

	subs = packets.Subscribe{}
	subs.Address.FromString("contact2_address")
	err = iot.PushPacketUpFromBottom(contact2, &subs)

	WaitForActions(guru)

	// expect 2 sub acks
	got, _ = contact1.(*testContact).popResultAsString()
	want = "[S,=BAvjRqi8ESrF4XpR4ASFuojhyAOA_bpf,pub2self,0]"
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	got, _ = contact2.(*testContact).popResultAsString()
	want = "[S,=BAvjRqi8ESrF4XpR4ASFuojhyAOA_bpf,pub2self,0]"
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	//WaitForActions(guru)
	// IterateAndWait(t, func() bool {
	// 	WaitForActions(guru)
	// 	cval := readCounter(iot.TopicsAdded)
	// 	return cval > 2
	// }, "timed out waiting for topics collected to be 3")

	// val := readCounter(iot.TopicsAdded)
	// got = fmt.Sprint("topics collected ", val)
	count, fract := guru.GetSubsCount()
	_ = fract
	got = fmt.Sprint("topics collected ", count)
	want = "topics collected 2" //
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	sendmessage := packets.Send{}
	sendmessage.Address.FromString("contact1_address")
	sendmessage.Source.FromString("contact2_address")
	sendmessage.Payload = []byte("hello, can you hear me")

	iot.PushPacketUpFromBottom(contact2, &sendmessage)

	WaitForActions(guru)

	//"[P,=AMwu23hGtbsMhhqkKVsPgsWJ/PwPCFd24Q,contact2_address,"hello, ...+17 more"

	got, _ = contact1.(*testContact).popResultAsString()
	want = `[P,=zC7beEa1uwyGGqQpWw-CxYn8_A8IV3bh,contact2_address,"hello, can you hear me"]`
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	// FIXME: L has no test and doesn't work.
	// how do we test that it's there?
	// lookmsg := packets.Lookup{}
	// lookmsg.Address = []byte("contact1_address")
	// lookmsg.Source = []byte("contact2_address")
	// iot.PushPacketUpFromBottom(contact2, &lookmsg)

	// WaitForActions(guru)

	// // FIXME: L has no test and doesn't work.
	// got = contact2.(*testContact).getResultAsString()
	// want = `[L,contact1_address,=zC7beEa1uwyGGqQpWw+CxYn8/A8IV3bhYkAfKKktWv4,contact2_address,,count,1]`
	// if got != want {
	// 	t.Errorf("got %v, want %v", got, want)
	// }

	// unsub := packets.Unsubscribe{}
	// unsub.Address = []byte("contact1_address")
	// err = iot.PushPacketUpFromBottom(contact1, &unsub)

	// lookmsg = packets.Lookup{}
	// lookmsg.Address = []byte("contact1_address")
	// lookmsg.Source = []byte("contact2_address")
	// iot.PushPacketUpFromBottom(contact2, &lookmsg)

	// WaitForActions(guru)

	// got = contact2.(*testContact).getResultAsString()
	// // note that the count is ZERO
	// want = `[L,contact1_address,=zC7beEa1uwyGGqQpWw+CxYn8/A8IV3bhYkAfKKktWv4,contact2_address,,count,0]`
	// if got != want {
	// 	t.Errorf("got %v, want %v", got, want)
	// }

	_ = ok
	_ = err

}
