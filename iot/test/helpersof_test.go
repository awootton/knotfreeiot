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
	"bufio"
	"errors"
	"fmt"
	"net"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/awootton/knotfreeiot/iot"
	"github.com/awootton/knotfreeiot/packets"
	"github.com/awootton/knotfreeiot/tokens"
	"github.com/prometheus/client_golang/prometheus"

	dto "github.com/prometheus/client_model/go"
)

var globalClusterExec *iot.ClusterExecutive

const starttime = uint32(1577840400) // Wednesday, January 1, 2020 1:00:00 AM

// a typical bottom contact with a q instead of a writer
type testContact struct {
	iot.ContactStruct

	//downMessages chan packets.Interface

	mostRecent chan (packets.Interface)

	doNotReconnect bool

	index int

	// map from hashedaddress to string address?
}

func makeTestContact(config *iot.ContactStructConfig, token string) iot.ContactInterface {

	acontact := testContact{}
	acontact.mostRecent = make(chan (packets.Interface), 1000)

	acontact.SetReader(&iot.DevNull{})
	acontact.SetWriter(&iot.DevNull{})

	if len(token) == 0 {
		token = string(tokens.Get32xTokenLocal()) //GetImpromptuGiantToken()
	}

	// go func(cc *testContact) {
	// 	for {
	// 		thing := <-cc.downMessages

	// 		str := thing.String()
	// 		if strings.HasPrefix(str, "[P,contactTopic45") {
	// 			//fmt.Println("into mostRecent", thing, reflect.TypeOf(thing))
	// 		}
	// 		fmt.Println("into mostRecent", thing, reflect.TypeOf(thing), cc.String())

	// 		if reflect.TypeOf(thing) == reflect.TypeOf(&packets.Disconnect{}) {
	// 			if cc.doNotReconnect == true {
	// 				e, _ := thing.GetOption("error")
	// 				cc.Close(errors.New(string(e)))
	// 				return // we're done forever.
	// 			}
	// 			// we pretend to reconnect by not closing
	// 		} else {

	// 			send, isSend := thing.(*packets.Send)
	// 			if isSend {
	// 				send.Address Alias = []byte("")
	// 			}
	// 			//fmt.Println("appending mostRecent", thing)
	// 			cc.mostRecent = append(cc.mostRecent, thing)
	// 		}
	// 	}
	// }(&acontact)

	iot.AddContactStruct(&acontact.ContactStruct, &acontact, config)

	if len(token) == 0 {
		token = tokens.GetImpromptuGiantToken()
	}
	connect := packets.Connect{}
	connect.SetOption("token", []byte(token))
	iot.PushPacketUpFromBottom(&acontact, &connect)

	return &acontact
}

func (cc *testContact) Close(err error) {
	dis := packets.Disconnect{}
	dis.SetOption("error", []byte(err.Error()))
	cc.WriteDownstream(&dis)
	ss := &cc.ContactStruct
	ss.DoClose(err)
}

func (cc *testContact) getResultsCount() int { // who does this?
	return len(cc.mostRecent)
}

// func (cc *testContact) XXgetResultAsString() (string, bool) {
// 	if len(cc.mostRecent) == 0 {
// 		return "no message received", false
// 	}
// 	return cc.mostRecent[0].String(), true
// }

func (cc *testContact) popResultAsString() (string, bool) {

	select {
	case thing := <-cc.mostRecent:
		return thing.String(), true
	case t := <-time.After(time.Millisecond * 1000):
		_ = t
		return "no message received ", false
	}
	// val, ok := cc.getResultAsString()
	// if ok {
	// 	cc.mostRecent = cc.mostRecent[1:]
	// }
	// return val, ok
}

// write from the bottom of a node going down through the contact.
// and since this is test it ends up in an array: mostRecent
func (cc *testContact) WriteDownstream(packet packets.Interface) error {

	if cc.IsClosed() {
		return nil
	}
	send, isSend := packet.(*packets.Send)
	if isSend {
		_ = send //??
	}
	text := packet.String()
	cc.IncOutput(len(text))

	fmt.Println(cc.index, "APPENDING to mostRecent", text)

	// cc.mostRecent = append(cc.mostRecent, packet) //use stream instead of array
	cc.mostRecent <- packet

	if cc.IsClosed() == false && cc.GetConfig().IsGuru() == false { // fixme: ignore them only if jwtid doesn't match.
		u := iot.HasError(packet)
		if u != nil {
			// cc.mostRecent = append(cc.mostRecent, u)
			cc.mostRecent <- u
		}
	}
	return nil
}

// TODO: i think we can get rid of WriteUpstream as a method of ContactInterface
func (cc *testContact) WriteUpstream(cmd packets.Interface) error {
	fmt.Println("FIXME received from below does this exist?", cmd, reflect.TypeOf(cmd))
	return errors.New("Only upper contacts get WriteUpstream")
}

func ReadCounter(m prometheus.Counter) float64 {
	pb := &dto.Metric{}
	m.Write(pb)
	return pb.GetCounter().GetValue()
}

// SendText chops up the text and creates a packets.Interface packet.
func SendText(cc iot.ContactInterface, text string) {

	// forbidden: contact1.ContactStruct.input += float32(len(text))
	// this will do the same thing:
	_, _ = cc.Read([]byte(text))

	p, _ := iot.Text2Packet(text)
	iot.PushPacketUpFromBottom(cc, p)

}

// socket util
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

// socket util
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
			fmt.Println("read err here", err) // FIXME: return err?
		}
		return "" // normal for timeout
	}
	if len(str) > 0 {
		str = str[0 : len(str)-1]
	}
	conn.SetDeadline(time.Now().Add(600 * time.Second))
	return str
}

func openConnectedSocket(name string, t *testing.T, token string) *net.TCPConn {

	conn1, err := net.DialTimeout("tcp", name, time.Duration(10*time.Millisecond)) //net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		println("Dial 1 failed:", err.Error())
		t.Fail()
	}

	if len(token) == 0 {
		token = string(tokens.Get32xTokenLocal())
	}

	connect := packets.Connect{}
	connect.SetOption("token", []byte(token))
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
		println("Dial 2 failed:", err.Error())
		t.Fail()
	}
	return conn1.(*net.TCPConn)
}

// WaitForActions is a utility for test that attempts to wait for all async activity to finish.
// in test..depricated. call ce.WaitForActions() instead
func WaitForActions(ex *iot.Executive) {
	ex.WaitForActions()
	for i := 0; i < 10; i++ { // this is going to be a problem?
		time.Sleep(time.Millisecond)
		runtime.Gosched() // single this
	}
}

// GetNewContactFromAide contacts the aide
// fixme unwind and delete if possible. why public?
// only for tests
func getNewContactFromAide(aide *iot.Executive, token string) iot.ContactInterface {

	if aide == nil {
		return nil // fixme return error
	}
	cc := makeTestContact(aide.Config, token)
	return cc
}

// GetNewContactFromSlackestAide add a contact to the least used of the aides
// only for tests
func getNewContactFromSlackestAide(ce *iot.ClusterExecutive, token string) iot.ContactInterface {
	min := 1 << 30
	var smallestAide *iot.Executive
	for _, aide := range ce.Aides {
		cons, fract := aide.Looker.GetAllSubsCount()
		if cons < min {
			min = cons
			smallestAide = aide
		}
		_ = fract
	}
	if smallestAide == nil {
		return nil // fixme return error
	}
	//fmt.Println("smallest aide is ", smallestAide.Name)
	cc := makeTestContact(smallestAide.Config, token)
	return cc
}
