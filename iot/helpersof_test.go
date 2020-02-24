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
	"runtime"
	"strings"
	"time"

	"github.com/awootton/knotfreeiot/iot"
	"github.com/awootton/knotfreeiot/packets"
	"github.com/awootton/knotfreeiot/tokens"
	"github.com/prometheus/client_golang/prometheus"

	dto "github.com/prometheus/client_model/go"
)

const starttime = uint32(1577840400) // Wednesday, January 1, 2020 1:00:00 AM
var currentTime = uint32(1577840400)

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

func MakeTestContact(config *iot.ContactStructConfig) iot.ContactInterface {

	acontact := testContact{}
	acontact.downMessages = make(chan packets.Interface, 1000)
	acontact.mostRecent = make([]packets.Interface, 0, 1000)

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
					return // we're done forever.
				}
				// now we have to reconnect
				//fmt.Println("contact reattaching", cc.index)
				globalClusterExec.AttachContact(cc, AttachTestContact)
				// we should also reiterate our connect and our subscription.
				SendText(cc, "S "+cc.String()) // FIXME: this is garbage
			} else {
				cc.mostRecent = append(cc.mostRecent, thing)
			}
		}
	}(&acontact)
	iot.AddContactStruct(&acontact.ContactStruct, &acontact, config)

	connect := packets.Connect{}
	connect.SetOption("token", []byte(tokens.SampleSmallToken))
	iot.Push(&acontact, &connect)

	return &acontact
}

// the contact is already made but got closed or something and needs to
// re-attach
func AttachTestContact(cc iot.ContactInterface, config *iot.ContactStructConfig) {
	contact1 := cc.(*testContact)
	//contact1.downMessages = make(chan packets.Interface, 1000)
	iot.AddContactStruct(&contact1.ContactStruct, contact1, config)

	connect := packets.Connect{}
	connect.SetOption("token", []byte(tokens.SampleSmallToken))
	iot.Push(contact1, &connect)
}

// called by Lookup PushUp
func (cc *testUpperContact) WriteUpstream(cmd packets.Interface) {
	// call the Push
	iot.Push(cc.guruBottomContact, cmd)
}

func getTime() uint32 {
	return currentTime
}

func testNameResolver(name string, config *iot.ContactStructConfig) (iot.ContactInterface, error) {
	exec, ok := iot.GuruNameToConfigMap[name]
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
		iot.Push(&newLowerContact, &connect)

		// wire them up
		contactTop1.guruBottomContact = &newLowerContact
		go func() {
			for { // add timeout?
				packet := <-newLowerContact.downMessages
				iot.PushDown(&contactTop1, packet)
			}
		}()
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
	ss := cc.ContactStruct
	ss.Close(err)

	dis := packets.Disconnect{}
	dis.SetOption("error", []byte(err.Error()))
	cc.WriteDownstream(&dis)
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

func (cc *testContact) WriteDownstream(packet packets.Interface) {

	str := packet.String()
	if strings.HasPrefix(str, "[P,contactTopic45") {
		//fmt.Println("received from above", packet, reflect.TypeOf(packet))
	}
	cc.downMessages <- packet
}

func (cc *testContact) WriteUpstream(cmd packets.Interface) {
	fmt.Println("FIXME received from below does this exist?", cmd, reflect.TypeOf(cmd))
}

func readCounter(m prometheus.Counter) float64 {
	pb := &dto.Metric{}
	m.Write(pb)
	return pb.GetCounter().GetValue()
}

// SendText chops up the text and creates a packets.Interface packet.
func SendText(cc iot.ContactInterface, text string) {

	p, _ := iot.Text2Packet(text)
	iot.Push(cc, p)

}

// WaitForActions needs to be properly implemented.
// The correct thing to do is to inject tracer packets with wait groups into q's
// and then wait for that.
func WaitForActions() {
	for i := 0; i < 10; i++ { // this is going to be a problem
		time.Sleep(time.Millisecond)
		runtime.Gosched() // single this
	}
}
