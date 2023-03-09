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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/awootton/knotfreeiot/packets"
)

// The functions here describe a server of the 'packets' protocol.

type tcpContact struct {
	ContactStruct
	netDotTCPConn *net.TCPConn
}

// MakeTCPExecutive is a thing like a server, not the exec
func MakeTCPExecutive(ex *Executive, serverName string) *Executive {

	go listenForPacketsConnect(ex, serverName)

	return ex
}

type apiHandler struct {
	ex *Executive
}

func (api apiHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	//fmt.Println("req.RequestURI", req.RequestURI)
	if req.RequestURI == "/api2/getstats" { // GET
		// return the stats for just me.

		stats := api.ex.GetExecutiveStats()
		stats.Limits = api.ex.Limits
		bytes, err := json.Marshal(stats)
		if err != nil {
			fmt.Println("GetExecutiveStats marshal", err)
		}
		w.Write(bytes)

		API1GetStats.Inc()

	} else if req.RequestURI == "/api2/set" { // POST
		decoder := json.NewDecoder(req.Body)
		args := &UpstreamNamesArg{}
		err := decoder.Decode(args)
		if err != nil {
			http.Error(w, "decode error", 500)
			API1PostGurusFail.Inc()
			return
		}
		fmt.Println("/api2/set SetUpstreamNames len=", len(args.Names))
		API1PostGurus.Inc()
		if len(args.Names) > 0 && len(args.Names) == len(args.Addresses) {
			api.ex.Looker.SetUpstreamNames(args.Names, args.Addresses)
		} else {
			fmt.Println("bad names sent", args.Names, args.Addresses, args)
		}
		//fmt.Println("/api2/set done")

	} else if req.RequestURI == "/api2/clusterstats" { // POST

		// todo: add security.

		data, err := io.ReadAll(req.Body)
		if err != nil {
			http.Error(w, "read error 2", 500)
			API1PostGurusFail.Inc()
			return
		}
		stats := &ClusterStats{}
		err = json.Unmarshal(data, stats)
		if err != nil {
			http.Error(w, "decode error 2", 500)
			API1PostGurusFail.Inc()
			return
		}
		statsstr, error := json.Marshal(stats)
		fmt.Println("have new clusterstats ")
		_ = statsstr
		//fmt.Println("have new clusterstats", string(statsstr))
		_ = error
		api.ex.ClusterStats = stats
		api.ex.ClusterStatsString = string(data)

	} else {
		http.NotFound(w, req)
		IotHTTP404.Inc()
	}
}

// MakeHTTPExecutive sets up a http server for serving api1 and api2
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

func (cc *tcpContact) Close(err error) {

	if cc.netDotTCPConn != nil {
		fmt.Println("close tcp ", cc.netDotTCPConn.RemoteAddr())
		cc.netDotTCPConn.Close()
		cc.netDotTCPConn = nil
	}

	// hadConfig := cc.GetConfig() != nil
	// if hadConfig {
	// 	dis := packets.Disconnect{}
	// 	if err != nil {
	// 		dis.SetOption("error", []byte(err.Error()))
	// 	}
	// 	cc.WriteDownstream(&dis) // can't write to closed socket
	// }
	ss := &cc.ContactStruct
	ss.Close(err) // close my parent
}

func (cc *tcpContact) WriteDownstream(packet packets.Interface) error {
	//fmt.Println("received from above", packet, reflect.TypeOf(packet))
	if !cc.GetClosed() && !cc.GetConfig().GetLookup().isGuru {
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

func listenForPacketsConnect(ex *Executive, name string) {
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
		go handleConnection(tmpconn.(*net.TCPConn), ex)
	}
}

func handleConnection(tcpConn *net.TCPConn, ex *Executive) {

	// FIXME: all the *LogThing expressions in package need to be re-written for prom
	//srvrLogThing.Collect("Conn Accept")
	TCPServerConnAccept.Inc() // <-- like this

	fmt.Println("KF native contact add")

	cc := localMakeTCPContact(ex.Config, tcpConn)
	defer func() {
		fmt.Println("handleConnection exit close")
		cc.Close(nil)
	}()

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
			fmt.Println("packets KnotFree read err", err)
			TCPServerPacketReadError.Inc()
			cc.Close(err)
			return
		}
		// str := p.String() // giant performance problem with p.String()
		// if !strings.ContainsAny(str, "billing_stats_return_address_subscribe") {
		// 	fmt.Println("tcp got packet", p.String(), cc)
		// }

		err = PushPacketUpFromBottom(cc, p)
		if err != nil {
			//connLogThing.Collect("se err " + err.Error())
			fmt.Println("iot.push err", err)
			TCPServerIotPushEror.Inc()
			cc.Close(err)
			return
		}
	}
}

// SocketSetup sets common options
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
func GetServerStats(addr string) (*ExecutiveStats, error) {

	stats := &ExecutiveStats{}

	if len(addr) < 4 {
		return stats, errors.New("missing stats address")
	}
	if strings.HasPrefix(addr, ":") {
		return stats, errors.New("only port")
	}

	client := http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get("http://" + addr + "/api2/getstats")

	if err == nil && resp.StatusCode == 200 {

		bytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return stats, err
		}
		err = json.Unmarshal(bytes, &stats)
		if err != nil {
			return stats, err
		}
	} else {
		fmt.Println("GetServerStats failed ", addr, err)
	}
	return stats, err
}

// UpstreamNamesArg just has the one job
type UpstreamNamesArg struct {
	Names     []string
	Addresses []string
}

// PostUpstreamNames does SetUpstreamNames the hard way
// we are not going over the internet. Inside a ns should ba well under 1000 ms.
func PostUpstreamNames(guruList []string, addressList []string, addr string) error {

	arg := &UpstreamNamesArg{}
	arg.Names = guruList
	arg.Addresses = addressList

	if len(guruList) != len(addressList) {
		return errors.New("PostUpstreamNames len(guruList) != len(addressList)")
	}

	jbytes, err := json.Marshal(arg)
	if err != nil {
		fmt.Println("unreachable ?? bb")
		return errors.New("upstreamNamesArg marshal fail")
	}

	client := http.Client{Timeout: 1 * time.Second}
	resp, err := client.Post("http://"+addr+"/api2/set", "application/json", bytes.NewReader(jbytes))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return errors.New("upstreamNamesArg not 200")
	}
	return nil
}

// PostClusterStats sends some stats to
func PostClusterStats(stats *ClusterStats, addr string) error {

	jbytes, err := json.Marshal(stats)
	if err != nil {
		fmt.Println("unreachable ? PostClusterStats marshal fail")
		return errors.New("PostClusterStats marshal fail")
	}

	addstr := "http://" + addr + "/api2/clusterstats"
	client := http.Client{Timeout: 1 * time.Second}
	resp, err := client.Post(addstr, "application/json", bytes.NewReader(jbytes))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return errors.New("PostClusterStats not 200")
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
