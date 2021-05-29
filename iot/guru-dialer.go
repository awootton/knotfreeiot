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

package iot

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/awootton/knotfreeiot/packets"
	"github.com/awootton/knotfreeiot/tokens"
	"github.com/dgryski/go-maglev"
)

// ConnectGuruToSuperAide for testing a cluster with a supercluster
// we need channels from guru to aide.
func ConnectGuruToSuperAide(guru *Executive, aide *Executive) {

	me := *guru.Looker // *LookupTableStruct

	names := []string{aide.Name}
	addresses := []string{"noaddr"}

	GuruNameToConfigMap[aide.Name] = aide // for lookup later

	//guru.Looker.SetUpstreamNames(names, addresses)

	// don't block on anything.
	router := me.upstreamRouter
	router.mux.Lock()
	defer router.mux.Unlock()

	if len(names) != len(addresses) {
		fmt.Println("error len(names) != len(addresses) panic")
		return
	}

	if len(names) == len(router.channels) {
		for i, c := range router.channels {
			if names[i] != c.name {
				return // no changes
			}
		}
	}
	// maybe some more verifications?

	// if me.isGuru {
	// 	me.setGuruUpstreamNames(names)
	// 	return
	// }

	// we're a guru.
	oldContacts := router.channels

	router.channels = make([]*upperChannel, len(names))
	theNamesThisTime := make(map[string]string, len(names))

	// iterate the names passed in.
	// constuct a new list. populate it with existing upc
	// when possible

	for i, name := range names {
		address := addresses[i]
		theNamesThisTime[name] = address

		upc, found := router.name2channel[name]
		if found && upc.running {
			router.channels[i] = upc
		} else {
			fmt.Println("starting upper router from ", me.ex.Name, " to ", name)
			upc = &upperChannel{}
			upc.name = name
			upc.address = address
			upc.up = make(chan packets.Interface, 1280)
			upc.down = make(chan packets.Interface, 128)
			upc.ex = me.ex
			router.channels[i] = upc
			go upc.dialGuru()
		}
	}
	// lose the stale ones
	for _, upc := range oldContacts {
		_, found := theNamesThisTime[upc.name]
		if found == false {
			upc.running = false
			fmt.Println("forgetting upper router ", upc.name)
			close(upc.up)
			close(upc.down)
			delete(router.name2channel, upc.name)
		}
	}

	router.previousmaglev = router.maglev
	maglevsize := maglev.SmallM
	if DEBUG {
		maglevsize = 97
	}
	router.maglev = maglev.New(names, uint64(maglevsize))
	// order subscriptions to be forwarded to the new UpContact.

	// iterate all the subscriptions and push up (again) the ones that have been remapped.
	// iterate all subscriptions and delete the ones that don't map here anymore.
	// note that this will need to push up to the g u r u through the conn
	// just defined and it can BLOCK until the conn completes.
	// we must watch those buffers and not block here.
	go func() {
		command := callBackCommand{}
		command.callback = reSubscribeRemappedTopics

		for _, bucket := range me.allTheSubscriptions {
			command.wg.Add(1)
			bucket.incoming <- &command
		}
		command.wg.Wait()
	}()

}

// dialGuru is used to connect an aide to a guru.
// if it's tcp then we open sockets and connect them to the channel
// if not then we simply make a contact in the guru and wire it to the channel
// note that the channel has a hidden arg for which guru to target
// we need a version of this to connect a guru to an aide in a super cluster.
func (upc *upperChannel) dialGuru() {

	defer func() {
		fmt.Println("** aren't we supposed to never quit this? **")
	}()

	isTCP := false
	if upc.ex.ce != nil && upc.ex.ce.isTCP {
		isTCP = true
	}

	upc.running = true
	if isTCP {
		// in prod:
		fmt.Println("dialGuru dialGuruAndServe started with", upc.address)
		for upc.running {

			err := upc.dialGuruAndServe(upc.address)
			if err != nil {
				fmt.Println("dialGuru dialGuruAndServe err", upc.address, err)
			} else {
				// there's always an error or else we'd still be in dialAideAndServe
				fmt.Println("dialGuru dialGuruAndServe noerr", upc.address, err)
			}
			time.Sleep(time.Second * 5)
		}

	} else {
		for upc.running {

			// the test we can find all the other nodes in the ce.
			// or in GuruNameToConfigMap
			config, found := GuruNameToConfigMap[upc.name]
			if !found {
				panic("all your tests will fail")
			}
			guru := config.Looker.ex // the ex of the guru

			token := tokens.GetImpromptuGiantToken()
			contact := &ContactStruct{}
			AddContactStruct(contact, contact, guru.Config) // force contact to guru

			contact.SetExpires(contact.contactExpires + 60*60*24*365*10) // in 10 years

			defer contact.Close(errors.New("finished"))

			fmt.Println("upc start from ", upc.ex.Name, " to ", guru.Name)

			// when the guru writes we want to get that
			// and put it in upc.down
			myPipe := newMyPipe()
			myPipe.name = upc.ex.Name
			contact.realWriter = myPipe
			contact.realReader = myPipe

			go upc.readFromPipe(myPipe) // and q into upc.down

			go func() {
				for p := range upc.down {
					//fmt.Println("upc.down ", p)
					err := PushDownFromTop(upc.ex.Looker, p)
					if err != nil {
						fmt.Println(" UPC err PushDown ", err)
					}
				}
				fmt.Println("don't to be here. the upc.down channel should never close  ")
			}()

			connect := packets.Connect{}
			connect.SetOption("token", []byte(token))
			err := PushPacketUpFromBottom(contact, &connect)
			if err != nil {
				fmt.Println("connect guru test dial conn ", err)
			}
			for p := range upc.up {

				//fmt.Println("UPC pushing to guru ", p)
				// needs to be cloned because it's still also in aide
				buff := &bytes.Buffer{}
				p.Write(buff)
				buff2 := bytes.NewBuffer(buff.Bytes())
				p2, err := packets.ReadPacket(buff2)

				if contact.GetClosed() {
					continue
				}
				err = PushPacketUpFromBottom(contact, p2)
				if err != nil {
					fmt.Println(" UPC err pushing to aide ", err)
					break
				}
			}
			fmt.Println("don't quit who closed the chan")
		}
	}
}

