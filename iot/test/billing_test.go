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
	"testing"

	"github.com/awootton/knotfreeiot/iot"
	"github.com/awootton/knotfreeiot/tokens"
)

// this token comes from tokens.TestMakeToken1connection
// One connection, one subscription.
var sampleToken1 = `eyJhbGciOiJFZERTQSIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2OTc5MzE2NjMsImlzcyI6Il85c2giLCJqdGkiOiIxMjM0NTYiLCJpbiI6MjAsIm91dCI6MjAsInN1IjoxLCJjbyI6MSwidXJsIjoia25vdGZyZWUubmV0In0.W6dYgJvedMuolrYXCYzQauaAynu80bmX3Qtq5lSACGxvaro6tqttnGozxXBDVzHO5IYzut9vb5Yi9i-ThCwfBA`

func TestSubscriptionOverrun(t *testing.T) {

	tokens.LoadPublicKeys()

	got := ""
	want := ""
	localtime := starttime
	getTime := func() uint32 {
		return localtime
	}

	// the token above allows for 1 contact and 1 sub we'll make just one
	// it will post every 5 minutes for an hour and then go quiet.
	// it should get closed.

	// allContacts := make([]*testContact, 0)

	// make cluster with 1 guru and 1 aides.
	// don't call operate or it will lose an aide.
	ce := iot.MakeSimplestCluster(getTime, false, 1, "")
	globalClusterExec = ce
	aide1 := ce.Aides[0]

	c1 := getNewContactFromAide(aide1, sampleToken1)
	// llContacts = append(allContacts, c1.(*testContact))
	c1.SetExpires(localtime + 60*60*60) // an hour

	// c2 := getNewContactFromAide(aide1, sampleToken1)
	// allContacts = append(allContacts, c2.(*testContact))

	SendText(c1, "S "+c1.String()+" debg 12345678")          // subscribe to my name
	SendText(c1, "S "+"someotherchannel "+" debg 12345678")  // subscribe to another channel
	SendText(c1, "S "+"someotherchannel2 "+" debg 12345678") // subscribe to another channel

	//	SendText(c2, "S "+c1.String()+" debg 12345678") // subscribe to c1's name

	// c1.(*testContact).doNotReconnect = true
	// c2.(*testContact).doNotReconnect = true

	ce.WaitForActions()

	ok := true

	for seconds := 0; seconds < 20*60; seconds++ {
		localtime += 1
		ce.Heartbeat(localtime)
		if seconds%10 == 1 {
			ce.WaitForActions()
		}
		c1.SetExpires(localtime + 60*60) // an hour
		got, ok = c1.(*testContact).popResultAsString()
		if ok {
			fmt.Println("got", got)
			break
		}
	}
	if got == "" {
		IterateAndWait(t, func() bool {
			got, ok := c1.(*testContact).popResultAsString()
			fmt.Println("c1 got", got)
			return ok
		}, "timed out waiting for TestSubscriptionOverrun result")
	}
	fmt.Println("got", got)
	fmt.Println("subscriptions. aide1", aide1.GetExecutiveStats().Subscriptions*float64(aide1.GetExecutiveStats().Limits.Subscriptions))

	got, _ = c1.(*testContact).popResultAsString()
	want = `[P,=jZae727K08KaOmKSgOaGzww_XVqGr_PK,ping," BILLING ERROR 2.9 subscriptions > 2",error," BILLING ERROR 2.9 subscriptions > 2"]`
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	_ = got
	_ = want
}

