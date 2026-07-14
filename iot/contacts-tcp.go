package iot

import (
	"errors"
	"log"
	"net"
	"reflect"
	"time"

	"github.com/awootton/knotfreeiot/packets"
)

// When a contack is a TCP contact.

// The functions here describe a server of the 'packets' protocol.

type tcpContact struct {
	ContactStruct
	netDotTCPConn *net.TCPConn
}

func (cc *tcpContact) DoClosingWork(err error) {
	// do we need a mutex here?
	// No, it is only called from the one place and only once
	log.Println("tcpContact DoClosingWork con=", cc.GetKey().Sig(), err)
	if cc.netDotTCPConn != nil {
		//log.Println("close tcp ", cc.netDotTCPConn.RemoteAddr())
		cc.netDotTCPConn.Close()
		cc.netDotTCPConn = nil
	}
	ss := &cc.ContactStruct
	ss.DoClosingWork(err) // close my parent too
}

func (cc *tcpContact) IsClosed() bool {
	return cc.ContactStruct.IsClosed()
}

// WriteDownstream writes a packet to the tcp connection going towards the user.
// error returned always nil
func (cc *tcpContact) WriteDownstream(packet packets.Interface) error {

	if cc.IsClosed() {
		return errors.New("tcpContact closed and can't writeDownstream")
	}
	got, ok := packet.GetOption("debg")
	isDebug := ok && string(got) == "12345678"
	if isDebug {
		log.Println("tcpContact WriteDownstream con=", cc.GetKey().Sig(), packet.Sig())
	}
	// var goterr error
	// var wg sync.WaitGroup we don't need to wait. It's in the Q and that's enough.
	// wg.Add(1)
	cc.commands <- ContactCommander{
		who: "tcp WriteDownstream",
		fn: func(ss *ContactStruct) {
			//defer wg.Done()
			u := HasError(packet)
			if u != nil && !cc.config.IsGuru() {
				log.Println("tcpContact ERROR write disconnect con=", cc.GetKey().Sig(), packet.Sig())
				u.Write(cc) // write disconnect
				cc.DoClose(errors.New(u.String()))
				_ = errors.New(u.String())
			} else {
				if cc.netDotTCPConn == nil {
					return
				}
				// if isDebug {
				// 	log.Println("tcpContact in the socket con=", cc.GetKey().Sig(), packet.Sig())
				// }
				// this must not block, otherwise the whole channel gets stuck.
				err := packet.Write(cc)
				// cc.netDotTCPConn.SetNoDelay(true)
				// when do we flush? do we need to flush?
				if err != nil {
					log.Println("tcpContact write ERROR con=", cc.GetKey().Sig(), err)
					// close the connection???
				}
			}
		},
	}
	return nil
}

func (cc *tcpContact) WriteUpstream(cmd packets.Interface) error {
	log.Println("FIXME tcp received from below dead code ERROR delete me", cmd, reflect.TypeOf(cmd))
	err := cmd.Write(cc)
	if err != nil {
		cc.DoClose(err)
	}
	return err
}

func listenForPacketsConnect(ex *Executive, name string) {
	log.Println("TCPUtil listen top ", name, "at", ex.Name, "with", ex.GetTCPAddress())
	ln, err := net.Listen("tcp", name)
	if err != nil {
		// handle error
		//srvrLogThing.Collect(err.Error())
		//log.Println("server didnt' stary ", err)
		TCPServerDidntStart.Inc()
		return
	}
	for {
		log.Println("TCPUtil Server listening for packets connections ", name, ex.Name)
		tmpconn, err := ln.Accept()
		if err != nil {
			//	srvrLogThing.Collect(err.Error())
			//log.Println("accept err ", err)
			TCPServerAcceptError.Inc()
			continue
		}
		log.Println("TCPUtil Server accepted connection ", name, ex.Name, tmpconn.RemoteAddr())

		go handleConnection(tmpconn.(*net.TCPConn), ex)
	}
}

