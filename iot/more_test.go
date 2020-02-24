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
	"strconv"
	"testing"

	"github.com/awootton/knotfreeiot/iot"
	"github.com/awootton/knotfreeiot/packets"
	"github.com/awootton/knotfreeiot/tokens"
	"github.com/gbrlsnchs/jwt/v3"
)

var globalClusterExec *iot.ClusterExecutive

func TestGrowGurus(t *testing.T) {

	tokens.SavePublicKey("1iVt", string(tokens.GetSamplePublic()))

	got := ""
	want := ""

	subsStressSize := 100

	ce := iot.MakeSimplestCluster(getTime, testNameResolver)
	globalClusterExec = ce

	c1 := ce.GetNewContact(MakeTestContact)
	SendText(c1, "S "+c1.String()) // subscribe to my name

	c2 := ce.GetNewContact(MakeTestContact)
	SendText(c2, "S "+c2.String()) // subscribe to my name

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
		//fmt.Println("cmd", cmd)
		SendText(c1, cmd)
		currentTime += 60 // a minute
		ce.Operate()
	}

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
		got = fmt.Sprint(c1test.mostRecent)
		//fmt.Println("received", got)

	}
	got = c1test.getResultAsString() // pause for a moment
	for i := 0; i < subsStressSize; i++ {

		got = "none"
		want = "a_test_message"
		p := c1test.mostRecent
		if len(p) != 0 && reflect.TypeOf(p[0]) == reflect.TypeOf(&packets.Send{}) {
			send := p[0].(*packets.Send)
			got = string(send.Payload)
		} else {
			fmt.Println("expected Send, got ", reflect.TypeOf(p[0]))
		}
		if len(c1test.mostRecent) > 0 {
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
		currentTime += 60 // a minute
		ce.Operate()
	}
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

	tokens.SavePublicKey("1iVt", string(tokens.GetSamplePublic()))

	got := ""
	want := ""

	contactStressSize := 50

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
	want = "topics collected 4"
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	// add a contact a minute and see what happens.
	for i := 0; i < contactStressSize; i++ {
		ci := ce.GetNewContact(MakeTestContact)
		allContacts = append(allContacts, ci.(*testContact))
		SendText(ci, "S "+ci.String())
		currentTime += 60 // a minute
		ce.Operate()
	}

	got = ct.getResultAsString() // pause for a moment

	got = fmt.Sprint("topics collected ", ce.GetSubsCount())
	want = "topics collected 107" // + strconv.FormatInt(int64(contactStressSize*3+2), 10)
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	fmt.Println("total minions", len(ce.Aides)) // 4

	// check that they all get messages
	for _, cc := range allContacts {
		command := "P " + cc.String() + " x x x a_test_message" + cc.String()
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
		if len(p) != 0 && reflect.TypeOf(p[0]) == reflect.TypeOf(&packets.Send{}) {
			send := p[0].(*packets.Send)
			got = string(send.Payload)

		} else {
			fmt.Println("expected Send, got ", reflect.TypeOf(p))

		}
		if len(cc.mostRecent) > 0 {
			cc.mostRecent = cc.mostRecent[1:]
		}
		if got != want {
			fmt.Println("no most recent", i, cc)
			t.Errorf("got %v, want %v", got, want)
		}
	}

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
			cc := e.Value.(iot.ContactInterface)
			cc.Close(errors.New("test close"))
		}
	}

	for _, cc := range allContacts {
		command := "P " + cc.String() + " x x x a_test_message2" + cc.String()
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
		if len(p) != 0 && reflect.TypeOf(p[0]) == reflect.TypeOf(&packets.Send{}) {
			send := p[0].(*packets.Send)
			got = string(send.Payload)
			cc.mostRecent = cc.mostRecent[1:]
		} else {
			fmt.Println("i expected Send, got ", reflect.TypeOf(p))
		}
		if got != want {
			fmt.Println("i no most recent", i, cc)
			t.Errorf("got %v, want %v", got, want)

		}
	}

	for i, cc := range allContacts {
		if i == 0 {
			continue // the first one has no message
		}
		cc.doNotReconnect = true
		cc.Close(errors.New("test"))
		currentTime += 60 // a minute
		ce.Operate()
	}
	ce.Operate()

	got = fmt.Sprint("total minions", len(ce.Aides))
	want = "total minions1"
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

}

// 123480 ns/op	    1248 B/op	      22 allocs/op  	~8000/sec
func BenchmarkCheckToken(b *testing.B) {
	ticket := []byte("eyJhbGciOiJFZDI1NTE5IiwidHlwIjoiSldUIn0.eyJleHAiOjE2MDk0NjI4MDAsImlzcyI6IjFpVnQiLCJqdGkiOiIxMjM0NTYiLCJpbiI6NzAwMDAsIm91dCI6NzAwMDAsInN1IjoyLCJjbyI6Mn0.N22xJiYz_FMQu_nG_cxlQk7gnvbeO9zOiuzbkZYWpxSzAPtQ_WyCVwWYBPZtA-0Oj-AggWakTNsmGoe8JIzaAg")
	publicKey := tokens.GetSamplePublic()
	// run the verify function b.N times
	for n := 0; n < b.N; n++ {

		p, ok := tokens.VerifyTicket(ticket, publicKey)
		_ = p
		_ = ok

	}
}

// this is not especially quick
// 122662 ns/op	    1088 B/op	      19 allocs/op 	~8000/sec
func BenchmarkCheckToken2(b *testing.B) {
	ticket := []byte("eyJhbGciOiJFZDI1NTE5IiwidHlwIjoiSldUIn0.eyJleHAiOjE2MDk0NjI4MDAsImlzcyI6IjFpVnQiLCJqdGkiOiIxMjM0NTYiLCJpbiI6NzAwMDAsIm91dCI6NzAwMDAsInN1IjoyLCJjbyI6Mn0.N22xJiYz_FMQu_nG_cxlQk7gnvbeO9zOiuzbkZYWpxSzAPtQ_WyCVwWYBPZtA-0Oj-AggWakTNsmGoe8JIzaAg")
	publicKey := tokens.GetSamplePublic()
	payload := tokens.KnotFreePayload{}
	algo := jwt.NewEd25519(jwt.Ed25519PublicKey(publicKey))

	// run the verify function b.N times
	for n := 0; n < b.N; n++ {

		hd, err := jwt.Verify([]byte(ticket), algo, &payload)
		_ = hd
		_ = err
		if payload.Connections != 2 {
			fmt.Println("wrong")
		}
		payload.Connections = -1

	}
}
