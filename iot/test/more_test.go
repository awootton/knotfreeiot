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
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"testing"

	"github.com/awootton/knotfreeiot/iot"
	"github.com/awootton/knotfreeiot/packets"
	"github.com/awootton/knotfreeiot/tokens"
)

var globalClusterExec *iot.ClusterExecutive

func TestGrowGurus(t *testing.T) {

	tokens.LoadPublicKeys()

	got := ""
	want := ""
	localtime := starttime
	getTime := func() uint32 {
		return localtime
	}
	subsStressSize := 100

	ce := iot.MakeSimplestCluster(getTime, testNameResolver, false, 1)
	globalClusterExec = ce

	stats := ce.Aides[0].GetExecutiveStats()
	bytes, _ := json.Marshal(stats)
	fmt.Println(string(bytes))

	c1 := ce.GetNewContactFromSlackestAide(MakeTestContact, "")
	SendText(c1, "S "+c1.String()) // subscribe to my name

	c2 := ce.GetNewContactFromSlackestAide(MakeTestContact, "")
	SendText(c2, "S "+c2.String()) // subscribe to my name

	WaitForActions(ce.Aides[0])

	c1test := c1.(*testContact)
	got = c1test.getResultAsString() // always nil
	c2test := c2.(*testContact)
	got = c2test.getResultAsString() // always nil MakeTestContact prodices auto popping clien

	// there one in the aide and one in the guru
	got = fmt.Sprint("topics collected ", ce.GetSubsCount())
	want = "topics collected 6"
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	// add a subscription a minute and see what happens.
	for i := 0; i < subsStressSize; i++ {
		cmd := "S " + c1.String() + "_" + strconv.FormatInt(int64(i), 10)
		//fmt.Println("sub cmd", cmd)
		SendText(c1, cmd)
		localtime += 60 // a minute
		ce.Operate()
	}

	WaitForActions(ce.Aides[0])

	//fmt.Println("c1 has ", c1test.mostRecent)

	got = fmt.Sprint("guru count ", len(ce.Gurus))
	want = "guru count 2"
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	fmt.Println("total minions", len(ce.Aides))

	// check that they all get messages after the expansion
	for i := 0; i < subsStressSize; i++ {
		command := "P " + c1.String() + "_" + strconv.FormatInt(int64(i), 10) + " x x x a_test_message"
		//fmt.Println(command)
		SendText(c2, command) // publish to c1 from c2
	}
	WaitForActions(ce.Aides[0])
	got = c1test.getResultAsString()
	for i := 0; i < subsStressSize; i++ {

		got = "none"
		want = "a_test_message"
		p := c1test.mostRecent
		if len(p) != 0 && reflect.TypeOf(p[0]) == reflect.TypeOf(&packets.Send{}) {
			send := p[0].(*packets.Send)
			got = string(send.Payload)
		} else {
			fmt.Println("expected Send, got ", p)
		}
		if len(c1test.mostRecent) > 0 {
			//fmt.Println("popping", c1test.mostRecent[0])
			c1test.mostRecent = c1test.mostRecent[1:]
		}
		if got != want {
			fmt.Println("no most recent", i)
			t.Errorf("got %v, want %v", got, want)
		}
	}

	// delete a subscription a minute and see what happens.
	for i := 4; i < subsStressSize; i++ {
		cmd := "U " + c1.String() + "_" + strconv.FormatInt(int64(i), 10)
		//fmt.Println("cmd", cmd)
		SendText(c1, cmd)
		localtime += 60 // a minute
		ce.Operate()
	}
	WaitForActions(ce.Aides[0])
	ce.Operate()
	subsStressSize = 4

	got = fmt.Sprint("guru count ", len(ce.Gurus))
	want = "guru count 1"
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	fmt.Println("total minions", len(ce.Aides))

}

