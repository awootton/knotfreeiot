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
	"encoding/json"
	"errors"
	"fmt"
	"io"
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

	netDotTCPConn *net.TCPConn
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

type apiHandler struct {
	ex *Executive
}

func (api apiHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	//fmt.Println("req.RequestURI", req.RequestURI)
	if req.RequestURI == "/api1/getstats" {

		stats := api.ex.GetExecutiveStats()
		stats.Limits = api.ex.Limits
		bytes, err := json.Marshal(stats)
		if err != nil {
			fmt.Println("GetExecutiveStats marshal", err)
		}
		w.Write(bytes)

		API1GetStats.Inc()

	} else if req.RequestURI == "/api2/set" {
		decoder := json.NewDecoder(req.Body)
		args := &UpstreamNamesArg{}
		err := decoder.Decode(args)

		if err != nil {
			http.Error(w, "decode error", 500)
			API1PostGurusFail.Inc()
			return
		}
		API1PostGurus.Inc()
		if len(args.Names) > 0 && len(args.Names) == len(args.Addresses) {
			api.ex.Looker.SetUpstreamNames(args.Names, args.Addresses)
		}

	} else {
		http.NotFound(w, req)
		IotHTTP404.Inc()
	}
}

// MakeHTTPExecutive sets up a http server
func MakeHTTPExecutive(ex *Executive, serverName string) *Executive {

	mux := http.NewServeMux()
	mux.Handle("/api1/", apiHandler{ex})
	mux.Handle("/api2/", apiHandler{ex})

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
		//fmt.Println("fixme kkk", tcpAddr, err)
		TCPNameResolverFail1.Inc()
		_ = tcpAddr
		return cc, err
	}
	go func(cc *tcpUpperContact) {
		failed := 0
		for {
			// we're very agressive so not time.Sleep(100 << failed * time.Millisecond)
			// todo: tell prometheius we're dialing
			conn, err := net.DialTimeout("tcp", address, time.Duration(uint64(time.Second)*uint64(cc.GetConfig().defaultTimeoutSeconds)))
			if err != nil {
				//fmt.Println("dial 2 fail", address, err, failed)
				TCPNameResolverFail2.Inc()
				//return cc, err
				failed++
				if failed > 10 {
					time.Sleep(time.Duration(100*failed) * time.Millisecond)
				}
				continue
			}
			cc.netDotTCPConn = conn.(*net.TCPConn)
			cc.realReader = cc.netDotTCPConn
			cc.realWriter = cc.netDotTCPConn
			//fmt.Println("tcpUpperContact connected", cc)
			TCPNameResolverConnected.Inc()

			conn.(*net.TCPConn).SetNoDelay(true)
			conn.(*net.TCPConn).SetWriteBuffer(4096)

			connect := packets.Connect{}
			connect.SetOption("token", []byte(tokens.SampleSmallToken)) // FIXME: fixme, use token that the ex has
			cc.WriteUpstream(&connect)
			if err != nil {
				//return cc, err
				fmt.Println("push c fail", conn, err)
				failed++
			}
			//fmt.Println("TCPNameResolver starting read loop ", cc)
			failed = 0
			for { // add timeout?
				// conn.SetReadDeadline(time.Duration(uint64(time.Second) * uint64(cc.GetConfig().defaultTimeoutSeconds)))
				//fmt.Println("waiting for packet from above ")
				packet, err := packets.ReadPacket(conn)
				if err != nil {
					fmt.Println("amke a note ", err) // FIXME: inc counter in prom
					break
				}
				PushDown(cc, packet)
			}
		}
	}(cc)

	// don't return until it's connected.
	count := 0
	for cc.netDotTCPConn == nil {
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

	if cc.GetConfig() == nil {
		fmt.Println("closed already", cc)
		return errors.New(fmt.Sprint("closed already", cc))
	}
	if cc.netDotTCPConn == nil {
		fmt.Println("need non nil netDotTCPConn", cc)
		return errors.New(fmt.Sprint("need non nil tcpConn", cc))
	}
	err := p.Write(cc)
	if err != nil {
		cc.Close(err)
		return err
	}
	return nil
}

func (cc *tcpContact) Close(err error) {
	hadConfig := cc.GetConfig() != nil
	ss := &cc.ContactStruct
	ss.Close(err) // close my parent
	if hadConfig {
		dis := packets.Disconnect{}
		dis.SetOption("error", []byte(err.Error()))
		cc.WriteDownstream(&dis)
		cc.netDotTCPConn.Close()
	}
}

func (cc *tcpContact) WriteDownstream(packet packets.Interface) error {
	//fmt.Println("received from above", cmd, reflect.TypeOf(cmd))
	if cc.GetClosed() == false && cc.GetConfig().GetLookup().isGuru == false {
		u := HasError(packet)
		if u != nil {
			u.Write(cc)
			cc.Close(errors.New(u.String()))
			return errors.New(u.String()) // ?
		}
	}
	err := packet.Write(cc)
	if err != nil {
		cc.Close(err)
	}
	return err
}

func (cc *tcpContact) WriteUpstream(cmd packets.Interface) error {
	fmt.Println("FIXME tcp received from below dead code", cmd, reflect.TypeOf(cmd))
	err := cmd.Write(cc)
	if err != nil {
		cc.Close(err)
	}
	return err
}

