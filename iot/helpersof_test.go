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
	"time"

	"github.com/awootton/knotfreeiot/badjson"
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
}

type testUpperContact struct {
	iot.ContactStruct

	guruBottomContact *testContact
}

func MakeTestContact(config *iot.ContactStructConfig) iot.ContactInterface {
	contact1 := testContact{}
	contact1.downMessages = make(chan packets.Interface, 100)
	contact1.mostRecent = make([]packets.Interface, 0, 100)

	go func(cc *testContact) {
		for {
			thing := <-contact1.downMessages

			if reflect.TypeOf(thing) == reflect.TypeOf(&packets.Disconnect{}) {
				// now we have to reconnect
				fmt.Println("contact reattaching", cc)
				globalClusterExec.AttachContact(cc, AttachTestContact)
				// we should also reiterate our connect and our subscription.
				SendText(cc, "S "+cc.String())
			} else {
				cc.mostRecent = append(cc.mostRecent, thing)
			}
		}
	}(&contact1)
	iot.AddContactStruct(&contact1.ContactStruct, config)

	connect := packets.Connect{}
	connect.SetOption("token", []byte(tokens.SampleSmallToken))
	iot.Push(&contact1, &connect)

	return &contact1
}

// the contact is already made but got closed or something and needs to
// re-attach
func AttachTestContact(cc iot.ContactInterface, config *iot.ContactStructConfig) {
	contact1 := cc.(*testContact)
	contact1.downMessages = make(chan packets.Interface, 100)
	iot.AddContactStruct(&contact1.ContactStruct, config)

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
		iot.AddContactStruct(&newLowerContact.ContactStruct, exec.Config)

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

func (cc *testContact) get() (packets.Interface, bool) {
	select {
	case msg := <-cc.downMessages:
		return msg, true
	case <-time.After(10 * time.Millisecond):
		return nil, false
	}
}

func (cc *testContact) Close(err error) {
	ss := cc.ContactStruct
	ss.Close(err)

	dis := packets.Disconnect{}
	dis.SetOption("error", []byte(err.Error()))
	cc.WriteDownstream(&dis)
}

func (cc *testContact) getResultAsString() string {
	gotmsg, ok := cc.get()
	got := ""
	if ok {
		got = gotmsg.String()
	} else {
		got = fmt.Sprint("no message received", gotmsg)
	}
	return got
}

func (cc *testContact) WriteDownstream(packet packets.Interface) {
	//fmt.Println("received from above", cmd, reflect.TypeOf(cmd))
	cc.downMessages <- packet
}

func (cc *testContact) WriteUpstream(cmd packets.Interface) {
	fmt.Println("FIXME received from below", cmd, reflect.TypeOf(cmd))
	//cc.downMessages <- cmd
}

func readCounter(m prometheus.Counter) float64 {
	pb := &dto.Metric{}
	m.Write(pb)
	return pb.GetCounter().GetValue()
}

// SendText chops up the text and creates a packets.Interface packet.
func SendText(cc iot.ContactInterface, text string) {

	// parse the text
	segment, err := badjson.Chop(text)
	if err != nil {
		fmt.Println(err)
	}
	uni := packets.Universal{}
	uni.Args = make([][]byte, 64) // much too big
	tmp := segment.Raw()          // will not be quoted
	uni.Cmd = packets.CommandType(tmp[0])
	segment = segment.Next()

	// traverse the result
	i := 0
	for s := segment; s != nil; s = s.Next() {
		stmp := s.Raw()
		uni.Args[i] = []byte(stmp)
		i++
		if i > 10 {
			break
		}
	}
	p, err := packets.FillPacket(&uni)
	if err != nil {
		fmt.Println("problem with packet", err)
	}
	iot.Push(cc, p)
	//cctest, _ := cc.(*testContact)
	//got := cctest.getResultAsString()
	//_ = got
}