// myPipe wrapper of Pipe
type myPipe struct {
	pr *io.PipeReader
	pw *io.PipeWriter

	written string
	readden string

	name string
}

func newMyPipe() *myPipe {
	m := &myPipe{}
	pr, pw := io.Pipe()
	m.pr = pr
	m.pw = pw
	m.written = ""
	return m
}

func (m *myPipe) String() string {
	return m.name + ": " + hex.EncodeToString([]byte(m.readden)) + "*" + hex.EncodeToString([]byte(m.written))
}

func (m *myPipe) Read(data []byte) (int, error) {

	n, err := m.pr.Read(data)

	// got := string(data[0:n])
	// m.readden = m.readden + got
	// m.written = m.written[n:]

	return n, err
}

func (m *myPipe) Write(data []byte) (int, error) {

	// m.written = m.written + string(data)
	// fmt.Println("pw", m)

	n, err := m.pw.Write(data)

	return n, err
}

func (upc *upperChannel) readFromPipe(m *myPipe) {
	for {
		p, err := packets.ReadPacket(m)

		if err != nil {
			fmt.Println("readFromPipe ReadPacket err  ", err)
			break // it will never resync or recover
		} else {
			//fmt.Println("test got packet from guru  ", p)

			// fmt.Println("rp", m)
			// buff := &bytes.Buffer{}
			// p.Write(buff)
			// s := string(buff.Bytes())
			// if strings.HasPrefix(m.readden, s) {
			// 	m.readden = m.readden[len(s):]
			// 	fmt.Println("rp", m)
			// }

			upc.down <- p
		}
	}
}

func (upc *upperChannel) dialGuruAndServe(address string) error {

	fmt.Println("top of dialGuruAndServe ", address, upc.name, " for ", upc.ex.Name)

	// todo: tell prometheius we're dialing
	conn, err := net.DialTimeout("tcp", address, time.Duration(uint64(2*time.Second)))
	if err != nil {
		fmt.Println("dial dialGuruAndServe fail", address, " with ", err)
		TCPNameResolverFail2.Inc()
		return nil
	}
	// have conn.(*net.TCPConn)

	TCPNameResolverConnected.Inc()

	conn.(*net.TCPConn).SetNoDelay(true)
	conn.(*net.TCPConn).SetWriteBuffer(4096)

	fmt.Println("dialGuruAndServe ready to ReadPacket from ", address)

	var founderr error

	upc.down = nil // not used
	go func() {
		for founderr == nil {
			p, err := packets.ReadPacket(conn)
			if err != nil {
				fmt.Println("dialGuruAndServe readPacket err", p, err, address)
				founderr = err
				conn.Close()
				return
			}
			//fmt.Println("not upc.down ", p)
			err = PushDownFromTop(upc.ex.Looker, p)
			if err != nil {
				fmt.Println("dialGuruAndServe PushDownFromTop error ", err)
				founderr = err
				conn.Close()
				return
			}
		}
	}()

	upc.down = nil     // not used
	var mux sync.Mutex // needed?

	connect := &packets.Connect{}
	connect.SetOption("token", []byte(tokens.GetImpromptuGiantToken()))
	mux.Lock()
	err = connect.Write(conn)
	mux.Unlock()
	if err != nil {
		fmt.Println("write c fail", conn, err)
		conn.Close()
		founderr = err
		return err
	}

	go func() {
		for founderr == nil {
			time.Sleep(time.Second)
			p := &packets.Ping{}
			upc.up <- p
		}
	}()

	for p := range upc.up {
		mux.Lock()
		err := p.Write(conn)
		mux.Unlock()
		//fmt.Println("L pushing to guru ", p)
		if err != nil || founderr != nil {
			if err == nil {
				err = founderr
			}
			if founderr == nil {
				founderr = err
			}
			fmt.Println("err L pushing to guru ", err)
			conn.Close()
			return err
		}
	}
	return nil
}