func handleConnection(tcpConn *net.TCPConn, ex *Executive) {

	// FIXME: all the *LogThing expressions in package need to be re-written for prom
	//srvrLogThing.Collect("Conn Accept")
	TCPServerConnAccept.Inc() // <-- like this

	cc := localMakeTCPContact(ex.Config, tcpConn)
	defer func() {
		// log.Println("tcpContact handleConnection exit close")
		cc.DoClose(nil)
	}()

	log.Println("tcpContact add, ", tcpConn.RemoteAddr(), cc.GetKey().Sig(), ex.Name)

	TCPServerNewConnection.Inc()

	err := SocketSetup(tcpConn)
	if err != nil {
		//connLogThing.Collect("server err " + err.Error())
		log.Println("setup err", err)
		return
	}

	// defer log.Println("tcpContact QUIT, ", tcpConn.RemoteAddr(), cc.GetKey().Sig(), ex.Name)

	// we might just for over the range of the handler input channel?
	for !ex.IsClosed() {

		// SetReadDeadline
		if cc.GetToken() == nil {
			err := cc.netDotTCPConn.SetDeadline(time.Now().Add(2 * time.Second))
			if err != nil {
				log.Println("tcpContact deadline err 3", err)
				cc.DoClose(err)
				return // quit, close the sock, be forgotten
			}
		} else {
			err := cc.netDotTCPConn.SetDeadline(time.Now().Add(30 * time.Minute))
			if err != nil {
				log.Println("tcpContact deadline err 4", err, tcpConn.RemoteAddr())
				cc.DoClose(err)
				return // quit, close the sock, be forgotten, start over
			}
		}

		// read a packet and push it up from the bottom.
		// I wish I knew if this contact needs bigger buffers. It might be dialAideAndServe

		// log.Println("tcpContact waiting for packet con=", cc.GetKey().Sig(), ex.Name)
		p, err := packets.ReadPacket(cc)
		if err != nil {
			// We get these whenever someone dials, does someting and then disconnects.
			// Like this little fucker: PublishTestTopic Dialing  :8384
			// it's fairly normal.
			log.Println("tcpContact read closed", time.Now(), cc.key.Sig(), err, tcpConn.RemoteAddr(), ex.isGuru, ex.Name)
			TCPServerPacketReadError.Inc()
			cc.DoClose(err)
			return
		}
		// how can I tell if this is an channelToAnyAide packet? I can't print ALL of these.
		{
			skipMe := false // make this a case stmt.
			_, isSub := p.(*packets.Subscribe)
			if isSub {
				skipMe = true
			}
			_, isUnSub := p.(*packets.Unsubscribe)
			if isUnSub {
				skipMe = true
			}
			_, isCon := p.(*packets.Connect)
			if isCon {
				skipMe = true
			}
			// ? cc.LogMeVerbose
			// what are all the empty sends I'm getting? eg. tcpContact got packet  bvp1 send to:ET9M frm:w2q_ with: guru-20e1689727145d45d663f2696fd88183
			if !skipMe && serviceDebugSession1 {
				log.Println("tcpContact got packet ", cc.GetKey().Sig(), "packet sig", p.Sig(), "my name", ex.Name)
			}
		}

		err = PushPacketUpFromBottom(cc, p)
		if err != nil {
			//connLogThing.Collect("se err " + err.Error())
			log.Println("iot.push err", err, tcpConn.RemoteAddr())
			TCPServerIotPushError.Inc()
			cc.DoClose(err)
			return
		}
	}
}

// SocketSetup sets common options
func SocketSetup(tcpConn *net.TCPConn) error {
	//tcpConn := conn.(*net.TCPConn)
	err := tcpConn.SetReadBuffer(4096 * 16)
	if err != nil {
		log.Println("tcpContact SocketSetup err1 " + err.Error())
		return err
	}
	err = tcpConn.SetWriteBuffer(4096 * 16)
	if err != nil {
		log.Println("tcpContact SocketSetup err2 " + err.Error())
		return err
	}
	err = tcpConn.SetNoDelay(true)
	if err != nil {
		log.Println("tcpContact SocketSetup err3 " + err.Error())
		return err
	}
	// SetReadDeadline and SetWriteDeadline

	err = tcpConn.SetDeadline(time.Now().Add(20 * time.Minute))
	if err != nil {
		log.Println("tcpContact SocketSetup err4 " + err.Error())
		return err
	}
	return nil
}

// localMakeTCPContact is a factory
func localMakeTCPContact(config *ContactStructConfig, tcpConn *net.TCPConn) *tcpContact {
	contact1 := tcpContact{}

	AddContactStruct(&contact1.ContactStruct, &contact1, config)
	contact1.netDotTCPConn = tcpConn
	contact1.realReader = tcpConn
	contact1.realWriter = tcpConn

	return &contact1
}

// Copyright 2019,2020,2021,2026 Alan Tracey Wootton
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
