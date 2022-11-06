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

package main

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
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
	"strconv"
	"strings"
	"time"

	"github.com/awootton/knotfreeiot/mainhelpers"

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

	// no need to keep doing this mainhelpers.TrySomeS3Stuff()

	h := sha256.New()
	h.Write([]byte("AnonymousAnonymous"))
	hashBytes := h.Sum(nil)
	fmt.Println("Hello. sha256 of AnonymousAnonymous is " + base64.RawURLEncoding.EncodeToString(hashBytes))

	var htmp iot.HashType
	hptr := &htmp
	hptr.HashBytes([]byte("alice_vociferous_mcgrath"))
	var tmpbuf [24]byte
	hptr.GetBytes(tmpbuf[:])
	fmt.Println("Hello. fyi, standard hash of alice_vociferous_mcgrath is " + base64.RawURLEncoding.EncodeToString(tmpbuf[:]))

	isGuru := flag.Bool("isguru", false, "")

	// means that the limits are very small - for testing
	nano := flag.Bool("nano", false, "")

	token := flag.String("token", "", " an access token for our guru, if any")

	flag.Parse()

	if *token == "" {
		*token = tokens.GetImpromptuGiantToken()
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

type ApiHandler struct {
	ce *iot.ClusterExecutive

	staticStuffHandler webHandler
}

func (api ApiHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {

	fmt.Println("ApiHandler ServeHTTP", req.RequestURI, req.Host)

	if req.RequestURI == "/api1/paypal-transaction-complete" {

		w.Header().Set("Access-Control-Allow-Origin", "*")

		var buff1024 [1024]byte
		n, err := req.Body.Read(buff1024[:])
		buf := buff1024[:n]
		fmt.Println("paypal-transaction-complete request read body", string(buf), n, err)
		_ = buf
		// there's a facilitatorAccessToken  ??

		// FIXME: check with paypal that this actually happened.

		//make a 32x token
		tok, payload := mainhelpers.Make32xLargeToken()

		ctx := req.Context()
		err = tokens.LogNewToken(ctx, &payload, string(buf))
		if err != nil {
			fmt.Println("ERROR logging token", err)
		}
		w.Write([]byte(tok))

	} else if req.RequestURI == "/api1/getallstats" {

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

		w.Header().Set("Access-Control-Allow-Origin", "*")
		api.ServeMakeToken(w, req)

	} else if req.RequestURI == "/api1/getPublicKey" {

		//sss := base64.RawURLEncoding.EncodeToString([]byte(tokens.FindPublicKey("yRst")))

		sss := base64.RawURLEncoding.EncodeToString(api.ce.PublicKeyTemp[:])

		fmt.Println("serve /api1/getPublicKey ", sss)

		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Write([]byte(sss))

	} else if req.RequestURI == "/api1/getGiantPassword" {

		sss := tokens.MakeRandomPhrase(14)

		w.Write([]byte(sss))

	} else if req.RequestURI == "/help" {

		sss := "/api1/getallstats\n"
		sss += "/api1/getstats\n"
		sss += "/api1/getToken\n"
		sss += "/api1/getPublicKey\n"
		sss += "/api1/getGiantPassword\n"

		w.Write([]byte(sss))

	} else {
		// http.NotFound(w, req)
		// fmt.Fprintf(w, "expected known path "+req.RequestURI)
		// iot.HTTPServe404.Inc()
		api.staticStuffHandler.ServeHTTP(w, req)
	}
}

func GetRandomB64String() string {
	var tmp [18]byte
	rand.Read(tmp[:])
	return base64.RawURLEncoding.EncodeToString(tmp[:])
}

type SuperMux struct {
	ce *iot.ClusterExecutive
	//super,
	sub *http.ServeMux
}

type pinfo struct {
	// these are the reply buffers
	buff []byte
}
type RequestReplyStruct struct {
	originalRequest []byte
	firstLine       string
	replyParts      []pinfo
}

var servedMap = map[string]*RequestReplyStruct{}

func (superMux *SuperMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	//fmt.Println("giant token ", tokens.GetImpromptuGiantToken())

	domainParts := strings.Split(r.Host, ".")
	// eg [dummy localhost:8085]
	// eg [knotfree net]
	// eg [subdomain knotfree net]
	// eg [subdomain knotfreeiot net]
	// eg [subdomain go2here io]
	// fmt.Println("serving domainParts ", domainParts)

	isGetToken := len(r.RequestURI) >= 9 && r.RequestURI[:9] == "/api1/get"
	isGetToken = isGetToken || r.RequestURI == "/mqtt"

	if len(domainParts) > 2 && !isGetToken && domainParts[0] != "www" {
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
		// write the header to a buffer
		firstLine := r.Method + " " + r.URL.String() + " " + r.Proto + "\n"
		fmt.Println("first line", firstLine)
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
		buf.WriteString("\n")
		// write the body to a buffer
		n, err := buf.Write(theBody)
		if err != nil || (n != len(theBody)) {
			http.Error(w, "http theBody write ", 500)
		}
		// now the whole original request packet is in buf

		isCachable := strings.Contains(firstLine, "/static/") || strings.Contains(firstLine, "/images/")
		haveAlready, ok := servedMap[firstLine]
		if ok && isCachable {

			size := 0
			for _, databuf := range haveAlready.replyParts {
				size += len(databuf.buff)
			}

			fmt.Println("serving from cache ", firstLine, size)

			hj, ok := w.(http.Hijacker)
			if !ok {
				http.Error(w, "webserver doesn't support hijacking", http.StatusInternalServerError)
				return
			}
			conn, responseBuffer, err := hj.Hijack()
			if err != nil {
				fmt.Println("hijack error  ", err)
			}
			defer func() {
				fmt.Println("closing hijack socket with " + r.URL.String())
				conn.Close()
			}()

			for i, databuf := range haveAlready.replyParts {
				n, err := responseBuffer.Write(databuf.buff[:])
				if err != nil {
					fmt.Println("responseBuffer.Write ERROR ", firstLine[0:len(firstLine)-2])
				}
				_ = i
				_ = n
			}
			return
		}

		fmt.Println("http is request ", firstLine[0:len(firstLine)-2])

		pastWritesIndex := 0
		packetStruct := &RequestReplyStruct{}

		packetStruct.originalRequest = buf.Bytes()
		packetStruct.firstLine = firstLine[0 : len(firstLine)-2]

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
		connect.SetOption("token", []byte(tokens.GetImpromptuGiantToken()))
		err = iot.PushPacketUpFromBottom(contact, &connect)
		if err != nil {
			fmt.Println("connect problems subdomain dial conn ", err)
			http.Error(w, err.Error(), 500)
			return
		}
		// subscribe
		myRandomAddress := GetRandomB64String()
		// fmt.Println("will be using myRandomAddress ", myRandomAddress)
		// just for test: myRandomAddress = "atwdummytest9999999"
		subs := packets.Subscribe{}
		subs.Address.FromString(myRandomAddress)
		subs.Address.EnsureAddressIsBinary()
		//fmt.Println(" our return addr will be ", subs.Address.String())
		err = iot.PushPacketUpFromBottom(contact, &subs)

		// define a reader and a writer
		gotDataChan := new(iot.ByteChan)
		gotDataChan.TheChan = make(chan []byte, 1)
		contact.SetWriter(gotDataChan)
		// we don't need for the contact to read. We'll push directly

		if buf.Len() > 60*1024 {
			// stream it
			fmt.Println("ERROR fixme: implement this streaming thing")
		} else {
			// just send it all at once in one Send
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
		}

		hj, ok := w.(http.Hijacker)
		if !ok {
			http.Error(w, "webserver doesn't support hijacking", http.StatusInternalServerError)
			return
		}
		conn, responseBuffer, err := hj.Hijack()
		if err != nil {
			fmt.Println("hijack error  ", err)
		}
		defer func() {
			fmt.Println("closing hijack socket with " + r.URL.String())
			conn.Close()
		}()

		{ // The Receive-a-packet loop
			running := true
			//hadHeader := false
			theLengthWeNeed := 0
			theAmountWeGot := 0
			for running { // data := range gotDataChan {
				select {
				case somedata := <-gotDataChan.TheChan:
					if somedata == nil {
						running = false
					}
					fmt.Println("packet on gotDataChan.TheChan theLengthWeNeed= ", theLengthWeNeed)
					// var cmd packets.Interface.
					cmd, err := packets.ReadPacket(bytes.NewReader(somedata))
					if err != nil {
						fmt.Println("packet parse problem with gotDataChan  ", err)
						http.Error(w, err.Error(), 500)
						return
					}
					keys, values := cmd.GetOptionKeys()
					//fmt.Println("packet user keys ", keys, values)
					_ = keys
					_ = values

					switch cmd.(type) {
					case *packets.Send:
						snd := cmd.(*packets.Send)
						end := 32
						if end > len(snd.Payload) {
							end = len(snd.Payload)
						}
						packetCountStr, ok := snd.GetOption("of")
						if ok == true {
							fmt.Println("packet count total= ", packetCountStr)
							// we have the last packet.
							running = false
							break
						}
						packetCountStr, ok = snd.GetOption("indx")
						if !ok {
							packetCountStr = []byte("0")
						}
						if packetCountStr[0] == '[' { // some idiot wrapped it in []
							packetCountStr = packetCountStr[1:]
						}
						if packetCountStr[len(packetCountStr)-1] == ']' {
							packetCountStr = packetCountStr[0 : len(packetCountStr)-1]
						}
						packetIncomingIndex, _ := strconv.Atoi(string(packetCountStr))
						//fmt.Println("packet count is ", packetCount)
						//if packetCount != packetsReceived {
						//	fmt.Println("we seem to have lost a PACKET:", packetCount, packetsReceived)
						//} pastWritesIndex
						// pad out the buffer
						for packetIncomingIndex >= len(packetStruct.replyParts) {
							pi := &pinfo{}
							packetStruct.replyParts = append(packetStruct.replyParts, *pi)
						}
						packetStruct.replyParts[packetIncomingIndex].buff = snd.Payload

						fmt.Println("have http reply packet #", packetIncomingIndex, "for ", firstLine)
						if packetIncomingIndex == 0 {
							headerEndBytes := []byte("\r\n\r\n")
							headerPos := bytes.Index(snd.Payload, headerEndBytes)
							if headerPos <= 0 {
								fmt.Println("no header was found in first packet")
							} else {
								// parse the header
								header := snd.Payload[0:headerPos]
								clStr := "Content-Length:"
								clPos := bytes.Index(header, []byte(clStr))
								if clPos <= 0 {
									fmt.Println("no Content-Length was found in first packet")
								}
								hpart := header[clPos+len(clStr):]
								lineEndBytes := []byte("\r\n")
								endPos := bytes.Index(hpart, lineEndBytes)
								//fmt.Println("is this a number? ", hpart[0:endPos])
								cldigits := string(hpart[0:endPos])
								i, err := strconv.Atoi(strings.Trim(cldigits, " "))
								if err != nil {
									fmt.Println("ERROR finding Content-Length", hpart[0:endPos])
								}
								fmt.Println("theLengthWeNeed is ", i)
								theLengthWeNeed = i + len(header) + 4
							}
						}

						for true { // loop over packetlist stuff we can write
							if pastWritesIndex >= len(packetStruct.replyParts) {
								break // at the end
							}
							nextPi := packetStruct.replyParts[pastWritesIndex]

							// end := 32
							// if len(nextPi.buff) < end {
							// 	end = len(nextPi.buff)
							// }

							//fmt.Println("got a reply payload packet index ", pastWritesIndex, "d=", string(nextPi.buff[0:end]))
							n, err := responseBuffer.Write(nextPi.buff)
							pastWritesIndex += 1
							theAmountWeGot += len(nextPi.buff)
							if err != nil {
								fmt.Println("got a reply write err:", err)
								running = false
								break
							}
							if n != len(nextPi.buff) {
								fmt.Println("writing len wanted, needed:", len(nextPi.buff), n)
							}
							fmt.Println("So far we have got", theAmountWeGot, " of ", theLengthWeNeed, "for", packetStruct.firstLine)
							if theAmountWeGot >= theLengthWeNeed {
								fmt.Println("looks like we made it ! :")
								responseBuffer.Flush()
								running = false
								// push a close packet or something
								// close the connection -- below
							}
							//responseBuffer.Flush()
						}

					default:
						// no match. do nothing. panic?
						fmt.Println("got weird packet instead of publish ", reflect.TypeOf(cmd))
						w.Write([]byte("error got weird packet"))
					}
				// is this the only way to know that we're done??
				case <-time.After(22 * time.Second):
					errMsg := "timed out waiting for html reply " + firstLine
					fmt.Println(errMsg)
					// http.Error(w, errMsg, 500)
					running = false
				}
			}

			//fmt.Println("closing html write ")
			responseBuffer.Flush()
			// un sub
			// close the contact
			unsub := packets.Unsubscribe{}
			unsub.Address.FromString(myRandomAddress)
			err = iot.PushPacketUpFromBottom(contact, &unsub)
			_ = err
			contact.Close(errors.New("normal close"))
			if isCachable {
				size := 0
				for _, databuf := range packetStruct.replyParts {
					size += len(databuf.buff)
				}
				if size > 1500 {
					fmt.Println("adding to cache ", firstLine, size)
					servedMap[firstLine] = packetStruct
				}
			}
		}
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

	staticStuffHandler := webHandler{ce,
		http.FileServer(http.Dir("./docs"))} // FIXME: points to gotohere static assets (a react build)

	//supermux.sub.Handle("/api1/", ApiHandler{ce})
	//supermux.sub.Handle("/help", ApiHandler{ce})
	supermux.sub.Handle("/mqtt", wsAPIHandler{ce})

	supermux.sub.Handle("/", ApiHandler{ce, staticStuffHandler})
	// moved to webHandler:
	// fs := http.FileServer(http.Dir("./docs/_site"))
	// supermux.sub.Handle("/", fs)

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

	// see dns-sd -B _mqtt._tcp
	// this is NOT working.
	// I don't even know how it would ever work.
	// how would we open a port to 5353 if it's already open?
	// lsof -Pn -i4 | grep 5353
	// returns
	// Google      530 awootton  236u  IPv4 0x7453d2bfa091be27      0t0  UDP *:5353
	// so wtf?

	//Setup our service export

	// fmt.Println("starting mDns")
	// host, _ := os.Hostname()
	// info := []string{"Knotfree mqtt service"}
	// srvc := "_mqtt._tcp"
	// domain := "local."
	// hostname := ""
	// service, _ := mdns.NewMDNSService(host, srvc, domain, hostname, 1883, nil, info)

	// fmt.Println("mDns HostName", service.HostName)
	// fmt.Println("mDns Service", service.Service)
	// fmt.Println("mDns IPs", service.IPs)
	// server, _ := mdns.NewServer(&mdns.Config{Zone: service})
	// _ = server

	// defer func() {
	// 	server.Shutdown()
	// 	fmt.Println("mDns shutdown")
	// }()

	// this don't work
	//let's check what's avail
	// Make a channel for results and start listening
	// entriesCh := make(chan *mdns.ServiceEntry, 4)
	// go func() {
	// 	for entry := range entriesCh {
	// 		fmt.Printf("Got new entry: %v\n", entry)
	// 	}
	// }()

	// // Start the lookup
	// //  need params.DisableIPv6 = true
	// //mdns.Lookup("_mqtt._tcp", entriesCh)
	// //close(entriesCh)

	// params := mdns.DefaultParams("_mqtt._tcp")
	// params.Entries = entriesCh
	// params.DisableIPv6 = true
	// something := mdns.Query(params)
	// _ = something
	// fmt.Println("mdns.Query err", something)

}

func startPublicServer9102(ce *iot.ClusterExecutive) {
	fmt.Println("http metrics service 9102")
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
		fmt.Println("http grafana service " + s.Addr)
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

	// fmt.Println("ws ServeHTTP", r.RequestURI)

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
}

func ParsePayload(httpBytes string) (string, map[string]string, string) {
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

var bootTimeSec int64 = 0
var tokensServed int64 = 0

func (api ApiHandler) ServeMakeToken(w http.ResponseWriter, req *http.Request) {

	if bootTimeSec == 0 {
		bootTimeSec = time.Now().Unix()
	}

	remoteAddr := req.RemoteAddr
	parts := strings.Split(remoteAddr, ":") // eg [::1]:49326
	// lose the port
	parts = parts[0 : len(parts)-1]
	remoteAddr = strings.Join(parts, ":")
	// remoteAddr += req.Header.Get("HTTP_X_FORWARDED_FOR")

	ctx := req.Context()

	// what is IP or id of sender?
	fmt.Println("token req RemoteAddr", remoteAddr) // F I X M E: use db see below

	now := time.Now().Unix()
	numberOfMinutesPassed := (now - bootTimeSec) / 6 // now it's 10 sec
	if tokensServed > numberOfMinutesPassed {
		iot.BadTokenRequests.Inc()
		http.Error(w, "Token dispenser is too busy now. Try in a minute, or, you could buy $5 worth and pass them to all your friends", 500)
		return
	}
	if numberOfMinutesPassed > 60 {
		// reset the allocator every hour
		bootTimeSec = now
		tokensServed = 0
	}

	var buff1024 [1024]byte
	n, err := req.Body.Read(buff1024[:])
	buf := buff1024[:n]
	fmt.Println("tpoken request read body", string(buf), n)
	_ = buf

	tokenRequest := &tokens.TokenRequest{}
	err = json.Unmarshal(buf, tokenRequest)
	if err != nil {
		iot.BadTokenRequests.Inc()
		fmt.Println("TokenRequest err", err.Error())
		http.Error(w, err.Error(), 500)
		return
	} else {
		// todo: calc cost of this token and have limit.
		// move this phat routine somewhere else TODO:

		clientPublicKey := tokenRequest.Pkey
		if len(clientPublicKey) != 64 {
			iot.BadTokenRequests.Inc()
			http.Error(w, "bad client key", 500)
			return
		}

		//payload := tokenRequest.Payload // can we not use the request payload?
		// we only need variable sizes when collcecting money
		payload := tokens.KnotFreeTokenPayload{}
		payload.Connections = 2 // TODO: move into standard x-small token
		// 30 days
		payload.ExpirationTime = uint32(time.Now().Unix() + 60*60*24*30)

		payload.Input = 32 * 4  // TODO: move into standard x-small token
		payload.Output = 32 * 4 // TODO: move into standard x-small token

		payload.Issuer = "_9sh"
		payload.JWTID = tokens.GetRandomB64String()
		nonce := payload.JWTID

		// targetSite := "knotfree.net" // which is mapped to localhost by /etc/hosts
		// if os.Getenv("KNOT_KUNG_FOO") == "atw" {
		// 	targetSite = "knotfree0.com"
		// }

		payload.Subscriptions = 20 // TODO: move into standard x-small token

		//  Host:"building_bob_bottomline_boldness.knotfree2.com:8085"

		parts := strings.Split(req.Host, ".")
		partslen := len(parts)
		if partslen < 2 {
			//http.Error(w, "expected at least 2 parts in Host "+req.Host, 500)
			//return
			parts = strings.Split("local.localhost", ".")
			partslen = len(parts)
		}
		targetSite := parts[partslen-2] + "." + parts[partslen-1]

		payload.URL = targetSite + "/mqtt"

		exp := payload.ExpirationTime
		if exp > uint32(time.Now().Unix()+60*60*24*365) {
			// more than a year in the future not allowed now.
			exp = uint32(time.Now().Unix() + 60*60*24*365)
			fmt.Println("had long token ", string(payload.JWTID)) // TODO: store in db
		}

		cost := tokens.CalcTokenPrice(&payload, uint32(time.Now().Unix()))
		fmt.Println("token cost is " + fmt.Sprintf("%f", cost))

		if cost > 0.012 {
			http.Error(w, "token too expensive at "+fmt.Sprintf("%f", cost), 500)
			return
		}

		signingKey := tokens.GetPrivateKey("_9sh")
		tokenString, err := tokens.MakeToken(&payload, []byte(signingKey))
		if err != nil {
			iot.BadTokenRequests.Inc()
			http.Error(w, err.Error(), 500)
			return
		}
		signingKey = "unused now"

		when := time.Unix(int64(exp), 0)
		year, month, day := when.Date()

		comments := make([]interface{}, 3)
		tmp := fmt.Sprintf(" expires: %v-%v-%v", year, int(month), day)
		comments[0] = tokenRequest.Comment + tmp
		comments[1] = "" //payload
		comments[2] = string(tokenString)
		returnval, err := json.Marshal(comments)
		returnval = []byte(strings.ReplaceAll(string(returnval), `"`, ``))
		returnval = []byte(strings.ReplaceAll(string(returnval), ` `, `_`))
		//fmt.Println("sending token package ", string(returnval))

		returnval = tokenString

		err = tokens.LogNewToken(ctx, &payload, remoteAddr)
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
		//time.Sleep(8 * time.Second)
		time.Sleep(1 * time.Second)
		w.Write(bytes)
		tokensServed++
		fmt.Println("done sending free token")
	}
}

type webHandler struct { // is this even used?
	ce *iot.ClusterExecutive

	//	fs1 http.Handler
	fs2 http.Handler

	//fs := http.FileServer(http.Dir("./docs/_site"))
	// supermux.sub.Handle("/", fs)

}

// serves /
func (api webHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	fmt.Println("webHandler ServeHTTP", r.RequestURI)

	// isGotohere := false // when this is true we serve the react build in docs/_site2
	// domainParts := strings.Split(r.Host, ".")
	// if len(domainParts) >= 2 && (domainParts[len(domainParts)-2] == "gotohere" || domainParts[len(domainParts)-2] == "gotolocal") {
	// 	isGotohere = true
	// }
	// just kidding. we're dumping the old _site jekyl thing
	// the react build at _site2 switches for knotfree and gotohere.
	// isGotohere = true
	// if isGotohere {
	api.fs2.ServeHTTP(w, r)
	// } else {
	// 	api.fs1.ServeHTTP(w, r)
	// }
}
