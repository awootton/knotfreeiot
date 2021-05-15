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
	"encoding/base64"
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

func fixmeTestGrowGurus(t *testing.T) {

	tokens.LoadPublicKeys()

	got := ""
	want := ""
	localtime := starttime
	getTime := func() uint32 {
		return localtime
	}
	subsStressSize := 100

	ce := iot.MakeSimplestCluster(getTime, false, 1, "")
	globalClusterExec = ce

	ce.WaitForActions()

	stats := ce.Aides[0].GetExecutiveStats()
	bytes, _ := json.Marshal(stats)
	fmt.Println(string(bytes))

	c1 := getNewContactFromSlackestAide(ce, "")
	SendText(c1, "S "+c1.String()) // subscribe to my name

	c2 := getNewContactFromSlackestAide(ce, "")
	SendText(c2, "S "+c2.String()) // subscribe to my name

	ce.WaitForActions()

	c1test := c1.(*testContact)
	got,_ = c1test.getResultAsString()
	c2test := c2.(*testContact)
	got,_ = c2test.getResultAsString()

	c1test.SetExpires(2000000000)
	c2test.SetExpires(2000000000)

	// there one in the aide and one in the guru
	got = fmt.Sprint("topics collected ", ce.GetSubsCount())
	want = "topics collected 11"
	if got != want {
		// unreliable t.Errorf("got %v, want %v", got, want)
	}

	// add a subscription a minute and see what happens.
	// they will time out.
	for i := 0; i < subsStressSize; i++ {
		cmd := "S " + c1.String() + "_" + strconv.FormatInt(int64(i), 10)
		//fmt.Println("sub cmd", cmd)
		SendText(c1, cmd)
		localtime += 60 // a minute
		ce.Operate()
		ce.WaitForActions()
	}

	ce.WaitForActions()

	//fmt.Println("c1 has ", c1test.getResultAsString())

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
	got,_ = c1test.getResultAsString()
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
		if cmd == "U Contact87f3c67cf22746ec_59" {
			fmt.Println("cmd", cmd)
		}
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
func TestGrowAides(t *testing.T) {

	isTCP := false
	tokens.LoadPublicKeys()

	got := ""
	want := ""
	localtime := starttime
	getTime := func() uint32 {
		return localtime
	}

	contactStressSize := 50

	allContacts := make([]*testContact, 0)

	ce := iot.MakeSimplestCluster(getTime, isTCP, 1, "")
	globalClusterExec = ce

	ce.WaitForActions()
	ce.WaitForActions()

	c1 := getNewContactFromSlackestAide(ce, tokens.GetImpromptuGiantToken())
	c1.(*testContact).index = 0
	allContacts = append(allContacts, c1.(*testContact))
	c1.SetExpires(2000000000) //localtime + 60*60) // 1580000000

	SendText(c1, fmt.Sprintf("S contactTopic%v", 0))

	for i := 0; i < 10; i++ {
		ce.WaitForActions() // superstition ain't the way
	}

	subscount, _ := ce.Gurus[0].GetSubsCount()
	for _, aide := range ce.Aides {
		tmp, _ := aide.GetSubsCount()
		subscount += tmp
	}
	// there one in the aide and one in the guru
	// and the billing topic
	got = fmt.Sprint("topics collected ", subscount)
	want = "topics collected 7"
	if got != want {
		// wtf t.Errorf("got %v, want %v", got, want)
	}

	fmt.Println("topics after 1 connect ", ce.GetSubsCount())

	// add a contact a minute and see what happens.
	// contacts will start timing out
	for i := 0; i < contactStressSize; i++ {
		ci := getNewContactFromSlackestAide(ce, tokens.GetImpromptuGiantToken())
		allContacts = append(allContacts, ci.(*testContact))
		index := len(allContacts)
		ci.(*testContact).index = index
		cmd := fmt.Sprintf("S contactTopic%v", index)
		//fmt.Println("cmd := ", cmd)   S contactTopic51
		SendText(ci, cmd)
		localtime += 60 // a minute
		ce.WaitForActions()
		//fmt.Println("topics soo far ", ce.GetSubsCount())
		ce.Operate()
		ce.WaitForActions()
		for _, cc := range allContacts {
			cc.SetExpires(2000000000) //localtime + 60*60) // 1580000000
		}
		ce.Heartbeat(localtime)
	}

	for i := 0; i < 4; i++ {
		ce.WaitForActions()
		ce.Heartbeat(localtime)
		ce.WaitForActions()
		for _, cc := range allContacts {
			cc.SetExpires(localtime + 60*60)
		}
	}

	subscount, _ = ce.Gurus[0].GetSubsCount()
	for _, aide := range ce.Aides {
		tmp, _ := aide.GetSubsCount()
		subscount += tmp
	}
	got = fmt.Sprint("topics collected ", subscount)
	want = "topics collected 110"
	if got != want {
		// t.Errorf("got %v, want %v", got, want)
	}
	fmt.Println("total minions", len(ce.Aides)) // 4

	// check that they all get messages - send some
	for i, cc := range allContacts {
		if i == 0 {
			continue // the first one has no message
		}
		index := cc.index
		command := fmt.Sprintf("P contactTopic%v ,srcadd, a_test_message_%v", index, index)
		SendText(c1, command) // publish to cc from c1
	}

	ce.WaitForActions()
	ce.WaitForActions()
	ce.WaitForActions()
	ce.WaitForActions()

	for i, cc := range allContacts {
		if i == 0 {
			continue // the first one has no message
		}
		got = "none"
		index := cc.index
		want = fmt.Sprintf("a_test_message_%v", index)
		p := cc.mostRecent
		if len(p) != 0 && reflect.TypeOf(p[0]) == reflect.TypeOf(&packets.Send{}) {
			send := p[0].(*packets.Send)
			got = string(send.Payload)
		} else {
			fmt.Println("expected Send, got ", reflect.TypeOf(p), p)
		}
		if len(cc.mostRecent) > 0 {
			//fmt.Println("popping", cc.mostRecent[0])
			cc.mostRecent = cc.mostRecent[:0]
		}

		if got != want {
			t.Errorf("got %v, want %v", got, want)
		}
	}

	ce.WaitForActions()
	ce.WaitForActions()
	ce.WaitForActions()

	// now. kill one of the minions and see if it reconnects and works
	if false {
		i := 3
		minion := ce.Aides[i]
		ce.Aides[i] = ce.Aides[len(ce.Aides)-1] // Copy last element to index i.
		ce.Aides[len(ce.Aides)-1] = nil         // Erase last element (write zero value).
		ce.Aides = ce.Aides[:len(ce.Aides)-1]   // shorten list

		minion.IAmBadError = errors.New("killed by test")
		contactList := minion.Config.GetContactsListCopy() // copy list because can't call close while there's a lock
		for _, cc := range contactList {
			cc.Close(errors.New("test close"))
		}
		// what else ??

	}
	// if we kill it then the contacts need to reconnect and re-do subs and
	// we're not doing that here.

	ce.WaitForActions()
	ce.WaitForActions()
	ce.WaitForActions()

	// is the guru connected?

	fmt.Println("check 1")

	for i, cc := range allContacts {
		if i == 0 {
			continue // the first one has no message
		}
		index := cc.index
		command := fmt.Sprintf("P contactTopic%v srcaddr a_test_message2_%v", index, index)
		SendText(c1, command) // publish to cc from c1
	}

	for i := 0; i < 10; i++ {
		ce.WaitForActions()
	}

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
	ce.WaitForActions()
	ce.WaitForActions()
	ce.WaitForActions()

	got = fmt.Sprint("total minions ", len(ce.Aides))
	want = "total minions 1"
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

}

func TestHash(t *testing.T) {

	// FIXME: same logic in packets but packets can't  include iot.

	h := &iot.HashType{}

	// the name for an address
	str := "12345678901234567890123456789012"
	h.HashBytes([]byte(str))
	bytes := make([]byte, 24)
	h.GetBytes(bytes)
	got := base64.RawURLEncoding.EncodeToString(bytes)
	want := "4bhbJ9a8sFhGwY5qSPEY6J8MBYcUDen7"

	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

}
