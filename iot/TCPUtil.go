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
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
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

	ex.Looker.NameResolver = TCPNameResolver

	go server(ex, serverName)

	return ex
}

func hello(w http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(w, "hello from, atw\n")
}

type apiHandler struct {
	ex *Executive
}

func (api apiHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {

	if req.RequestURI == "/api1/getstats" {

		stats := api.ex.GetExecutiveStats()
		bytes, err := json.Marshal(stats)
		if err != nil {
			fmt.Println("GetExecutiveStats marshal", err)
		}
		w.Write(bytes)

	} else {
		http.NotFound(w, req)
		//fmt.Fprintf(w, "expected known path "+api.ex.Name)
	}
	//fmt.Fprintf(w, "hello from, atw and "+api.ex.Name)
	//fmt.Println(req.RequestURI) //api1/getstats
}

// MakeHTTPExecutive sets up an http server
func MakeHTTPExecutive(ex *Executive, serverName string) *Executive {

	mux := http.NewServeMux()
	mux.HandleFunc("/hello", hello)
	mux.Handle("/api1/", apiHandler{ex})

	s := &http.Server{
		Addr:           serverName,
		Handler:        mux,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	go func(s *http.Server) {
		fmt.Println("http service " + s.Addr)
		err := s.ListenAndServe()
		_ = err
		fmt.Println("ListenAndServe returned !!!!!  arrrrg", err)
	}(s)
	return ex
}

// type GuruNameResolver func(name string, config *ContactStructConfig) (ContactInterface, error)

// TCPNameResolver is a socket factory producing tcp connected sockets for the top of an aide.
func TCPNameResolver(address string, config *ContactStructConfig) (ContactInterface, error) {

	ce := config.ce
	if ce == nil {
		return nil, errors.New("need ce")
	}

	cc := &tcpUpperContact{}
	InitUpperContactStruct(&cc.ContactStruct, config)

	servAddr := address //"127.0.0.1:7654"
	tcpAddr, err := net.ResolveTCPAddr("tcp", servAddr)
	if err != nil {
		fmt.Println("fixme kkk", tcpAddr, err)
		return cc, err
	}
	go func(cc *tcpUpperContact) {
		failed := 0
		for {
			// we're very agressive so not time.Sleep(100 << failed * time.Millisecond)
			// todo: tell prometheius we're dialing
			conn, err := net.DialTimeout("tcp", address, time.Duration(uint64(time.Second)*uint64(cc.GetConfig().defaultTimeoutSeconds)))
			if err != nil {
				fmt.Println("dial 3 fail", address, err, failed)
				//return cc, err
				failed++
				continue
			}
			cc.tcpConn = conn.(*net.TCPConn)
			fmt.Println("tcpUpperContact connected", cc)

			conn.(*net.TCPConn).SetNoDelay(true)
			conn.(*net.TCPConn).SetWriteBuffer(4096)

			connect := packets.Connect{}
			connect.SetOption("token", []byte(tokens.SampleSmallToken)) // fixme, use token that the ex has
			cc.WriteUpstream(&connect)
			if err != nil {
				//return cc, err
				fmt.Println("push c fail", conn, err)
				failed++
			}
			fmt.Println("TCPNameResolver starting read loop ", cc)
			failed = 0
			for { // add timeout?
				// conn.SetReadDeadline(time.Duration(uint64(time.Second) * uint64(cc.GetConfig().defaultTimeoutSeconds)))
				//fmt.Println("waiting for packet from above ")
				packet, err := packets.ReadPacket(conn)
				if err != nil {
					fmt.Println("amke a note ", err)
					break
				}
				PushDown(cc, packet)
			}
		}
	}(cc)

	// don't return until it's connected.
	count := 0
	for cc.tcpConn == nil {
		time.Sleep(time.Millisecond)
		count++
		if count > 50 { // ?? how much ??
			return nil, errors.New("timeout trying to connect to " + address)
		}
	}

	return cc, nil
}

// called by Lookup PushUp
func (cc *tcpUpperContact) WriteUpstream(p packets.Interface) error {

	if cc.tcpConn == nil {
		fmt.Println("need non nil tcpConn", cc)
		return errors.New(fmt.Sprint("need non nil tcpConn", cc))
	}
	if cc.GetConfig() == nil {
		fmt.Println("we are closed", cc)
		return errors.New(fmt.Sprint("we are closed", cc))
	}
	err := p.Write(cc.tcpConn)
	if err != nil {
		cc.Close(err)
		return err
	}
	return nil
}

func (cc *tcpContact) Close(err error) {
	if cc.GetConfig() != nil {
		dis := packets.Disconnect{}
		dis.SetOption("error", []byte("nil config"))
		cc.WriteDownstream(&dis)
		cc.tcpConn.Close()
	}
	ss := cc.ContactStruct
	ss.Close(err) // close my parent
}

func (cc *tcpContact) WriteDownstream(packet packets.Interface) error {
	//fmt.Println("received from above", cmd, reflect.TypeOf(cmd))
	err := packet.Write(cc.tcpConn)
	if err != nil {
		cc.Close(err)
	}
	return err
}

func (cc *tcpContact) WriteUpstream(cmd packets.Interface) error {
	fmt.Println("FIXME tcp received from below", cmd, reflect.TypeOf(cmd))
	err := cmd.Write(cc.tcpConn)
	if err != nil {
		cc.Close(err)
	}
	return err
}

func server(ex *Executive, name string) {
	fmt.Println("tcp server starting", name)
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
			fmt.Println("accept err ", err)
			continue
		}
		go handleConnection(tmpconn.(*net.TCPConn), ex) //,handler types.ProtocolHandler)
	}
}

func handleConnection(tcpConn *net.TCPConn, ex *Executive) {

	// FIXME: all the *LogThing expressions need to be re-written for prom
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
			fmt.Println("packets 3 read err", err)
			cc.Close(err)
			return
		}
		//fmt.Println("tcp got packet", p, cc)
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
