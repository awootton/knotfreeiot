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

	"github.com/awootton/knotfreeiot/iot"
	"github.com/awootton/knotfreeiot/packets"
)

var globalClusterExec *iot.ClusterExecutive

// test auto scale in the minions and also reconnect when a minion is lost.
func TestExec(t *testing.T) {

	got := ""
	want := ""

	allContacts := make([]*testContact, 0)

	ce := iot.MakeSimplestCluster(getTime, testNameResolver)
	globalClusterExec = ce

	c1 := ce.GetNewContact(MakeTestContact)
	allContacts = append(allContacts, c1.(*testContact))
	SendText(c1, "S "+c1.String()) // subscribe to my name

	ct := c1.(*testContact)
	got = ct.getResultAsString() // // pause for a moment

	// there one in the aide and one in the guru
	got = fmt.Sprint("topics collected ", ce.GetSubsCount())
	want = "topics collected 2"
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	// add a contact a minute and see what happens.
	for i := 0; i < 100; i++ {
		ci := ce.GetNewContact(MakeTestContact)
		allContacts = append(allContacts, ci.(*testContact))
		SendText(ci, "S "+ci.String())
		currentTime += 60 // a minute
		ce.Operate()
	}

	got = ct.getResultAsString() // pause for a moment

	got = fmt.Sprint("topics collected ", ce.GetSubsCount())
	want = "topics collected 202"
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	fmt.Println("total minions", len(ce.Aides))

	// check that they all get messages
	for _, cc := range allContacts {
		command := "P " + cc.String() + " dalias saddr salias a_test_message" + cc.String()
		//fmt.Println(command)
		SendText(c1, command) // publish to cc from c1
	}
	got = ct.getResultAsString() // pause for a moment
	for i, cc := range allContacts {
		if i == 0 {
			continue // the first one has no message
		}
		got = "none"
		want = "a_test_message" + cc.String()
		p := cc.mostRecent
		if p != nil && reflect.TypeOf(p) == reflect.TypeOf(&packets.Send{}) {
			send := p.(*packets.Send)
			got = string(send.Payload)
			if got != want {
				t.Errorf("got %v, want %v", got, want)
			}
		}
	}

	// now. kill one of the minions and see if it reconnects and works
	i := 3
	minion := ce.Aides[i]
	ce.Aides[i] = ce.Aides[len(ce.Aides)-1] // Copy last element to index i.
	ce.Aides[len(ce.Aides)-1] = nil         // Erase last element (write zero value).
	ce.Aides = ce.Aides[:len(ce.Aides)-1]   // shorten list

	l := minion.Config.GetContactsList()
	e := l.Front()
	for ; e != nil; e = e.Next() {
		cc := e.Value.(iot.ContactInterface)
		cc.Close(errors.New("test close"))
	}

	for _, cc := range allContacts {
		command := "P " + cc.String() + " dalias saddr salias a_test_message2" + cc.String()
		//fmt.Println(command)
		SendText(c1, command) // publish to cc from c1
	}
	got = ct.getResultAsString() // pause for a moment

	for i, cc := range allContacts {
		if i == 0 {
			continue // the first one has no message
		}
		got = "none"
		want = "a_test_message2" + cc.String()
		p := cc.mostRecent
		if p != nil && reflect.TypeOf(p) == reflect.TypeOf(&packets.Send{}) {
			send := p.(*packets.Send)
			got = string(send.Payload)
		}
		if got != want {
			t.Errorf("got %v, want %v", got, want)
		}
	}

}
