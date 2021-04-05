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

func TestTwoTierTcp(t *testing.T) {

	tokens.LoadPublicKeys()
	got := ""
	want := ""
	ok := true
	var err error
	localtime := starttime
	getTime := func() uint32 {
		return localtime
	}
	ce := iot.MakeSimplestCluster(getTime, true, 2, "")
	globalClusterExec = ce

	ce.WaitForActions()

	sss, _ := iot.GetServerStats(ce.Aides[0].GetHTTPAddress())
	fmt.Println("aide stats", sss)
	sss, _ = iot.GetServerStats(ce.Gurus[0].GetHTTPAddress())
	fmt.Println("guru stats", sss)

	n := sss.Subscriptions * float32(ce.Gurus[0].Limits.Subscriptions)
	if n != 1.0 {
		// t.Errorf("got %v, want %v", n, 1.0)
	}

	sock1 := openPlainSocket(ce.Aides[0].GetTextAddress(), t)
	sock2 := openPlainSocket(ce.Aides[1].GetTextAddress(), t)

	sock1.SetNoDelay(true)
	sock2.SetNoDelay(true)

	ce.WaitForActions()

	connectStr := "C token " + `'` + tokens.SampleSmallToken + `'` + "\n"
	sock1.Write([]byte(connectStr))
	sock2.Write([]byte(connectStr))

	// subscribe
	sock1.Write([]byte("S sock1channel  \n"))
	sock2.Write([]byte("S sock2channel  \n"))

	ce.WaitForActions() // blech sock2 has to finish before sock1 sends

	sock1.Write([]byte("P sock2channel :retadd: some_test_hello1\n"))

	ce.WaitForActions()

	got = readLine(sock2)
	want = `[P,=1CHKeHF6q1WLMSylXwB0gRs+VVKJvEiH,retadd,some_test_hello1]`
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	aideStats, _ := iot.GetServerStats(ce.Aides[0].GetHTTPAddress())
	fmt.Println("aide stats2", aideStats)
	guruStats, _ := iot.GetServerStats(ce.Gurus[0].GetHTTPAddress())
	fmt.Println("guru stats2", guruStats)

	// the guru gained a subscription because the aide connected to it.
	n = guruStats.Subscriptions * float32(ce.Gurus[0].Limits.Subscriptions)
	if n != 4.0 {
		//  t.Errorf("got %v, want %v", n, 4.0)
	}

	//TODO: two aides test

	_ = got
	_ = want
	//
	_ = ok
	_ = err

	sock1.Close()
	sock2.Close()
	ce.WaitForActions()

}

func TestSimpleText(t *testing.T) {

	tokens.LoadPublicKeys()
	got := ""
	want := ""
	ok := true
	var err error
	localtime := starttime
	getTime := func() uint32 {
		return localtime
	}
	var ce *iot.ClusterExecutive

	// set up
	guru := iot.NewExecutive(100, "guru", getTime, true, ce)

	iot.MakeTCPExecutive(guru, "localhost:8088")
	iot.MakeTextExecutive(guru, "localhost:7465")

	time.Sleep(10 * time.Millisecond)
	// open a socket
	sock1 := openPlainSocket("localhost:7465", t)
	sock2 := openPlainSocket("localhost:7465", t)

	sock1.SetNoDelay(true)
	sock2.SetNoDelay(true)

	connectStr := "C token " + `'` + tokens.SampleSmallToken + `'` + "\n"
	sock1.Write([]byte(connectStr))
	sock2.Write([]byte(connectStr))

	// subscribe
	sock1.Write([]byte("S sock1channel  \n"))
	sock2.Write([]byte("S sock2channel  \n"))

	WaitForActions(guru)

	sock1.Write([]byte("P sock2channel :ra: some_test_hello2\n"))

	WaitForActions(guru)

	got = readLine(sock2)
	want = `[P,=1CHKeHF6q1WLMSylXwB0gRs+VVKJvEiH,ra,some_test_hello2]`
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	_ = got
	_ = want
	_ = guru
	_ = ok
	_ = err
	_ = sock2

}

// start one server, send it some packets.

func TestSimpleExecutive(t *testing.T) {

	tokens.LoadPublicKeys()

	got := ""
	want := ""
	ok := true
	var err error
	localtime := starttime
	getTime := func() uint32 {
		return localtime
	}
	var ce *iot.ClusterExecutive

	// set up
	guru := iot.NewExecutive(100, "guru", getTime, true, ce)

	iot.MakeTCPExecutive(guru, "localhost:8089")

	time.Sleep(10 * time.Millisecond)
	// open a socket
	sock1 := openConnectedSocket("localhost:8089", t, "")
	sock2 := openConnectedSocket("localhost:8089", t, "")

	sock1.SetNoDelay(true)
	sock2.SetNoDelay(true)

	// subscribe
	p, _ := iot.Text2Packet("S sock1channel")
	p.Write(sock1)
	p, _ = iot.Text2Packet("S sock2channel")
	p.Write(sock2)

	WaitForActions(guru)

	p, _ = iot.Text2Packet("P sock2channel:bbcc:some_test_hello3")
	p.Write(sock1)

	WaitForActions(guru)

	p = readSocket(sock2)
	got = fmt.Sprint(p)
	want = `[P,=1CHKeHF6q1WLMSylXwB0gRs+VVKJvEiH,bbcc,some_test_hello3]`
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	_ = got
	_ = want
	_ = guru
	_ = ok
	_ = err
	_ = sock2

}
