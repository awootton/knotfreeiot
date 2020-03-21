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
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/awootton/knotfreeiot/iot"
	"github.com/awootton/knotfreeiot/tokens"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/crypto/nacl/box"
)

// Hint: add "127.0.0.1 knotfreeserver" to /etc/hosts
func main() {

	tokens.LoadPublicKeys()

	tokens.LoadPrivateKeys("~/atw/privateKeys4.txt")

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
	mainLimits.Input = 10 * 1000
	mainLimits.Output = 10 * 1000
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

	if req.RequestURI == "/api1/getallstats" {

		stats := api.ce.Aides[0].ClusterStatsString

		w.Write([]byte(stats))

	} else if req.RequestURI == "/api1/getstats" {

		stats := api.ce.Aides[0].GetExecutiveStats()
		bytes, err := json.Marshal(stats)
		if err != nil {
			fmt.Println("GetExecutiveStats marshal", err)
		}
		w.Write(bytes)

	} else if req.RequestURI == "/api1/getToken" {

		// what is IP or id of sender?
		fmt.Println("token req RemoteAddr", req.RemoteAddr) // FIXME: use db

		var buff1024 [1024]byte
		n, err := req.Body.Read(buff1024[:])
		buf := buff1024[:n]
		//fmt.Println("read body", string(buf), n)

		tokenRequest := &tokens.TokenRequest{}
		err = json.Unmarshal(buf, tokenRequest)
		if err != nil {
			badTokenRequests.Inc()
			fmt.Println("TokenRequest err", err.Error())
			http.Error(w, err.Error(), 500)
		} else {
			// todo: calc cost of this token and have limit.
			// move this phat routine somewhere else TODO:

			clientPublicKey := tokenRequest.Pkey
			if len(clientPublicKey) != 64 {
				badTokenRequests.Inc()
				http.Error(w, "bad client key", 500)
			}

			signingKey := tokens.GetPrivateKey("/9sh")

			payload := tokenRequest.Payload
			payload.Issuer = "/9sh"
			payload.JWTID = getRandomB64String()
			nonce := payload.JWTID

			exp := payload.ExpirationTime
			if exp > uint32(time.Now().Unix()+60*60*24*365) {
				// more than a year in the future not allowed now.
				exp = uint32(time.Now().Unix() + 60*60*24*365)
				fmt.Println("had long token ", string(payload.JWTID)) // TODO: store in db
			}

			tokenString, err := tokens.MakeToken(payload, []byte(signingKey))
			if err != nil {
				badTokenRequests.Inc()
				http.Error(w, err.Error(), 500)
				return
			}

			when := time.Unix(int64(exp), 0)
			year, month, day := when.Date()

			payload.JWTID = ""
			payload.ExpirationTime = 0

			comments := make([]interface{}, 3)
			tmp := fmt.Sprintf(" expires: %v-%v-%v", year, int(month), day)
			comments[0] = tokenRequest.Comment + tmp
			comments[1] = payload
			comments[2] = string(tokenString)
			returnval, err := json.Marshal(comments)
			returnval = []byte(strings.ReplaceAll(string(returnval), `"`, ``))
			returnval = []byte(strings.ReplaceAll(string(returnval), ` `, `_`))
			fmt.Println("sending token package ", string(returnval)) // FIXME: use db

			// box it up
			boxout := make([]byte, len(returnval)+box.Overhead)
			boxout = boxout[:0]
			var jwtid [24]byte
			copy(jwtid[:], []byte(nonce))

			var clipub [32]byte
			n, err := hex.Decode(clipub[:], []byte(clientPublicKey))
			if n != 32 {
				badTokenRequests.Inc()
				http.Error(w, "bad size2", 500)
				return
			}

			sealed := box.Seal(boxout, returnval, &jwtid, &clipub, api.ce.PrivateKeyTemp)

			reply := tokens.TokenReply{}
			reply.Nonce = nonce
			reply.Pkey = hex.EncodeToString(api.ce.PublicKeyTemp[:])
			reply.Payload = hex.EncodeToString(sealed)
			bytes, err := json.Marshal(reply)
			if err != nil {
				badTokenRequests.Inc()
				http.Error(w, err.Error(), 500)
				return
			}
			time.Sleep(8 * time.Second)
			w.Write(bytes)
		}

	} else {
		http.NotFound(w, req)
		//fmt.Fprintf(w, "expected known path "+req.RequestURI)
		httpServe404.Inc()
	}
}

func getRandomB64String() string {
	var tmp [18]byte
	rand.Read(tmp[:])
	return base64.RawStdEncoding.EncodeToString(tmp[:])
}

func startPublicServer(ce *iot.ClusterExecutive) {
	// an http server and reverse proxy.

	go startPublicServer3000(ce)
	go startPublicServer9090(ce)
	go startPublicServer8000(ce)

	go startPublicServer9102(ce)

	go func() {
		for {
			ce.Aides[0].Heartbeat(uint32(time.Now().Unix()))
			time.Sleep(10 * time.Second)
		}
	}()

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
	fmt.Println("http service 9102")
	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(":9102", nil)
	fmt.Println("http service 9102 FAIL")
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
			forwardsCount3000.Inc()
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
			forwardsCount9090.Inc()
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
			forwardsCount8000.Inc()
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
			forwardsAcceptl8000.Inc()
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
		forwardsDialFail8000.Inc()
		return
	}
	//fmt.Println("proxy 8000 connected")
	forwardsConnectedl8000.Inc()
	go copyIO(conn, proxy)
	go copyIO(proxy, conn)
}

func copyIO(src, dest net.Conn) {
	defer src.Close()
	defer dest.Close()
	io.Copy(src, dest)
}
