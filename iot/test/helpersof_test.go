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

const starttime = uint32(1577840400) // Wednesday, January 1, 2020 1:00:00 AM
//var currentTime = uint32(1577840400)

// a typical bottom contact with a q instead of a writer
type testContact struct {
	iot.ContactStruct

	downMessages chan packets.Interface

	mostRecent []packets.Interface

	doNotReconnect bool

	index int
}

type testUpperContact struct {
	iot.ContactStruct

	guruBottomContact *testContact
}

func MakeTestContact(config *iot.ContactStructConfig, token string) iot.ContactInterface {

	acontact := testContact{}
	acontact.downMessages = make(chan packets.Interface, 1000)
	acontact.mostRecent = make([]packets.Interface, 0, 1000)

	if len(token) == 0 {
		token = tokens.SampleSmallToken
	}

	go func(cc *testContact) {
		for {
			thing := <-cc.downMessages

			str := thing.String()
			if strings.HasPrefix(str, "[P,contactTopic45") {
				//fmt.Println("into mostRecent", thing, reflect.TypeOf(thing))
			}
			//fmt.Println("into mostRecent", thing, reflect.TypeOf(thing))

			if reflect.TypeOf(thing) == reflect.TypeOf(&packets.Disconnect{}) {
				if cc.doNotReconnect == true {
					e, _ := thing.GetOption("error")
					cc.Close(errors.New(string(e)))
					return // we're done forever.
				}
				// now we have to reconnect
				//fmt.Println("contact reattaching", cc.index)
				globalClusterExec.AttachContact(cc, AttachTestContact, token)
				// we should also reiterate our connect and our subscription.
				// FIXME: really, we should.
			} else {
				//fmt.Println("appending mostRecent", thing)
				cc.mostRecent = append(cc.mostRecent, thing)
			}
		}
	}(&acontact)
	iot.AddContactStruct(&acontact.ContactStruct, &acontact, config)

	if len(token) == 0 {
		token = tokens.SampleSmallToken
	}
	connect := packets.Connect{}
	connect.SetOption("token", []byte(token))
	iot.Push(&acontact, &connect)

	return &acontact
}

// the contact is already made but got closed or something and needs to
// re-attach
func AttachTestContact(cc iot.ContactInterface, config *iot.ContactStructConfig, token string) {
	contact1 := cc.(*testContact)
	//contact1.downMessages = make(chan packets.Interface, 1000)
	iot.AddContactStruct(&contact1.ContactStruct, contact1, config)

	if len(token) == 0 {
		token = tokens.SampleSmallToken
	}

	connect := packets.Connect{}
	connect.SetOption("token", []byte(token))
	iot.Push(contact1, &connect)
}

// called by Lookup PushUp
func (cc *testUpperContact) WriteUpstream(cmd packets.Interface) error {
	// call the Push
	err := iot.Push(cc.guruBottomContact, cmd)
	return err
}

func testNameResolver(name string, config *iot.ContactStructConfig) (iot.ContactInterface, error) {
	exec, ok := iot.GuruNameToConfigMap[name]

	//fmt.Println(config.Name, " resolving conn to ", name, " found ", exec)

	if ok && exec != nil { // todo: better names.
		// IRL this is a tcp connect to the guru
		// this is the contect that aide1 and aide2 will be using at their top
		contactTop1 := testUpperContact{}
		iot.InitUpperContactStruct(&contactTop1.ContactStruct, config)
		// This is the one attaching to the bottom of guru0
		// this work would be done after the socket accept by guru0
		newLowerContact := testContact{}
		newLowerContact.downMessages = make(chan packets.Interface, 1000)
		iot.AddContactStruct(&newLowerContact.ContactStruct, &newLowerContact, exec.Config)

		connect := packets.Connect{}
		connect.SetOption("token", []byte(tokens.SampleSmallToken))
		err := iot.Push(&newLowerContact, &connect)
		if err != nil {
			return &contactTop1, err
		}
		// wire them up
		contactTop1.guruBottomContact = &newLowerContact
		go func(contact *testUpperContact, lower *testContact) {
			for { // add timeout?
				packet := <-lower.downMessages
				_, ok := packet.(*packets.Disconnect)
				if ok {
					// except when test is calling close.
					//fmt.Println("guru no tsupposed to ever send disconnect", lower.GetConfig().Name)
				}
				//fmt.Println("top push down", packet)
				iot.PushDown(contact, packet)
			}
		}(&contactTop1, &newLowerContact)
		return &contactTop1, nil

	} else {
		return &testUpperContact{}, errors.New("unknown name " + name)
	}
}