func TestContactTimeout(t *testing.T) {

	tokens.LoadPublicKeys()

	got := ""
	want := ""
	localtime := starttime
	getTime := func() uint32 {
		return localtime
	}

	// the token above allows for 1 contact and we'll make just one
	// it will post every 5 minutes for an hour and then go quiet.
	// it should get closed.

	// allContacts := make([]*testContact, 0)

	// make cluster with 1 guru and 2 aides.
	// don't call operate or it will lose an aide.
	ce := iot.MakeSimplestCluster(getTime, false, 1, "")
	globalClusterExec = ce
	aide1 := ce.Aides[0]

	c1 := getNewContactFromAide(aide1, sampleToken1)
	// allContacts = append(allContacts, c1.(*testContact))
	SendText(c1, "S "+c1.String()) // subscribe to my name

	c1.(*testContact).doNotReconnect = true

	ce.WaitForActions()

	fmt.Println("contacts aide1", aide1.GetExecutiveStats().Connections*float64(aide1.GetExecutiveStats().Limits.Connections))

	for seconds := 0; seconds < 20*60; seconds += 10 {
		localtime += 10
		ce.Heartbeat(localtime)
		if seconds%300 == 1 {
			SendText(c1, "S "+c1.String()) // subscribe to my name, again
		}
		if seconds%10 == 1 {
			ce.WaitForActions()
		}
	}
	SendText(c1, "S "+c1.String()) // subscribe to my name, again
	ce.WaitForActions()

	got = fmt.Sprint("contacts aide1 ", aide1.GetExecutiveStats().Connections*float64(aide1.GetExecutiveStats().Limits.Connections))
	want = "contacts aide1 2.94"
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	fmt.Println("minutes passed", (localtime-starttime)/60)

	// now just wait for a while for c1 to get kicked off
	for seconds := 0; seconds < 18*60; seconds += 10 {
		localtime += 10
		ce.Heartbeat(localtime)

		ce.WaitForActions()

	}
	fmt.Println("minutes passed", (localtime-starttime)/60)

	got = fmt.Sprint("contacts aide1 ", aide1.GetExecutiveStats().Connections*float64(aide1.GetExecutiveStats().Limits.Connections))
	want = "contacts aide1 2.97"
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	// now just wait for a while for c1 to get kicked off
	for minutes := 0; minutes < 40; minutes++ {
		localtime += 60
		ce.Heartbeat(localtime)
		ce.WaitForActions()
	}

	got = fmt.Sprint("contacts aide1 ", aide1.GetExecutiveStats().Connections*float64(aide1.GetExecutiveStats().Limits.Connections))
	want = "contacts aide1 2.03"
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	got = fmt.Sprint(c1.(*testContact).mostRecent)
	want = `[]` // no disconnect error.
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

}

func TestConnectionsOver(t *testing.T) {

	tokens.LoadPublicKeys()

	got := ""
	want := ""
	localtime := starttime
	getTime := func() uint32 {
		return localtime
	}

	// the token above allows for 1 connection.
	// we'll make 3 and then forward time for 20 min and all the contacts will get dropped because
	// they have violated the terms of the token.

	// allContacts := make([]*testContact, 0)

	// mnake cluster with 1 guru and 2 aides.
	// don't call operate or it will lose an aide.
	ce := iot.MakeSimplestCluster(getTime, false, 2, "")
	globalClusterExec = ce
	aide1 := ce.Aides[0]
	aide2 := ce.Aides[1]

	ce.WaitForActions()

	c1 := getNewContactFromAide(aide1, sampleToken1)
	// allContacts = append(allContacts, c1.(*testContact))
	SendText(c1, "S "+c1.String()) // subscribe to my name

	c1.(*testContact).doNotReconnect = true

	ce.WaitForActions()

	// there's one in the aide and one in the guru
	// note that the factory (MakeTestContact) does a connect and all connects
	// subscribe to the jwtid
	// so there's 4 and not 2.
	got = fmt.Sprint("topics collected ", ce.GetSubsCount())
	want = "topics collected 7"
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	c2 := getNewContactFromAide(aide2, sampleToken1)
	// allContacts = append(allContacts, c2.(*testContact))
	SendText(c2, "S "+c1.String()) // subscribe to my name

	c3 := getNewContactFromAide(aide2, sampleToken1)
	// allContacts = append(allContacts, c3.(*testContact))
	SendText(c3, "S "+c3.String()) // subscribe to my name

	c2.(*testContact).doNotReconnect = true
	c3.(*testContact).doNotReconnect = true

	// we should be getting something if the token is too small for 2 contacts. is a 2 contact 2 sub token
	// the contacts should be c3 refused
	ce.WaitForActions()

	fmt.Println("contacts aide1", aide1.GetExecutiveStats().Connections*float64(aide1.GetExecutiveStats().Limits.Connections))
	fmt.Println("contacts aide2", aide2.GetExecutiveStats().Connections*float64(aide2.GetExecutiveStats().Limits.Connections))

	for minutes := 0; minutes < 40; minutes++ {
		localtime += 60
		ce.Heartbeat(localtime)
		ce.WaitForActions()
		if minutes%5 == 4 {
			SendText(c1, "S "+c1.String()) // subscribe to my name, again
			SendText(c2, "S "+c2.String()) // keep them from timing out
			SendText(c3, "S "+c3.String())
		}
	}
	ce.WaitForActions()

	got = fmt.Sprint("contacts aide1 ", aide1.GetExecutiveStats().Connections*float64(aide1.GetExecutiveStats().Limits.Connections))
	got += fmt.Sprint(" contacts aide2 ", aide2.GetExecutiveStats().Connections*float64(aide2.GetExecutiveStats().Limits.Connections))
	want = "contacts aide1 3 contacts aide2 0"
	if got != want {
		// unreliable because of
		t.Errorf("got %v, want %v", got, want)
	}
	// note the packet in the q of c3 describes the error.
	got = fmt.Sprint(c3.(*testContact).popResultAsString())
	//fmt.Println(got)
	want = `[P,=jZae727K08KaOmKSgOaGzww_XVqGr_PK,ping," BILLING ERROR 1.66 connections > 1",error," BILLING ERROR 1.66 connections > 1"]true`
	//fmt.Println(want)
	// for i := 40; i < 45; i++ {
	// 	fmt.Println(got[i])
	// 	fmt.Println(want[i])
	// }
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

}

