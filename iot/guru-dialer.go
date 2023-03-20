// Copyright 2019,2020,2021-2023 Alan Tracey Wootton
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
		fmt.Println("dialGuruAndServe started with", upc.address, upc.name)
		for upc.running {
			err := upc.dialGuruAndServe()
			if err != nil {
				fmt.Println("dialGuruAndServe returned err", err, upc.address, upc.name)
			} else {
				// there's always an error or else we'd still be in dialGureAndServe?
				fmt.Println("dialGureAndServe returned noerr", upc.address, upc.name)
			}
			time.Sleep(time.Second * 1)
		}

	} else {
		for upc.running { // this is a debug version.

			// the test we can find all the other nodes in the ce.
			// or in GuruNameToConfigMap
			config, found := GuruNameToConfigMap[upc.name]
			if !found {
				panic("all your tests will fail")
			}
			guru := config.Looker.ex // the ex of the guru

			token := tokens.Test32xToken //GetImpromptuGiantToken()
			contact := &ContactStruct{}
			AddContactStruct(contact, contact, guru.Config) // force contact to guru

			contact.SetExpires(contact.contactExpires + 60*60*24*365*10) // in 10 years

			defer func() {
				fmt.Println("guru-dialer finished")
				contact.Close(errors.New("finished"))
			}()

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
				if err != nil {
					_ = err
				}
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
			if len(upc.down) >= cap(upc.down) {
				fmt.Println("readFromPipe channel full")
			}
			upc.down <- p
		}
	}
}

// is supposed to until upc.running is false or some error
// if it fails then the caller (dialGuru) will restart it
func (upc *upperChannel) dialGuruAndServe() error {

	var err error

	upc.founderr = nil
	upc.conn = nil

	fmt.Println("starting/restarting dialGuruAndServe ", upc.address, upc.name, " for ", upc.index)

	// todo: tell prometheius we're dialing
	upc.conn, err = net.DialTimeout("tcp", upc.address, time.Duration(uint64(2*time.Second)))
	if err != nil {
		fmt.Println("dial dialGuruAndServe fail", upc.address, upc.name, " with ", err)
		TCPNameResolverFail2.Inc()
		return err
	}
	// have conn.(*net.TCPConn)

	TCPNameResolverConnected.Inc()

	upc.conn.(*net.TCPConn).SetNoDelay(true)
	upc.conn.(*net.TCPConn).SetWriteBuffer(4096)

	fmt.Println("dialGuruAndServe ready to ReadPacket from ", upc.address, upc.name)

	upc.down = nil // not used. wut? messages are pushed up directly to tcp

	var mux sync.Mutex // needed?

	connect := &packets.Connect{}
	connect.SetOption("token", []byte(tokens.GetImpromptuGiantToken()))
	mux.Lock()
	err = connect.Write(upc.conn)
	mux.Unlock()
	if err != nil {
		fmt.Println("dialGuruAndServe packets.Connect fail", upc.conn, err)
		upc.conn.Close()
		upc.founderr = err
		return err
	}

	go func() {
		command := callBackCommand{}
		command.callback = reSubscribeMyTopics
		command.index = upc.index

		for _, bucket := range upc.ex.Looker.allTheSubscriptions {
			command.wg.Add(1)
			if len(bucket.incoming)*4 >= cap(bucket.incoming)*3 {
				fmt.Println("dialGuru bucket.incoming channel full", bucket.index)
			}
			bucket.incoming <- &command
		}
		command.wg.Wait()
	}()

	go func() {
		for upc.founderr == nil && upc.running {
			time.Sleep(time.Second * 300)
			p := &packets.Ping{}
			if len(upc.up)*4 >= cap(upc.up)*3 {
				fmt.Println("dialGuru channel full")
			}
			upc.up <- p
		}
	}()

	// since we have a conn now...
	go func() {
		for upc.founderr == nil && upc.running {
			p, err := packets.ReadPacket(upc.conn) // guru sent this down to us
			if err != nil {
				fmt.Println("dialGuruAndServe readPacket err", p, err, upc.address, upc.name)
				upc.founderr = err
				upc.conn.Close()
				return
			}
			got, ok := p.GetOption("debg")
			if ok && string(got) == "12345678" {
				fmt.Println("dialguru receive", p.Sig())
			}
			err = PushDownFromTop(upc.ex.Looker, p)
			if err != nil {
				fmt.Println("dialGuruAndServe PushDownFromTop error ", err)
				upc.founderr = err
				upc.conn.Close()
				return
			}
		}
	}()

	for upc.founderr == nil && upc.running {
		var err error
		select {
		case p := <-upc.up:
			mux.Lock()
			err = p.Write(upc.conn)
			mux.Unlock()
		case <-time.After(time.Millisecond * 100):
		}
		if err != nil {
			upc.founderr = err
			fmt.Println("dialGuruAndServe err pushing to guru ", err, upc.address, upc.name, upc.conn.RemoteAddr())
			upc.conn.Close()
			return err
		}
	}
	return upc.founderr
}

// ConnectGuruToSuperAide for testing a cluster with a supercluster
// we need channels from guru to aide.
// you can be sure that this needs work.
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
			upc.index = i
			router.channels[i] = upc
			go upc.dialGuru() // to super aide?
		}
	}
	// lose the stale ones
	for _, upc := range oldContacts {
		_, found := theNamesThisTime[upc.name]
		if !found {
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
			if len(bucket.incoming)*4 >= cap(bucket.incoming)*3 {
				fmt.Println("super aide bucket.incoming is full error")
			}
			bucket.incoming <- &command
		}
		command.wg.Wait()
	}()

}
