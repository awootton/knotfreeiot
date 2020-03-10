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

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"time"

	"github.com/awootton/knotfreeiot/iot"
	"github.com/awootton/knotfreeiot/tokens"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Hint: add "127.0.0.1 knotfreeserver" to /etc/hosts
func main() {

	tokens.LoadPublicKeys()

	fmt.Println("Hello knotfreeserver")

	client := flag.Int("client", 0, "start a client test with an int of clients.")
	server := flag.Bool("server", false, "start a server.")
	//isGuru := flag.Bool("isguru", false, "")

	// means that the limits are very small
	nano := flag.Bool("nano", false, "")

	token := flag.String("token", "", " an access token for our superiors")

	flag.Parse()

	if *token == "" {
		*token = tokens.SampleSmallToken
	}

	var mainLimits = &iot.ExecutiveLimits{}
	mainLimits.Connections = 10 * 1000
	mainLimits.BytesPerSec = 10 * 1000
	mainLimits.Subscriptions = 1000 * 1000

	limits := mainLimits

	name := os.Getenv("POD_NAME")
	if len(name) == 0 {
		name = "apodnamefixme"
	}

	if *nano == true {
		limits = &iot.TestLimits
		fmt.Println("nano limits")
	}

	if *server {

		ce := iot.MakeTCPMain(name, limits, *token)
		startPublicServer(ce)
		for {
			time.Sleep(1000 * time.Second)
		}
	} else if *client > 0 {

		// FIXME: put the stress tests back in here.

	} else {
		ce := iot.MakeTCPMain(name, limits, *token)
		startPublicServer(ce)
		for {
			time.Sleep(1000 * time.Second)
		}
	}

}

type apiHandler struct {
	ce *iot.ClusterExecutive
}

func (api apiHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {

	fmt.Println("ServeHTTP", req.RequestURI)

	if req.RequestURI == "/api1/getstats" {

		stats := api.ce.Aides[0].GetExecutiveStats()
		bytes, err := json.Marshal(stats)
		if err != nil {
			fmt.Println("GetExecutiveStats marshal", err)
		}
		w.Write(bytes)

	} else {
		http.NotFound(w, req)
		//fmt.Fprintf(w, "expected known path "+req.RequestURI)
		HttpServe404.Inc()
	}
}

func startPublicServer(ce *iot.ClusterExecutive) {
	// an http server and reverse proxy.

	go startPublicServer3000(ce)
	go startPublicServer9090(ce)
	go startPublicServer8000(ce)

	go startPublicServer9102(ce)

	mux := http.NewServeMux()
	mux.Handle("/api1/", apiHandler{ce})

	fs := http.FileServer(http.Dir("./docs/_site"))
	mux.Handle("/", fs)

	s := &http.Server{
		Addr:           ":8085",
		Handler:        mux,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 13,
	}
	go func(s *http.Server) {
		fmt.Println("http service " + s.Addr)
		err := s.ListenAndServe()
		_ = err
		fmt.Println("ListenAndServe 8085 returned !!!!!  arrrrg", err)
	}(s)

}

func startPublicServer9102(ce *iot.ClusterExecutive) {
	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(":9102", nil)
}

func startPublicServer3000(ce *iot.ClusterExecutive) {
	// an http server and reverse proxy.

	mux := http.NewServeMux()

	if true {
		origin, _ := url.Parse("http://grafana.monitoring:3000/")
		director := func(req *http.Request) {
			req.Header.Add("X-Forwarded-Host", req.Host)
			req.Header.Add("X-Origin-Host", origin.Host)
			req.URL.Scheme = "http"
			req.URL.Host = origin.Host
			//fmt.Println("fwd graf:", req.URL.Host, req.URL.Port(), req.URL.Path)
			ForwardsCount3000.Inc()
		}
		proxy := &httputil.ReverseProxy{Director: director}
		mux.Handle("/", proxy)
	}

	s := &http.Server{
		Addr:           ":3000",
		Handler:        mux,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 13,
	}
	go func(s *http.Server) {
		fmt.Println("http service " + s.Addr)
		err := s.ListenAndServe()
		_ = err
		fmt.Println("ListenAndServe 3000 returned !!!!!  arrrrg", err)
	}(s)

}

func startPublicServer9090(ce *iot.ClusterExecutive) {
	// an http server and reverse proxy.

	mux := http.NewServeMux()

	if true {
		origin, _ := url.Parse("http://prometheus-operated.monitoring:9090/")
		director := func(req *http.Request) {
			req.Header.Add("X-Forwarded-Host", req.Host)
			req.Header.Add("X-Origin-Host", origin.Host)
			req.URL.Scheme = "http"
			req.URL.Host = origin.Host
			//fmt.Println("fwd prom:", req.URL.Host, req.URL.Port(), req.URL.Path)
			ForwardsCount9090.Inc()
		}
		proxy := &httputil.ReverseProxy{Director: director}
		mux.Handle("/", proxy)
	}

	s := &http.Server{
		Addr:           ":9090",
		Handler:        mux,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 13,
	}
	go func(s *http.Server) {
		fmt.Println("http service " + s.Addr)
		err := s.ListenAndServe()
		_ = err
		fmt.Println("ListenAndServe 9090 returned !!!!!  arrrrg", err)
	}(s)

}

func xxxstartPublicServer8000(ce *iot.ClusterExecutive) {
	// an http server and reverse proxy.

	mux := http.NewServeMux()

	if true {
		origin, _ := url.Parse("http://libra.libra:8000/")
		director := func(req *http.Request) {
			req.Header.Add("X-Forwarded-Host", req.Host)
			req.Header.Add("X-Origin-Host", origin.Host)
			req.URL.Scheme = "http"
			req.URL.Host = origin.Host
			//fmt.Println("fwd prom:", req.URL.Host, req.URL.Port(), req.URL.Path)
			ForwardsCount8000.Inc()
		}
		proxy := &httputil.ReverseProxy{Director: director}
		mux.Handle("/", proxy)
	}

	s := &http.Server{
		Addr:           ":8000",
		Handler:        mux,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 13,
	}
	go func(s *http.Server) {
		fmt.Println("http service " + s.Addr)
		err := s.ListenAndServe()
		_ = err
		fmt.Println("ListenAndServe 8000 returned !!!!!  arrrrg", err)
	}(s)

}

func startPublicServer8000(ce *iot.ClusterExecutive) {
	fmt.Println("tcp service 8000")
	ln, err := net.Listen("tcp", ":8000")
	if err != nil {
		fmt.Println("tcp 8000 listen fail")
		//panic(err)
		return
	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			//panic(err)
			ForwardsAcceptl8000.Inc()
			break
		}
		go handleRequest(conn)
	}
}

func handleRequest(conn net.Conn) {
	fmt.Println("new client")
	proxy, err := net.Dial("tcp", "libra.libra:8000")
	if err != nil {
		//panic(err)
		//fmt.Println("startPublicServer8000 FAIL to dial", err)
		ForwardsDialFail8000.Inc()
		return
	}
	//fmt.Println("proxy 8000 connected")
	ForwardsConnectedl8000.Inc()
	go copyIO(conn, proxy)
	go copyIO(proxy, conn)
}

func copyIO(src, dest net.Conn) {
	defer src.Close()
	defer dest.Close()
	io.Copy(src, dest)
}
