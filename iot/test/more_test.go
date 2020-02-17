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
	"strconv"
	"testing"

	"github.com/awootton/knotfreeiot/iot"
)

func TestExec(t *testing.T) {

	got := ""
	want := ""

	ce := iot.MakeSimplestCluster(getTime, testNameResolver)

	c1 := ce.GetNewContact(MakeTestContact)
	SendText(c1, "S chan1")

	ct := c1.(*testContact)
	got = ct.getResultAsString()

	// there one in the aide and one in the guru
	got = fmt.Sprint("topics collected ", ce.GetSubsCount())
	want = "topics collected 2"
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	// add a contact a minute and see what happens.
	for i := 0; i < 100; i++ {
		ci := ce.GetNewContact(MakeTestContact)
		SendText(ci, "S chan"+strconv.FormatInt(int64(i), 10))
		currentTime += 60 // a minute
		ce.Operate()
	}

	got = ct.getResultAsString()
	// there one in the aide and one in the guru
	got = fmt.Sprint("topics collected ", ce.GetSubsCount())
	want = "topics collected 200"
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	fmt.Println("total minions", len(ce.Aides))

}
