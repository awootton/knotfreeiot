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
	"bufio"
	"fmt"
	"net"
	"time"

	"github.com/awootton/knotfreeiot/packets"
)

type textContact struct {
	tcpContact
}

// MakeTextExecutive is a thing like a server, not the exec
func MakeTextExecutive(ex *Executive, serverName string) *Executive {

	go textServer(ex, serverName)

	return ex

}

// a simple iot wire protocol that is text based.

func (cc *textContact) WriteDownstream(packet packets.Interface) error {
	text := packet.String()
	bytes := []byte(text + "\n")
	//fmt.Println("writing down ", string(bytes))
	_, err := cc.tcpConn.Write(bytes)
	if err != nil {
		cc.Close(err)
	}
	return err
}

// textServer serves a line oriented text protocol
func textServer(ex *Executive, name string) {
	fmt.Println("Server starting ", name)
	ln, err := net.Listen("tcp", name)
	if err != nil {
		// handle error
		//srvrLogThing.Collect(err.Error())
		fmt.Println("server didnt' stary ", err)
		return
	}
	for ex.IAmBadError == nil {
		//fmt.Println("Server listening")
		tmpconn, err := ln.Accept()
		if err != nil {
			//	srvrLogThing.Collect(err.Error())
			fmt.Println("accetp err ", err)
			continue
		}
		go textConnection(tmpconn.(*net.TCPConn), ex) //,handler types.ProtocolHandler)
	}
}

func textConnection(tcpConn *net.TCPConn, ex *Executive) {

	//srvrLogThing.Collect("Conn Accept")
	lineReader := bufio.NewReader(tcpConn)

	cc := localMakeTextContact(ex.Config, tcpConn)
	defer cc.Close(nil)

	// connLogThing.Collect("new connection")

	err := SocketSetup(tcpConn)
	if err != nil {
		//connLogThing.Collect("server err " + err.Error())
		fmt.Println("setup err", err)
		return
	}
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
		str, err := lineReader.ReadString('\n')
		if len(str) > 0 {
			str = str[0 : len(str)-1]
		}
		//fmt.Println("got line ", str)
		if err != nil {
			//connLogThing.Collect("se err " + err.Error())
			fmt.Println("packets 1 read err", err)
			cc.Close(err)
			return
		}
		p, err := Text2Packet(str)
		if err != nil {
			//connLogThing.Collect("se err " + err.Error())
			fmt.Println("packets 2 read err", err)
			cc.Close(err)
			return
		}
		//fmt.Println("t got packet", p)
		err = Push(cc, p)
		if err != nil {
			//connLogThing.Collect("se err " + err.Error())
			fmt.Println("text.push err", err)
			cc.Close(err)
			return
		}
	}
}

// localMakeTextContact is a factory
func localMakeTextContact(config *ContactStructConfig, tcpConn *net.TCPConn) *textContact {
	contact1 := textContact{}
	AddContactStruct(&contact1.ContactStruct, &contact1, config)
	contact1.tcpConn = tcpConn
	return &contact1
}
