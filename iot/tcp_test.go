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
	"bufio"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/awootton/knotfreeiot/iot"
	"github.com/awootton/knotfreeiot/packets"
	"github.com/awootton/knotfreeiot/tokens"
)

func TestSimpleText(t *testing.T) {

	tokens.SavePublicKey("1iVt", string(tokens.GetSamplePublic()))

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

func TestSimpleEx(t *testing.T) {

	tokens.SavePublicKey("1iVt", string(tokens.GetSamplePublic()))

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

func readSocket(conn *net.TCPConn) packets.Interface {

	err := conn.SetDeadline(time.Now().Add(10 * time.Millisecond))
	if err != nil {
		// /srvrLogThing.Collect("cl err4 " + err.Error())
		return &packets.Disconnect{}
	}
	p, err := packets.ReadPacket(conn)
	if err != nil {
		str := err.Error() // "read tcp 127.0.0.1:50053->127.0.0.1:1234: i/o timeout"
		if !(strings.HasPrefix(str, "read tcp ") && strings.HasSuffix(str, ": i/o timeout")) {
			fmt.Println("read err her", err)
		}
		return &packets.Disconnect{} // normal for timeout
	}
	conn.SetDeadline(time.Now().Add(600 * time.Second))
	return p
}

func readLine(conn *net.TCPConn) string {

	err := conn.SetDeadline(time.Now().Add(10 * time.Millisecond))
	if err != nil {
		// /srvrLogThing.Collect("cl err4 " + err.Error())
		fmt.Println("read line fail1", err)
		return ""
	}
	lineReader := bufio.NewReader(conn)
	str, err := lineReader.ReadString('\n')
	if err != nil {
		str := err.Error() // "read tcp 127.0.0.1:50053->127.0.0.1:1234: i/o timeout"
		if !(strings.HasPrefix(str, "read tcp ") && strings.HasSuffix(str, ": i/o timeout")) {
			fmt.Println("read err her", err)
		}
		return "" // normal for timeout
	}
	if len(str) > 0 {
		str = str[0 : len(str)-1]
	}
	conn.SetDeadline(time.Now().Add(600 * time.Second))
	return str
}

func openConnectedSocket(name string, t *testing.T) *net.TCPConn {

	// tcpAddr, err := net.ResolveTCPAddr("tcp", name)
	// if err != nil {
	// 	println("ResolveTCPAddr failed:", err.Error())
	// 	t.Fail()
	// }

	conn1, err := net.DialTimeout("tcp", name, time.Duration(10*time.Millisecond)) //net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		println("Dial failed:", err.Error())
		t.Fail()
	}
	connect := packets.Connect{}
	connect.SetOption("token", []byte(tokens.SampleSmallToken))
	err = connect.Write(conn1)
	if err != nil {
		println("connect failed:", err.Error())
		t.Fail()
	}
	return conn1.(*net.TCPConn)
}

func openPlainSocket(name string, t *testing.T) *net.TCPConn {

	conn1, err := net.DialTimeout("tcp", name, time.Duration(10*time.Millisecond)) //net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		println("Dial failed:", err.Error())
		t.Fail()
	}
	return conn1.(*net.TCPConn)
}