// TestBills is for the accumulator
func TestBills(t *testing.T) {

	got := ""
	want := ""

	ba := &iot.BillingAccumulator{}

	testtime := starttime

	// 9k seconds
	for i := 0; i < 100; i++ {
		stats := &tokens.KnotFreeContactStats{}
		stats.Input = 900
		dt := 90 // seconds
		stats.Connections = float64(dt)

		ba.AddUsage(stats, testtime, 90)

		testtime += uint32(dt)

		//fmt.Println("input rate", ba.GetInput(testtime))
		//	fmt.Println("conn rate", ba.GetConnections(testtime))

	}
	got = fmt.Sprint(ba.GetInput(testtime), ba.GetConnections(testtime))
	want = "9.39 0.93"
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

}

func TestBillingAccumulatorContact(t *testing.T) {

	got := ""
	want := ""

	localtime := starttime
	getTime := func() uint32 {
		return localtime
	}
	_ = getTime

	// let's loop by seconds. 20 min.
	// two connections, offset in time.

	c1nextHeart := starttime + 40 + 10 // the heartbeat for a whole executive
	c2nextHeart := starttime + 20 + 10

	c1lastBillingTime := c1nextHeart
	c1nextBillingTime := c1nextHeart + 30 // the heartbeat for a client, starts at 30 sec and then goes to 60

	c2lastBillingTime := c2nextHeart
	c2nextBillingTime := c2nextHeart + 30

	ba := &iot.BillingAccumulator{}

	for t := uint32(0); t < 60*20; t++ {

		localtime = starttime + t
		now := localtime

		changed := false

		// clients heartbeat every 10 sec
		if now > c1nextHeart {

			if localtime > c1nextBillingTime {

				deltaTime := float64(c1nextBillingTime - c1lastBillingTime) // see contacts ContactStruct.Heartbeat
				c1lastBillingTime = c1nextBillingTime
				c1nextBillingTime += 60 // 60 secs after first time

				stats := &tokens.KnotFreeContactStats{}
				stats.Connections = deltaTime // one connection
				stats.Input = 32 * deltaTime  // so 32 per sec
				stats.Output = 16 * deltaTime // so 16

				ba.AddUsage(stats, now, int(deltaTime))
				changed = true
			}

			c1nextHeart += 10
		}
		if localtime > c2nextHeart {

			if localtime > c2nextBillingTime {

				deltaTime := float64(c2nextBillingTime - c2lastBillingTime) // see contacts ContactStruct.Heartbeat
				c2lastBillingTime = c2nextBillingTime
				c2nextBillingTime += 60 // 60 secs after first time

				stats := &tokens.KnotFreeContactStats{}
				stats.Connections = deltaTime // one connection
				stats.Input = 24 * deltaTime  // so 32 per sec
				stats.Output = 8 * deltaTime  // so 16

				ba.AddUsage(stats, now, int(deltaTime))
				changed = true
			}
			c2nextHeart += 10
		}
		if changed {
			statsResult := &tokens.KnotFreeContactStats{}
			ba.GetStats(now, statsResult)
			fmt.Println("stats", statsResult)
		}
	}

	got = fmt.Sprint(ba.GetConnections(localtime))
	want = "1.9"
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	got = fmt.Sprint(ba.GetInput(localtime))
	want = "53.21" // of 56
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	got = fmt.Sprint(ba.GetOutput(localtime))
	want = "22.8" // of 24
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

}
