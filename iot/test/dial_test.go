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
	"time"

	"github.com/awootton/knotfreeiot/iot"
	"github.com/awootton/knotfreeiot/tokens"
)

func Xnot_TestDialTCP(t *testing.T) {

	aTestDial1(t, true)

}

func TestDialPlain(t *testing.T) {

	aTestDial1(t, false)

}

func aTestDial1(t *testing.T, isTCP bool) {

	tokens.LoadPublicKeys()
	got := ""
	want := ""
	ok := true
	var err error
	localtime := starttime
	getTime := func() uint32 {
		return localtime
	}

	ce := iot.MakeSimplestCluster(getTime, isTCP, 2, "")
	globalClusterExec = ce

	ce.WaitForActions()
	time.Sleep(20 * time.Millisecond)

	c1 := getNewContactFromAide(ce.Aides[0], "")
	SendText(c1, "S contactTopic1")

	c2 := getNewContactFromAide(ce.Aides[1], "")
	SendText(c2, "S contactTopic2")

	time.Sleep(20 * time.Millisecond)

	SendText(c1, "P contactTopic2,dummyreturn,hello_msg") // send/pub from c1 to c2

	IterateAndWait(t, func() bool {
		ce.WaitForActions()
		got = fmt.Sprint(c2.(*testContact).mostRecent)
		return got != "[]"
	}, "timed out waiting aTestDial1")

	// ce.WaitForActions() // FIXME: use iterate and wait
	// for i := 0; i < 20; i++ {
	// 	//	localtime += 60
	// 	//	ce.Heartbeat(localtime)
	// 	ce.WaitForActions()
	// }
	// time.Sleep(20 * time.Millisecond)
	// time.Sleep(20 * time.Millisecond)
	// time.Sleep(20 * time.Millisecond)
	// time.Sleep(20 * time.Millisecond)

	got = fmt.Sprint(c2.(*testContact).mostRecent)
	want = `[[P,=LiNMB4JFOy7dUJJ9vMVyzhMzLz6ozRnQ,dummyreturn,hello_msg]]`
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	got = fmt.Sprint(c1.(*testContact).mostRecent)
	want = `[]`
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	_ = got
	_ = want
	//
	_ = ok
	_ = err

}
