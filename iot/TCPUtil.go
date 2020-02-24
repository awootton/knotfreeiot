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
	"fmt"
	"net"
	"reflect"
	"time"

	"github.com/awootton/knotfreeiot/packets"
	"github.com/awootton/knotfreeiot/tokens"
)

// The functions here describe a server of the 'packets' protocol.

// a typical bottom contact with a q instead of a writer
type tcpContact struct {
	ContactStruct

	tcpConn *net.TCPConn
}

type tcpUpperContact struct {
	tcpContact

	doNotReconnect bool
}

// MakeTCPExecutive is a thing like a server, not the exec
func MakeTCPExecutive(ex *Executive, serverName string) *Executive {

	ex.Looker.NameResolver = tcpNameResolver

	go server(ex, serverName)

	return ex

}

// type GuruNameResolver func(name string, config *ContactStructConfig) (ContactInterface, error)

func tcpNameResolver(name string, config *ContactStructConfig) (ContactInterface, error) {

	cc := &tcpUpperContact{}
	InitUpperContactStruct(&cc.ContactStruct, config)

	servAddr := name //"127.0.0.1:7654"
	tcpAddr, err := net.ResolveTCPAddr("tcp", servAddr)
	if err != nil {
		fmt.Println("fixme kkk", tcpAddr, err)
		return cc, err
	}
	for {
		conn, err := net.DialTimeout("tcp", name, time.Duration(uint64(time.Second)*uint64(cc.GetConfig().defaultTimeoutSeconds)))
		fmt.Println(conn, err)
		conn.(*net.TCPConn).SetNoDelay(true)
		conn.(*net.TCPConn).SetWriteBuffer(4096)

		connect := packets.Connect{}
		connect.SetOption("token", []byte(tokens.SampleSmallToken)) // fixme, use token that the ex has
		Push(cc, &connect)

		for { // add timeout?
			// conn.SetReadDeadline(time.Duration(uint64(time.Second) * uint64(cc.GetConfig().defaultTimeoutSeconds)))
			packet, err := packets.ReadPacket(conn)
			if err != nil {
				fmt.Println("amke a note ", err)
				break
			}
			PushDown(cc, packet)
		}
	}
}

// called by Lookup PushUp
func (cc *tcpUpperContact) WriteUpstream(p packets.Interface) {

	err := p.Write(cc.tcpConn)
	if err != nil {
		cc.Close(err)
	}

}

func (cc *tcpContact) Close(err error) {

	fmt.Println("closing  ", err)

	if cc.GetConfig() != nil {
		dis := packets.Disconnect{}
		dis.SetOption("error", []byte(err.Error()))
		cc.WriteDownstream(&dis)
		cc.tcpConn.Close()
	}
	ss := cc.ContactStruct
	ss.Close(err) // close my parent
}

func (cc *tcpContact) WriteDownstream(packet packets.Interface) {
	//fmt.Println("received from above", cmd, reflect.TypeOf(cmd))
	err := packet.Write(cc.tcpConn)
	if err != nil {
		cc.Close(err)
	}
}

func (cc *tcpContact) WriteUpstream(cmd packets.Interface) {
	fmt.Println("FIXME tcp received from below", cmd, reflect.TypeOf(cmd))
	err := cmd.Write(cc.tcpConn)
	if err != nil {
		cc.Close(err)
	}
}

func server(ex *Executive, name string) {
	//fmt.Println("Server starting")
	ln, err := net.Listen("tcp", name)
	if err != nil {
		// handle error
		//srvrLogThing.Collect(err.Error())
		fmt.Println("server didnt' stary ", err)
		return
	}
	for {
		//fmt.Println("Server listening")
		tmpconn, err := ln.Accept()
		if err != nil {
			//	srvrLogThing.Collect(err.Error())
			fmt.Println("accetp err ", err)
			continue
		}
		go handleConnection(tmpconn.(*net.TCPConn), ex) //,handler types.ProtocolHandler)
	}
}

// RunAConnection - FIXME: this is really a protoA connection.
//
func handleConnection(tcpConn *net.TCPConn, ex *Executive) {

	//srvrLogThing.Collect("Conn Accept")

	cc := localMakeTCPContact(ex.Config, tcpConn)
	defer cc.Close(nil)

	// connLogThing.Collect("new connection")

	err := SocketSetup(tcpConn)
	if err != nil {
		//connLogThing.Collect("server err " + err.Error())
		fmt.Println("setup err", err)
		return
	}
	// we might just for over the range of the handler input channel?
	for ex.IAmBadError == nil {
		// SetReadDeadline
		err := cc.tcpConn.SetDeadline(time.Now().Add(20 * time.Minute))
		if err != nil {
			//connLogThing.Collect("server err2 " + err.Error())
			fmt.Println("deadline err", err)
			cc.Close(err)
			return // quit, close the sock, be forgotten
		}

		//fmt.Println("waiting for packet")

		p, err := packets.ReadPacket(cc.tcpConn)
		if err != nil {
			//connLogThing.Collect("se err " + err.Error())
			fmt.Println("packets read err", err)
			cc.Close(err)
			return
		}
		//fmt.Println("got packet", p)
		err = Push(cc, p)
		if err != nil {
			//connLogThing.Collect("se err " + err.Error())
			fmt.Println("iot.push err", err)
			cc.Close(err)
			return
		}
	}
}

// SocketSetup sets common options
//
func SocketSetup(tcpConn *net.TCPConn) error {
	//tcpConn := conn.(*net.TCPConn)
	err := tcpConn.SetReadBuffer(4096)
	if err != nil {
		//srvrLogThing.Collect("SS err1 " + err.Error())
		return err
	}
	err = tcpConn.SetWriteBuffer(4096)
	if err != nil {
		//srvrLogThing.Collect("SS err2 " + err.Error())
		return err
	}
	err = tcpConn.SetNoDelay(true)
	if err != nil {
		//	srvrLogThing.Collect("SS err3 " + err.Error())
		return err
	}
	// SetReadDeadline and SetWriteDeadline

	err = tcpConn.SetDeadline(time.Now().Add(20 * time.Minute))
	if err != nil {
		// /srvrLogThing.Collect("cl err4 " + err.Error())
		return err
	}
	return nil
}

// localMakeTCPContact is a factory
func localMakeTCPContact(config *ContactStructConfig, tcpConn *net.TCPConn) *tcpContact {
	contact1 := tcpContact{}

	AddContactStruct(&contact1.ContactStruct, &contact1, config)
	contact1.tcpConn = tcpConn

	return &contact1
}