// func (cc *testContact) get() (packets.Interface, bool) {
// 	select {
// 	case msg := <-cc.downMessages:
// 		return msg, true
// 	case <-time.After(10 * time.Millisecond):
// 		return nil, false
// 	}
// }

func (cc *testContact) Close(err error) {
	dis := packets.Disconnect{}
	dis.SetOption("error", []byte(err.Error()))
	cc.WriteDownstream(&dis)

	ss := &cc.ContactStruct
	ss.Close(err)
}

func (cc *testContact) getResultAsString() string {
	// gotmsg, ok := cc.get()
	// got := ""
	// if ok {
	// 	got = gotmsg.String()
	// } else {
	// 	got = fmt.Sprint("no message received", gotmsg)
	// }
	if len(cc.mostRecent) == 0 {
		return "no message received"
	}
	return cc.mostRecent[0].String()
}

func (cc *testContact) WriteDownstream(packet packets.Interface) error {

	if cc.GetClosed() {
		return nil
	}
	_, ok := packet.(*packets.Disconnect)
	if ok {
		// Close(..) in this file will write Disconnect
		// unless his token is bad, I guess.
		//fmt.Println("guru not supposed to ever send disconnect", cc.GetConfig().Name)
	}
	text := packet.String()
	cc.ContactStruct.GetBilling().Input += float32(len(text))

	_, ok = packet.(*packets.Send)
	if ok {
		//fmt.Println("have send", cc.GetConfig().Name)
	}

	cc.downMessages <- packet

	if !cc.GetConfig().IsGuru() { // fixme: ignore them only if jwtid doesn't match.
		u := iot.HasError(packet)
		if u != nil {
			cc.downMessages <- u
		}
	}
	return nil
}

func (cc *testContact) WriteUpstream(cmd packets.Interface) error {
	fmt.Println("FIXME received from below does this exist?", cmd, reflect.TypeOf(cmd))
	return errors.New("Only upper contacts get WriteUpstream")
}

func readCounter(m prometheus.Counter) float64 {
	pb := &dto.Metric{}
	m.Write(pb)
	return pb.GetCounter().GetValue()
}

// SendText chops up the text and creates a packets.Interface packet.
func SendText(cc iot.ContactInterface, text string) {

	contact1 := cc.(*testContact)
	contact1.ContactStruct.GetBilling().Input += float32(len(text))

	p, _ := iot.Text2Packet(text)
	iot.Push(cc, p)

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

func openConnectedSocket(name string, t *testing.T, token string) *net.TCPConn {

	// tcpAddr, err := net.ResolveTCPAddr("tcp", name)
	// if err != nil {
	// 	println("ResolveTCPAddr failed:", err.Error())
	// 	t.Fail()
	// }

	conn1, err := net.DialTimeout("tcp", name, time.Duration(10*time.Millisecond)) //net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		println("Dial 1 failed:", err.Error())
		t.Fail()
	}

	if len(token) == 0 {
		token = tokens.SampleSmallToken
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
// in test
func WaitForActions(ex *iot.Executive) {
	ex.WaitForActions()
	for i := 0; i < 10; i++ { // this is going to be a problem?
		time.Sleep(time.Millisecond)
		runtime.Gosched() // single this
	}
}