// test auto scale in the minions and also reconnect when a minion is lost.
func TestExec(t *testing.T) {

	tokens.LoadPublicKeys()

	got := ""
	want := ""
	localtime := starttime
	getTime := func() uint32 {
		return localtime
	}

	contactStressSize := 50

	allContacts := make([]*testContact, 0)

	ce := iot.MakeSimplestCluster(getTime, testNameResolver, false, 1)
	globalClusterExec = ce

	c1 := ce.GetNewContactFromSlackestAide(MakeTestContact, "")
	allContacts = append(allContacts, c1.(*testContact))
	SendText(c1, "S "+c1.String()) // subscribe to my name

	ce.WaitForActions()

	// there one in the aide and one in the guru
	got = fmt.Sprint("topics collected ", ce.GetSubsCount())
	want = "topics collected 4"
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	// add a contact a minute and see what happens.
	for i := 0; i < contactStressSize; i++ {
		ci := ce.GetNewContactFromSlackestAide(MakeTestContact, "")
		allContacts = append(allContacts, ci.(*testContact))
		index := len(allContacts)
		ci.(*testContact).index = index
		indexstr := strconv.FormatInt(int64(index), 10)
		cmd := "S contactTopic" + indexstr
		//fmt.Println("sub cmd1", cmd)
		SendText(ci, cmd) //ci.String())
		localtime += 60   // a minute
		ce.Operate()
	}

	ce.WaitForActions()

	got = fmt.Sprint("topics collected ", ce.GetSubsCount())
	want = "topics collected 107" // + strconv.FormatInt(int64(contactStressSize*3+2), 10)
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	fmt.Println("total minions", len(ce.Aides)) // 4

	// check that they all get messages
	for i, cc := range allContacts {
		if i == 0 {
			continue // the first one has no message
		}
		index := strconv.FormatInt(int64(cc.index), 10)
		command := "P " + "contactTopic" + index + " x x x a_test_message_" + index
		//fmt.Println("sending 4", command)
		SendText(c1, command) // publish to cc from c1
	}

	ce.WaitForActions()

	for i, cc := range allContacts {
		if i == 0 {
			continue // the first one has no message
		}
		index := strconv.FormatInt(int64(cc.index), 10)
		got = "none"
		want = "a_test_message_" + index
		p := cc.mostRecent
		if len(p) != 0 && reflect.TypeOf(p[0]) == reflect.TypeOf(&packets.Send{}) {
			send := p[0].(*packets.Send)
			got = string(send.Payload)
		} else {
			fmt.Println("expected Send, got ", reflect.TypeOf(p), p)
		}
		if len(cc.mostRecent) > 0 {
			//fmt.Println("popping", cc.mostRecent[0])
			cc.mostRecent = cc.mostRecent[1:]
		}

		if got != want {
			fmt.Println("no most recent", i, cc)
			t.Errorf("got %v, want %v", got, want)
		}
	}

	ce.WaitForActions()

	// now. kill one of the minions and see if it reconnects and works
	if true {
		i := 3
		minion := ce.Aides[i]
		ce.Aides[i] = ce.Aides[len(ce.Aides)-1] // Copy last element to index i.
		ce.Aides[len(ce.Aides)-1] = nil         // Erase last element (write zero value).
		ce.Aides = ce.Aides[:len(ce.Aides)-1]   // shorten list

		l := minion.Config.GetContactsList()
		e := l.Front()
		for ; e != nil; e = e.Next() {
			cc := e.Value.(*testContact)
			cc.Close(errors.New("test close"))
		}
	}

	ce.WaitForActions()

	// is the guru connected?

	fmt.Println("check 1")

	for i, cc := range allContacts {
		if i == 0 {
			continue // the first one has no message
		}
		index := strconv.FormatInt(int64(cc.index), 10)
		command := "P " + "contactTopic" + index + " x x x a_test_message2_" + index
		//fmt.Println("pub2", command)
		SendText(c1, command) // publish to cc from c1
	}

	ce.WaitForActions()
	ce.WaitForActions()

	fmt.Println("check 2")

	for i, cc := range allContacts {
		if i == 0 {
			continue // the first one has no message
		}
		index := cc.index
		got = "none"
		want = "a_test_message2_" + strconv.FormatInt(int64(index), 10)
		p := cc.mostRecent
		if len(p) != 0 && reflect.TypeOf(p[0]) == reflect.TypeOf(&packets.Send{}) {
			send := p[0].(*packets.Send)
			got = string(send.Payload)
		} else {
			fmt.Println("i expected Send, got ", cc.mostRecent, want)
		}
		if len(cc.mostRecent) > 0 {
			if i > 42 {
				//fmt.Println("popping", cc.mostRecent[0])
			} else {
				//fmt.Println("popping", cc.mostRecent[0])
			}
			cc.mostRecent = cc.mostRecent[1:]
		}
		if got != want {
			fmt.Println("i no most recent", index, cc)
			t.Errorf("got %v, want %v", got, want)

		}
	}

	fmt.Println("check 3")
	// close all the contacts and the aides should shrink
	for i, cc := range allContacts {
		if i == 0 {
			continue // the first one has no message
		}
		cc.doNotReconnect = true
		cc.Close(errors.New("test"))
		localtime += 60 // a minute
		ce.Operate()
		ce.WaitForActions()
	}
	ce.Operate()
	ce.WaitForActions()
	ce.Operate()
	ce.WaitForActions()

	got = fmt.Sprint("total minions ", len(ce.Aides))
	want = "total minions 2"
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

}
