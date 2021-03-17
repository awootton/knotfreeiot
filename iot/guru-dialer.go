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
)

func (upc *upperChannel) dialGuru() {

	defer fmt.Println("** aren't we supposed to never quit this? **")

	isTCP := false
	if upc.ex.ce != nil && upc.ex.ce.isTCP {
		isTCP = true
	}

	upc.running = true
	if isTCP {
		// in prod:
		for upc.running {

			err := upc.dialGuruAndServe(upc.address)
			if err != nil {
				fmt.Println("dialGuru dialGuruAndServe err", upc.address, err)
			} else {
				// there's always an error or else we'd still be in dialAideAndServe
				fmt.Println("dialGuru dialGuruAndServe noerr", upc.address, err)
			}
			time.Sleep(time.Second)
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
			//pr, pw := io.Pipe()
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
				fmt.Println("don't wuit this ")
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
		fmt.Println("dial g fail", address, err)
		TCPNameResolverFail2.Inc()
		return nil
	}
	// have conn.(*net.TCPConn)

	TCPNameResolverConnected.Inc()

	conn.(*net.TCPConn).SetNoDelay(true)
	conn.(*net.TCPConn).SetWriteBuffer(4096)

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
				fmt.Println(" g err PushDown ", err)
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
			// mux.Lock()
			// err = p.Write(conn)
			// mux.Unlock()
			// if err != nil {
			// 	fmt.Println("write ping fail", conn, err)
			// 	conn.Close()
			// 	founderr = err
			// }
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
