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
	"math/rand"
	"testing"

	"github.com/awootton/knotfreeiot/iot"
	"github.com/awootton/knotfreeiot/packets"
	"github.com/awootton/knotfreeiot/tokens"
)

func TestReserveSubscription(t *testing.T) {

	localtime := starttime
	getTime := func() uint32 {
		return localtime
	}

	rand.Seed(123456)

	clusters := StartClusterOfClusters(getTime)
	_ = clusters

	aide0 := getAnAide(clusters, 0)
	aide9 := getAnAide(clusters, 9999) // a completely different cluster, will have to go through super.

	contact0 := makeTestContact(aide0.Config, "")
	contact9 := makeTestContact(aide9.Config, "")

	connect := packets.Connect{}
	connect.SetOption("token", []byte(tokens.SampleSmallToken))
	iot.PushPacketUpFromBottom(contact0, &connect)
	iot.PushPacketUpFromBottom(contact9, &connect)

	aMap := iot.GuruNameToConfigMap
	topGuruExec := aMap["guru0_0_1"]
	topGuruExecSubs, _ := topGuruExec.GetSubsCount()

	subs := packets.Subscribe{}
	subs.Address.FromString("contact9_address")
	err := iot.PushPacketUpFromBottom(contact9, &subs)
	if err != nil {
		t.Error("got error ")
	}

	IterateAndWait(t, func() bool {
		cnt, _ := topGuruExec.GetSubsCount()
		return cnt > topGuruExecSubs
	}, "timed out waiting for sub to move up")

	sendmessage := packets.Send{}
	sendmessage.Address.FromString("contact9_address")
	sendmessage.Source.FromString("contact0_address")
	sendmessage.Payload = []byte("can you hear me now?")
	err = iot.PushPacketUpFromBottom(contact0, &sendmessage)
	if err != nil {
		t.Error("got error ")
	}
	got := ""
	IterateAndWait(t, func() bool {
		got = contact9.(*testContact).getResultAsString()
		return got != "no message received"
	}, "timed out waiting for can you hear me now")

	fmt.Println("reply was " + got)
	want := `[P,=6X2eixvv3rz9Irvi85t2S5gdA0tRfB0B,contact0_address,"can you hear me now?"]`
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

}
