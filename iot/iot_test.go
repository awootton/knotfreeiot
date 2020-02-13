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

package iot_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/awootton/knotfreeiot/iot"
	"github.com/awootton/knotfreeiot/packets"
)

type testContact struct {
	iot.ContactStruct

	downMessages chan packets.Interface
}

func (cc *testContact) get() (packets.Interface, bool) {
	select {
	case msg := <-cc.downMessages:
		return msg, true
	case <-time.After(10 * time.Millisecond):
		return nil, false
	}
}

func (cc *testContact) getResultAsString() string {
	gotmsg, ok := cc.get()
	got := ""
	if ok {
		got = gotmsg.String()
	} else {
		got = fmt.Sprint("no message received", gotmsg)
	}
	return got
}

func (cc *testContact) WriteDownstream(cmd packets.Interface) {
	//fmt.Println("received from above", cmd, reflect.TypeOf(cmd))
	cc.downMessages <- cmd
}

func TestSend(t *testing.T) {

	got := ""
	want := ""
	ok := true
	var err error

	// set up
	mgr := iot.NewLookupTable(100)
	config := iot.NewContactStructConfig(mgr)

	// make a contact
	contact1 := testContact{}
	contact1.downMessages = make(chan packets.Interface, 1000)
	iot.AddContactStruct(&contact1.ContactStruct, config)
	// another
	contact2 := testContact{}
	contact2.downMessages = make(chan packets.Interface, 1000)
	iot.AddContactStruct(&contact2.ContactStruct, config)

	// subscribe
	subs := packets.Subscribe{}
	subs.Address = []byte("contact1 address")
	err = iot.Push(&contact1, &subs)
	subs = packets.Subscribe{}
	subs.Address = []byte("contact2 address")
	err = iot.Push(&contact2, &subs)

	got = contact1.getResultAsString()
	want = "no message received<nil>"
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	sendmessage := packets.Send{}
	sendmessage.Address = []byte("contact1 address")
	sendmessage.Source = []byte("contact2 address")
	sendmessage.Payload = []byte("hello, can you hear me")

	iot.Push(&contact2, &sendmessage)

	got = contact1.getResultAsString()
	want = `[P,"contact1 address",=ygRnE97Kfx0usxBqx5cygy4enA1eojeRWdV/XMwSGzw,"contact2 address",,"hello, can you hear me"]`
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	// how do we test that it's there?
	lookmsg := packets.Lookup{}
	lookmsg.Address = []byte("contact1 address")
	lookmsg.Source = []byte("contact2 address")
	iot.Push(&contact2, &lookmsg)

	got = contact2.getResultAsString()
	want = `[L,"contact1 address",=ygRnE97Kfx0usxBqx5cygy4enA1eojeRWdV/XMwSGzw,"contact2 address",,count,1]`
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	unsub := packets.Unsubscribe{}
	unsub.Address = []byte("contact1 address")
	err = iot.Push(&contact1, &unsub)

	lookmsg = packets.Lookup{}
	lookmsg.Address = []byte("contact1 address")
	lookmsg.Source = []byte("contact2 address")
	iot.Push(&contact2, &lookmsg)

	got = contact2.getResultAsString()
	want = `[L,"contact1 address",=ygRnE97Kfx0usxBqx5cygy4enA1eojeRWdV/XMwSGzw,"contact2 address",,count,0]`
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	_ = ok
	_ = err

}