func server(ex *Executive, name string) {
	fmt.Println("knotfree server starting", name)
	ln, err := net.Listen("tcp", name)
	if err != nil {
		// handle error
		//srvrLogThing.Collect(err.Error())
		//fmt.Println("server didnt' stary ", err)
		TCPServerDidntStart.Inc()
		return
	}
	for {
		//fmt.Println("Server listening")
		tmpconn, err := ln.Accept()
		if err != nil {
			//	srvrLogThing.Collect(err.Error())
			//fmt.Println("accept err ", err)
			TCPServerAcceptError.Inc()
			continue
		}
		go handleConnection(tmpconn.(*net.TCPConn), ex) //,handler types.ProtocolHandler)
	}
}

func handleConnection(tcpConn *net.TCPConn, ex *Executive) {

	// FIXME: all the *LogThing expressions need to be re-written for prom
	//srvrLogThing.Collect("Conn Accept")
	TCPServerConnAccept.Inc()

	cc := localMakeTCPContact(ex.Config, tcpConn)
	defer cc.Close(nil)

	// connLogThing.Collect("new connection")
	TCPServerNewConnection.Inc()

	err := SocketSetup(tcpConn)
	if err != nil {
		//connLogThing.Collect("server err " + err.Error())
		fmt.Println("setup err", err)
		return
	}
	// we might just for over the range of the handler input channel?
	for ex.IAmBadError == nil {
		// SetReadDeadline
		if cc.GetToken() == nil {
			err := cc.netDotTCPConn.SetDeadline(time.Now().Add(2 * time.Second))
			if err != nil {
				fmt.Println("deadline err 3", err)
				cc.Close(err)
				return // quit, close the sock, be forgotten
			}
		} else {
			err := cc.netDotTCPConn.SetDeadline(time.Now().Add(20 * time.Minute))
			if err != nil {
				fmt.Println("deadline err 4", err)
				cc.Close(err)
				return // quit, close the sock, be forgotten
			}
		}
		//fmt.Println("waiting for packet")

		p, err := packets.ReadPacket(cc)
		if err != nil {
			//connLogThing.Collect("se err " + err.Error())
			//fmt.Println("packets 3 read err", err)
			TCPServerPacketReadError.Inc()
			cc.Close(err)
			return
		}
		//fmt.Println("tcp got packet", p, cc)
		err = Push(cc, p)
		if err != nil {
			//connLogThing.Collect("se err " + err.Error())
			//fmt.Println("iot.push err", err)
			TCPServerIotPushEror.Inc()
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
		fmt.Println("SS err1 " + err.Error())
		return err
	}
	err = tcpConn.SetWriteBuffer(4096)
	if err != nil {
		fmt.Println("SS err2 " + err.Error())
		return err
	}
	err = tcpConn.SetNoDelay(true)
	if err != nil {
		fmt.Println("SS err3 " + err.Error())
		return err
	}
	// SetReadDeadline and SetWriteDeadline

	err = tcpConn.SetDeadline(time.Now().Add(20 * time.Minute))
	if err != nil {
		fmt.Println("cl err4 " + err.Error())
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

// GetServerStats asks nicely over http
func GetServerStats(addr string) *ExecutiveStats {
	//result := ""
	stats := ExecutiveStats{}

	client := http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get("http://" + addr + "/api1/getstats")

	if err == nil && resp.StatusCode == 200 {
		var bytes [1024]byte
		n, err := resp.Body.Read(bytes[:])
		// = string(bytes[0:n])
		//fmt.Println("GetServerStats returned ", resp, err)

		err = json.Unmarshal(bytes[0:n], &stats)
		_ = err
	} else {
		fmt.Println("GetServerStats failed ", resp, err)
	}
	return &stats
}

// UpstreamNamesArg just has the one job
type UpstreamNamesArg struct {
	Names     []string
	Addresses []string
}

// PostUpstreamNames does SetUpstreamNames the hard way
func PostUpstreamNames(guruList []string, addressList []string, addr string) error {

	arg := &UpstreamNamesArg{}
	arg.Names = guruList
	arg.Addresses = addressList

	jbytes, err := json.Marshal(arg)
	if err != nil {
		fmt.Println("unreachable ?? bb")
		return errors.New("upstreamNamesArg marshal fail")
	}

	resp, err := http.Post("http://"+addr+"/api2/set", "application/json", bytes.NewReader(jbytes))
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return errors.New("upstreamNamesArg not 200")
	}
	return nil
}

// ByteCountingReader keeps track of how much was read.
type ByteCountingReader struct {
	count      int
	realReader io.Reader
}

func (bcr *ByteCountingReader) Read(p []byte) (int, error) {
	n, err := bcr.realReader.Read(p)
	bcr.count += n
	return n, err
}

// ByteCountingWriter keeps track of how much was written.
type ByteCountingWriter struct {
	count      int
	realWriter io.Writer
}

func (bcw *ByteCountingWriter) Write(p []byte) (int, error) {
	n, err := bcw.realWriter.Write(p)
	bcw.count += n
	return n, err
}
