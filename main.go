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
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/awootton/knotfreeiot/iot"
	"github.com/awootton/knotfreeiot/packets"
	"github.com/awootton/knotfreeiot/tokens"
	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/crypto/nacl/box"
)

// Hint: add "127.0.0.1 knotfreeserver" to /etc/hosts
func main() {

	tokens.LoadPublicKeys()

	tokens.LoadPrivateKeys("~/atw/privateKeys4.txt")

	fmt.Println("Hello knotfreeserver")

	isGuru := flag.Bool("isguru", false, "")

	// means that the limits are very small - for testing
	nano := flag.Bool("nano", false, "")

	token := flag.String("token", "", " an access token for our guru, if any")

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
		name = "DefaultPodName"
	}

	if *nano == true {
		limits = &iot.TestLimits
		fmt.Println("nano limits")
	}

	ce := iot.MakeTCPMain(name, limits, *token, *isGuru)
	startPublicServer(ce)
	for {
		time.Sleep(10000 * time.Second)
	}

}

type apiHandler struct {
	ce *iot.ClusterExecutive
}

func (api apiHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {

	fmt.Println("ServeHTTP", req.RequestURI, req.Host)

	ctx := req.Context()

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

		remoteAddr := req.RemoteAddr

		// what is IP or id of sender?
		fmt.Println("token req RemoteAddr", remoteAddr) // F I X M E: use db see below

		var buff1024 [1024]byte
		n, err := req.Body.Read(buff1024[:])
		buf := buff1024[:n]
		//fmt.Println("read body", string(buf), n)

		tokenRequest := &tokens.TokenRequest{}
		err = json.Unmarshal(buf, tokenRequest)
		if err != nil {
			iot.BadTokenRequests.Inc()
			fmt.Println("TokenRequest err", err.Error())
			http.Error(w, err.Error(), 500)
		} else {
			// todo: calc cost of this token and have limit.
			// move this phat routine somewhere else TODO:

			clientPublicKey := tokenRequest.Pkey
			if len(clientPublicKey) != 64 {
				iot.BadTokenRequests.Inc()
				http.Error(w, "bad client key", 500)
			}

			signingKey := tokens.GetPrivateKey("_9sh")

			payload := tokenRequest.Payload
			payload.Issuer = "_9sh"
			payload.JWTID = tokens.GetRandomB64String()
			nonce := payload.JWTID

			exp := payload.ExpirationTime
			if exp > uint32(time.Now().Unix()+60*60*24*365) {
				// more than a year in the future not allowed now.
				exp = uint32(time.Now().Unix() + 60*60*24*365)
				fmt.Println("had long token ", string(payload.JWTID)) // TODO: store in db
			}

			cost := tokens.CalcTokenPrice(tokenRequest.Payload)
			fmt.Println("token cost is " + fmt.Sprintf("%f", cost))

			if cost > 0.001 {
				http.Error(w, "token too expensive at "+fmt.Sprintf("%f", cost), 500)
			}

			tokenString, err := tokens.MakeToken(payload, []byte(signingKey))
			if err != nil {
				iot.BadTokenRequests.Inc()
				http.Error(w, err.Error(), 500)
				return
			}

			when := time.Unix(int64(exp), 0)
			year, month, day := when.Date()

			// payload.JWTID = ""
			//payload.ExpirationTime = 0

			comments := make([]interface{}, 3)
			tmp := fmt.Sprintf(" expires: %v-%v-%v", year, int(month), day)
			comments[0] = tokenRequest.Comment + tmp
			comments[1] = payload
			comments[2] = string(tokenString)
			returnval, err := json.Marshal(comments)
			returnval = []byte(strings.ReplaceAll(string(returnval), `"`, ``))
			returnval = []byte(strings.ReplaceAll(string(returnval), ` `, `_`))
			//fmt.Println("sending token package ", string(returnval)) // FIXME: use db

			err = tokens.LogNewToken(ctx, payload, remoteAddr)
			if err != nil {
				iot.BadTokenRequests.Inc()
				http.Error(w, err.Error(), 500)
				return
			}

			// box it up
			boxout := make([]byte, len(returnval)+box.Overhead)
			boxout = boxout[:0]
			var jwtid [24]byte
			copy(jwtid[:], []byte(nonce))

			var clipub [32]byte
			n, err := hex.Decode(clipub[:], []byte(clientPublicKey))
			if n != 32 {
				iot.BadTokenRequests.Inc()
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
				iot.BadTokenRequests.Inc()
				http.Error(w, err.Error(), 500)
				return
			}
			time.Sleep(8 * time.Second)
			w.Write(bytes)
		}
	} else if req.RequestURI == "/api1/getPublicKey" {

		sss := base64.RawURLEncoding.EncodeToString([]byte(tokens.FindPublicKey("yRst")))

		w.Write([]byte(sss))

	} else {
		http.NotFound(w, req)
		//fmt.Fprintf(w, "expected known path "+req.RequestURI)
		iot.HTTPServe404.Inc()
	}
}

