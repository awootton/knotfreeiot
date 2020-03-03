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
	currentTime = starttime

	ce := iot.MakeSimplestCluster(getTime, iot.TCPNameResolver, true, 2)
	globalClusterExec = ce

	WaitForActions()

	sss := iot.GetServerStats(ce.Aides[0].GetHTTPAddress())
	fmt.Println("aide stats", sss)
	sss = iot.GetServerStats(ce.Gurus[0].GetHTTPAddress())
	fmt.Println("guru stats", sss)

	n := sss.Subscriptions * float64(ce.Gurus[0].Limits.Subscriptions)
	if n != 1.0 {
		t.Errorf("got %v, want %v", n, 1.0)
	}

	sock1 := openPlainSocket(ce.Aides[0].GetTextAddress(), t)
	sock2 := openPlainSocket(ce.Aides[1].GetTextAddress(), t)

	sock1.SetNoDelay(true)
	sock2.SetNoDelay(true)

	WaitForActions()

	connectStr := "C token " + `"` + tokens.SampleSmallToken + `"` + "\n"
	sock1.Write([]byte(connectStr))
	sock2.Write([]byte(connectStr))

	// subscribe
	sock1.Write([]byte("S sock1channel  \n"))
	sock2.Write([]byte("S sock2channel  \n"))

	WaitForActions() // blech sock2 has to finish before sock1 sends

	sock1.Write([]byte("P sock2channel :::: some_test_hello\n"))

	WaitForActions()

	got = readLine(sock2)
	want = `[P,sock2channel,=1CHKeHF6q1WLMSylXwB0gRs+VVKJvEiHD7dB2+H78OQ,,,some_test_hello]`
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	aideStats := iot.GetServerStats(ce.Aides[0].GetHTTPAddress())
	fmt.Println("aide stats2", aideStats)
	guruStats := iot.GetServerStats(ce.Gurus[0].GetHTTPAddress())
	fmt.Println("guru stats2", guruStats)

	// the guru gained a subscription because the aide connected to it.
	n = guruStats.Subscriptions * float64(ce.Gurus[0].Limits.Subscriptions)
	if n != 3.0 {
		t.Errorf("got %v, want %v", n, 3.0)
	}

	//TODO: two aides test

	_ = got
	_ = want
	//
	_ = ok
	_ = err
	//_ = sock2

	sock1.Close()
	sock2.Close()
	WaitForActions()

}

func TestSimpleText(t *testing.T) {

	tokens.LoadPublicKeys()
	got := ""
	want := ""
	ok := true
	var err error
	currentTime = starttime

	// set up
	guru := iot.NewExecutive(100, "guru", getTime, true)

	iot.MakeTCPExecutive(guru, "localhost:8088")
	iot.MakeTextExecutive(guru, "localhost:7465")

	time.Sleep(10 * time.Millisecond)
	// open a socket
	sock1 := openPlainSocket("localhost:7465", t)
	sock2 := openPlainSocket("localhost:7465", t)

	sock1.SetNoDelay(true)
	sock2.SetNoDelay(true)

	connectStr := "C token " + `"` + tokens.SampleSmallToken + `"` + "\n"
	sock1.Write([]byte(connectStr))
	sock2.Write([]byte(connectStr))

	// subscribe
	sock1.Write([]byte("S sock1channel  \n"))
	sock2.Write([]byte("S sock2channel  \n"))
	time.Sleep(10 * time.Millisecond) // blech sock2 has to finish before sock1 sends

	sock1.Write([]byte("P sock2channel :::: some_test_hello\n"))

	WaitForActions()

	got = readLine(sock2)
	want = `[P,sock2channel,=1CHKeHF6q1WLMSylXwB0gRs+VVKJvEiHD7dB2+H78OQ,,,some_test_hello]`
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	//guru.IAmBadError = errors.New("naptime")
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
	currentTime = starttime

	// set up
	guru := iot.NewExecutive(100, "guru", getTime, true)

	iot.MakeTCPExecutive(guru, "localhost:8089")

	time.Sleep(10 * time.Millisecond)
	// open a socket
	sock1 := openConnectedSocket("localhost:8089", t)
	sock2 := openConnectedSocket("localhost:8089", t)

	sock1.SetNoDelay(true)
	sock2.SetNoDelay(true)

	// subscribe
	p, _ := iot.Text2Packet("S sock1channel")
	p.Write(sock1)
	p, _ = iot.Text2Packet("S sock2channel")
	p.Write(sock2)

	time.Sleep(10 * time.Millisecond) // blech

	// p = readSocket(sock1)
	// got = fmt.Sprint(p)
	// want = "[D]" // a D packet is normal when there's nothing to receive.
	// if got != want {
	// 	t.Errorf("got %v, want %v", got, want)
	// }

	p, _ = iot.Text2Packet("P sock2channel::::some_test_hello")
	p.Write(sock1)

	time.Sleep(10 * time.Millisecond)

	p = readSocket(sock2)
	got = fmt.Sprint(p)
	want = `[P,sock2channel,=1CHKeHF6q1WLMSylXwB0gRs+VVKJvEiHD7dB2+H78OQ,,,some_test_hello]`
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
