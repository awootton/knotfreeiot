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
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/awootton/knotfreeiot/iot"
	"github.com/awootton/knotfreeiot/packets"
	"github.com/prometheus/client_golang/prometheus"

	dto "github.com/prometheus/client_model/go"
)

const starttime = uint32(1577840400) // Wednesday, January 1, 2020 1:00:00 AM

type testContact struct {
	iot.ContactStruct

	downMessages chan timedPacket
}

type testUpperContact struct {
	iot.ContactStruct

	//downMessages chan packets.Interface
	bridgeBottomContact *testContact
}

type timedPacket struct {
	packet    packets.Interface
	timestamp uint32
}

// called by Lookup PushUp
func (cc *testUpperContact) WriteUpstream(cmd packets.Interface, timestamp uint32) {
	// call the Push
	iot.Push(cc.bridgeBottomContact, cmd, timestamp)
}

func TestTwoLevel(t *testing.T) {

	got := ""
	want := ""
	ok := true
	var err error

	// set up
	mgrTop0 := iot.NewLookupTable(100)
	configTop0 := iot.NewContactStructConfig(mgrTop0)

	mgr1 := iot.NewLookupTable(100)
	config1 := iot.NewContactStructConfig(mgr1)

	mgr2 := iot.NewLookupTable(100)
	config2 := iot.NewContactStructConfig(mgr2)

	mgr1.NameResolver = func(name string, config *iot.ContactStructConfig) (iot.ContactInterface, error) {
		if name == "top0" { // todo: better names.
			// IRL this is a tcp connect with contactTop1 as tcp client
			// this is the contect that mgr1 and mgr2 will be using at the top
			contactTop1 := testUpperContact{}
			//contactTop1.downMessages = make(chan packets.Interface, 1000)
			iot.InitUpperContactStruct(&contactTop1.ContactStruct, config)
			// This is the one attaching to the bottom of mgrTop0
			// this work would be done after the socket accept
			newLowerContact := testContact{}
			newLowerContact.downMessages = make(chan timedPacket, 1000)
			iot.AddContactStruct(&newLowerContact.ContactStruct, configTop0)

			// wire them up
			contactTop1.bridgeBottomContact = &newLowerContact
			go func() {
				cmd := <-newLowerContact.downMessages
				fmt.Println("cmd moving down", cmd)
				iot.PushDown(&contactTop1, cmd.packet, cmd.timestamp)
			}()

			return &contactTop1, nil
		} else {
			return &testUpperContact{}, errors.New("unknown name " + name)
		}
	}
	mgr2.NameResolver = mgr1.NameResolver
	// we have to tell mgr1 to connect to mgrTop1
	// send it an array of 1024 strings
	var names [1024]string
	for i := range names {
		names[i] = "top0"
	}
	mgr1.SetUpstreamNames(names)
	mgr2.SetUpstreamNames(names)

	// make a contact
	contact1 := testContact{}
	contact1.downMessages = make(chan timedPacket, 1000)
	iot.AddContactStruct(&contact1.ContactStruct, config1)
	// another
	contact2 := testContact{}
	contact2.downMessages = make(chan timedPacket, 1000)
	iot.AddContactStruct(&contact2.ContactStruct, config2)
	// note that they are in different lookups

	// subscribe
	subs := packets.Subscribe{}
	subs.Address = []byte("contact1 address")
	err = iot.Push(&contact1, &subs, starttime)

	got = contact1.getResultAsString()
	want = "no message received<nil>"
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	val := readCounter(iot.TopicsAdded)
	got = fmt.Sprint("topics collected ", val)
	// want = "topics collected 2"
	// if got != want {
	// 	t.Errorf("got %v, want %v", got, want)
	// }

	sendmessage := packets.Send{}
	sendmessage.Address = []byte("contact1 address")
	sendmessage.Source = []byte("contact2 address")
	sendmessage.Payload = []byte("can you hear me now?")

	iot.Push(&contact2, &sendmessage, starttime)

	got = contact1.getResultAsString()
	want = `[P,"contact1 address",=ygRnE97Kfx0usxBqx5cygy4enA1eojeRWdV/XMwSGzw,"contact2 address",,"can you hear me now?"]`
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	_ = got
	_ = want
	_ = config1
	_ = config2
	_ = configTop0
	_ = err
	_ = ok

	//	time.Sleep(100 * time.Second)

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
	contact1.downMessages = make(chan timedPacket, 1000)
	iot.AddContactStruct(&contact1.ContactStruct, config)
	// another
	contact2 := testContact{}
	contact2.downMessages = make(chan timedPacket, 1000)
	iot.AddContactStruct(&contact2.ContactStruct, config)

	// subscribe
	subs := packets.Subscribe{}
	subs.Address = []byte("contact1 address")
	err = iot.Push(&contact1, &subs, starttime)
	subs = packets.Subscribe{}
	subs.Address = []byte("contact2 address")
	err = iot.Push(&contact2, &subs, starttime)

	got = contact1.getResultAsString()
	want = "no message received<nil>"
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	val := readCounter(iot.TopicsAdded)
	got = fmt.Sprint("topics collected ", val)
	want = "topics collected 2"
	// if got != want {
	// 	t.Errorf("got %v, want %v", got, want)
	// }

	sendmessage := packets.Send{}
	sendmessage.Address = []byte("contact1 address")
	sendmessage.Source = []byte("contact2 address")
	sendmessage.Payload = []byte("hello, can you hear me")

	iot.Push(&contact2, &sendmessage, starttime)

	got = contact1.getResultAsString()
	want = `[P,"contact1 address",=ygRnE97Kfx0usxBqx5cygy4enA1eojeRWdV/XMwSGzw,"contact2 address",,"hello, can you hear me"]`
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	// how do we test that it's there?
	lookmsg := packets.Lookup{}
	lookmsg.Address = []byte("contact1 address")
	lookmsg.Source = []byte("contact2 address")
	iot.Push(&contact2, &lookmsg, starttime)

	got = contact2.getResultAsString()
	want = `[L,"contact1 address",=ygRnE97Kfx0usxBqx5cygy4enA1eojeRWdV/XMwSGzw,"contact2 address",,count,1]`
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	unsub := packets.Unsubscribe{}
	unsub.Address = []byte("contact1 address")
	err = iot.Push(&contact1, &unsub, starttime)

	lookmsg = packets.Lookup{}
	lookmsg.Address = []byte("contact1 address")
	lookmsg.Source = []byte("contact2 address")
	iot.Push(&contact2, &lookmsg, starttime)

	got = contact2.getResultAsString()
	// note that the count is ZERO
	want = `[L,"contact1 address",=ygRnE97Kfx0usxBqx5cygy4enA1eojeRWdV/XMwSGzw,"contact2 address",,count,0]`
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	_ = ok
	_ = err

}

func (cc *testContact) get() (packets.Interface, bool) {
	select {
	case msg := <-cc.downMessages:
		return msg.packet, true
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

func (cc *testContact) WriteDownstream(cmd packets.Interface, timestamp uint32) {
	//fmt.Println("received from above", cmd, reflect.TypeOf(cmd))
	cc.downMessages <- timedPacket{cmd, timestamp}
}

func (cc *testContact) WriteUpstream(cmd packets.Interface, timestamp uint32) {
	fmt.Println("FIXME received from below", cmd, reflect.TypeOf(cmd))
	//cc.downMessages <- cmd
}

func readCounter(m prometheus.Counter) float64 {
	pb := &dto.Metric{}
	m.Write(pb)
	return pb.GetCounter().GetValue()
}