func GetRandomB64String() string {
	var tmp [18]byte
	rand.Read(tmp[:])
	return base64.RawURLEncoding.EncodeToString(tmp[:])
}

type SuperMux struct {
	ce         *iot.ClusterExecutive
	super, sub *http.ServeMux
}

func parsePayload(httpBytes string) (string, map[string]string, string) {
	headerMap := make(map[string]string)
	pos := strings.Index(httpBytes, "\r\n\r\n")
	if pos <= 0 {
		fmt.Println("isWhat? no header end!!! this is bad ", httpBytes)
		return "", headerMap, ""
	}
	payload := httpBytes[pos+4:]
	headers := httpBytes[0:pos]
	headerParts := strings.Split(headers, "\r\n")

	firstLine := headerParts[0]
	headerParts = headerParts[1:]
	for _, head := range headerParts {
		pos = strings.Index(head, ":")
		if pos > 0 && len(head) > 3 {
			key := strings.Trim(head[0:pos], " ")
			val := strings.Trim(head[pos+1:], " ")
			headerMap[key] = val
		} else {
			fmt.Println("weird header found " + head)
		}
	}
	return firstLine, headerMap, payload
}

func (superMux *SuperMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	domainParts := strings.Split(r.Host, ".")

	// supermux.sub.Handle("/api1/", apiHandler{ce})
	// supermux.sub.Handle("/mqtt", wsAPIHandler{ce})
	// fs := http.FileServer(http.Dir("./docs/_site"))
	// supermux.sub.Handle("/", fs)
	isGetToken := len(r.RequestURI) >= 9 && r.RequestURI[:9] == "/api1/get"
	//isGetToken = isGetToken || r.RequestURI == "/mqtt"

	if len(domainParts) > 1 && isGetToken == false {
		// we have a subdomain
		subDomain := domainParts[0]
		fmt.Println("serving subdomain ", subDomain, "  with "+r.URL.String())

		// TODO: move this all to a method httpRequestContact (and todo: make that)

		// we need the whole original http req as a []byte

		clen := r.ContentLength
		if clen > 63*1024 {
			fmt.Println("http packet too long ")
			http.Error(w, "http packet too long ", 500)
			return
		}
		theBody := make([]byte, clen)
		if clen > 0 {
			n, err := r.Body.Read(theBody)
			if err != nil || (n != int(clen)) {
				http.Error(w, "http content read fail ", 500)
				return
			}
		}
		//fmt.Println("http header ", r.Header) // it's a map with Cookie
		// r.RequtURI is "/"
		// r.URL is "/"
		firstLine := r.Method + " " + r.URL.String() + " " + r.Proto + "\n"
		//fmt.Println("first line", firstLine)
		buf := new(bytes.Buffer)
		buf.WriteString(firstLine)
		for key, val := range r.Header {
			if key == "Cookie" {
				continue
			}
			for i := 0; i < len(val); i++ {
				tmp := key + ": " + val[i] + "\n"
				buf.WriteString(tmp)
			}
		}
		buf.WriteString("Accept-Encoding: identity\n") // no gzip
		buf.WriteString("\n")

		fmt.Println("about to write body to publish packet ", buf.String())

		n, err := buf.Write(theBody)
		if err != nil || (n != len(theBody)) {
			http.Error(w, "http theBody write ", 500)
		}
		//fmt.Println("http is ", buf.String())

		// we need to make a contact
		// make a reply address
		// serialize the request
		// publish it.
		// wait for the response and put that into the w http.ResponseWriter
		// copy over the response headers
		// unsub the reply address
		// close the contact.

		contact := &iot.ContactStruct{}
		iot.AddContactStruct(contact, contact, superMux.ce.Aides[0].Config)
		//contact.SetExpires(contact.contactExpires + 60*60*24*365*10) // in 10 years

		connect := packets.Connect{}
		connect.SetOption("token", []byte(tokens.SampleSmallToken)) //FIXME: What token??
		err = iot.PushPacketUpFromBottom(contact, &connect)
		if err != nil {
			fmt.Println("connect problems subdomain dial conn ", err)
			http.Error(w, err.Error(), 500)
			return
		}
		// define a reader and a writer
		//gotDataChan := make(chan byte)

		gotDataChan := new(iot.ByteChan)
		gotDataChan.TheChan = make(chan []byte, 1)
		contact.SetWriter(gotDataChan)
		// we don't need for the contact to read. We'll push directly

		// subscribe
		myRandomAddress := GetRandomB64String()
		// just for test: myRandomAddress = "atwdummytest9999999"
		subs := packets.Subscribe{}
		subs.Address.FromString(myRandomAddress)
		subs.Address.EnsureAddressIsBinary()
		//fmt.Println(" our return addr will be ", subs.Address.String())
		err = iot.PushPacketUpFromBottom(contact, &subs)

		pub := packets.Send{}
		pub.Address.FromString(subDomain) // !!!!!
		pub.Source = subs.Address
		//fmt.Println(" our send addr is ", pub.Address.String())
		pub.Address.EnsureAddressIsBinary()
		//fmt.Println(" our send addr is ", pub.Address.String())
		//fmt.Println(" our return addr is ", pub.Source.String())
		//pub.Payload = []byte("GET " + r.URL.String() + " HTTP/1.1\n\n")
		pub.Payload = buf.Bytes()

		//fmt.Println("payload ius ", string(pub.Payload))

		err = iot.PushPacketUpFromBottom(contact, &pub)

		//go
		//func()
		{
			running := true
			for running { // data := range gotDataChan {
				select {
				case somedata := <-gotDataChan.TheChan:
					if somedata == nil {
						running = false
					}
					// var cmd packets.Interface
					cmd, err := packets.ReadPacket(bytes.NewReader(somedata))
					if err != nil {
						fmt.Println("packet parse problem with gotDataChan  ", err)
						http.Error(w, err.Error(), 500)
						return
					}
					switch cmd.(type) {

					case *packets.Send:
						snd := cmd.(*packets.Send)
						fmt.Println("got a reply payload packet:", string(snd.Payload))
						firstLine, headerParts, theHtmlPart := parsePayload(string(snd.Payload))
						_ = firstLine
						for k, v := range headerParts { // copy the headers over
							w.Header().Add(k, v)
						}
						fmt.Println("writing theHtmlPart:", theHtmlPart)
						w.Write([]byte(theHtmlPart))
					default:
						// no match. do nothing. apnic?
						fmt.Println("got weird packet instead of publish ", reflect.TypeOf(cmd))
						w.Write([]byte("error got weird packet"))
					}

					running = false
				case <-time.After(16 * time.Second):
					errMsg := "timed out waiting for html reply"
					fmt.Println(errMsg)
					http.Error(w, errMsg, 500)
					running = false
				}
			}

			fmt.Println("closing html write ")
			// un sub
			// close the contact
			unsub := packets.Unsubscribe{}
			unsub.Address.FromString(myRandomAddress)
			err = iot.PushPacketUpFromBottom(contact, &unsub)
			_ = err
			contact.Close(errors.New("normal close"))
		}

		//aContact := superMux.ce.Aides[0]
		//pub := &iot.PushPacketUpFromBottom()

	} else {
		superMux.sub.ServeHTTP(w, r)
	}

}

func startPublicServer(ce *iot.ClusterExecutive) {
	// an http server and reverse proxy.

	go startPublicServer3100(ce)

	// for prometheus webpage
	go startPublicServer9090(ce) // eliminate?

	// go startPublicServer8000(ce) // was for libra

	go startPublicServer9102(ce) // promhttp.Handler for getting metrics

	go func() {
		for {
			ce.Aides[0].Heartbeat(uint32(time.Now().Unix()))
			time.Sleep(10 * time.Second)
		}
	}()

	supermux := &SuperMux{}
	supermux.ce = ce

	supermux.sub = http.NewServeMux()
	supermux.sub.Handle("/api1/", apiHandler{ce})
	supermux.sub.Handle("/mqtt", wsAPIHandler{ce})
	fs := http.FileServer(http.Dir("./docs/_site"))
	supermux.sub.Handle("/", fs)

	s := &http.Server{
		Addr:           ":8085",
		Handler:        supermux,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 13,
	}
	go func(s *http.Server) {
		fmt.Println("http service for ws " + s.Addr)
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

func startPublicServer3100(ce *iot.ClusterExecutive) {
	// an http server and reverse proxy.

	mux := http.NewServeMux()

	if true {
		origin, _ := url.Parse("http://grafana.monitoring:3100/")
		director := func(req *http.Request) {
			req.Header.Add("X-Forwarded-Host", req.Host)
			req.Header.Add("X-Origin-Host", origin.Host)
			req.URL.Scheme = "http"
			req.URL.Host = origin.Host
			//fmt.Println("fwd graf:", req.URL.Host, req.URL.Port(), req.URL.Path)
			iot.ForwardsCount3100.Inc()
		}
		proxy := &httputil.ReverseProxy{Director: director}
		mux.Handle("/", proxy)
	}

	s := &http.Server{
		Addr:           ":3100",
		Handler:        mux,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 13,
	}
	go func(s *http.Server) {
		fmt.Println("http service " + s.Addr)
		err := s.ListenAndServe()
		_ = err
		fmt.Println("ListenAndServe 3100 returned !!!!!  arrrrg", err)
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
			iot.ForwardsCount9090.Inc()
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

// func startPublicServer8000(ce *iot.ClusterExecutive) {
// 	fmt.Println("tcp service 8000")
// 	ln, err := net.Listen("tcp", ":8000")
// 	if err != nil {
// 		fmt.Println("tcp 8000 listen fail")
// 		//panic(err)
// 		return
// 	}
// 	for {
// 		conn, err := ln.Accept()
// 		if err != nil {
// 			//panic(err)
// 			iot.ForwardsAcceptl8000.Inc()
// 			break
// 		}
// 		go handleRequest(conn)
// 	}
// }

// func handleRequest(conn net.Conn) {
// 	fmt.Println("new client")
// 	proxy, err := net.Dial("tcp", "libra.libra:8000")
// 	if err != nil {
// 		//panic(err)
// 		//fmt.Println("startPublicServer8000 FAIL to dial", err)
// 		iot.ForwardsDialFail8000.Inc()
// 		return
// 	}
// 	//fmt.Println("proxy 8000 connected")
// 	iot.ForwardsConnectedl8000.Inc()
// 	go copyIO(conn, proxy)
// 	go copyIO(proxy, conn)
// }

func copyIO(src, dest net.Conn) {
	defer src.Close()
	defer dest.Close()
	io.Copy(src, dest)
}

var upgrader = websocket.Upgrader{}

type wsAPIHandler struct {
	ce *iot.ClusterExecutive
}

func (api wsAPIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	fmt.Println("ws ServeHTTP", r.RequestURI)

	allowAll := func(r *http.Request) bool {
		return true
	}
	upgrader.WriteBufferSize = 4096
	upgrader.ReadBufferSize = 4096
	upgrader.CheckOrigin = allowAll
	upgrader.Subprotocols = []string{"mqtt", "mqttv5", "mqttv3.1"}

	wsConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}

	iot.WebSocketLoop(wsConn, api.ce.Aides[0].Config)
	// for {
	// 	mt, message, err := wsConn.ReadMessage()
	// 	if err != nil {
	// 		log.Println("read:", err)
	// 		break
	// 	}

	// 	log.Printf("recv: %s", message)
	// 	err = wsConn.WriteMessage(mt, message)
	// 	if err != nil {
	// 		log.Println("write:", err)
	// 		break
	// 	}
	// }
}
