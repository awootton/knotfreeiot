package iot

// Copyright 2024 Alan Tracey Wootton
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

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/awootton/knotfreeiot/packets"
	"github.com/awootton/knotfreeiot/tokens"
)

func GetRandomB64String() string {
	var tmp [18]byte
	rand.Read(tmp[:])
	return base64.RawURLEncoding.EncodeToString(tmp[:])
}

type pinfo struct {
	// these are the reply buffers
	buff []byte
}
type RequestReplyStruct struct {
	// originalRequest []byte
	firstLine  string
	replyParts []pinfo
}

type myWriterType struct {
	packets      chan packets.Interface
	myPipeReader io.Reader
	myPipeWriter io.Writer
}

var _ = printBinary

func printBinary(bytes []byte) string {
	res := ""
	for i := 0; i < len(bytes); i++ {
		b := bytes[i]
		if b >= 32 && b < 127 {
			res += string(b)
		} else {
			res += "0x" + hex.EncodeToString(bytes[i:i+1])
		}
	}
	return res
}

func (w *myWriterType) Write(p []byte) (n int, err error) {

	// fmt.Println("myWriterType write:", printBinary(p))

	n, err = w.myPipeWriter.Write(p)
	if err != nil {
		fmt.Println("myWriterType write err:", err)
	}
	return n, err
}

func (w *myWriterType) Read(p []byte) (n int, err error) {
	n, err = w.myPipeReader.Read(p)
	if err != nil {
		fmt.Println("myWriterType read err:", err)
	}
	//	fmt.Println("myWriterType read:", printBinary(p[0:n]))
	return n, err
}

func HandleHttpSubdomainRequest(w http.ResponseWriter, r *http.Request, ex *Executive, subDomain string, theHost string) {

	r.Close = true // close when done? // see below

	// see if it even exists
	look := packets.Lookup{}
	look.Address.FromString(subDomain)
	look.SetOption("cmd", []byte("proxy-status"))
	gotPacket, err := ex.ce.PacketService.GetPacketReply(&look)
	if err != nil {
		fmt.Println("PacketService error ", err)
		http.Error(w, "PacketService error "+err.Error(), 500)
		return
	}
	gotPacketStr := gotPacket.(*packets.Send).Payload
	theStatus := ProxyStatusReturnType{}
	err = json.Unmarshal([]byte(gotPacketStr), &theStatus)
	if err != nil {
		fmt.Println("json unmarshal error ", err)
		http.Error(w, "json unmarshal error "+err.Error(), 500)
		return
	}
	if !theStatus.Exists {
		fmt.Println("subdomain does not exist ", subDomain)
		http.Error(w, "subdomain does not exist "+subDomain, 500)
		return
	}
	if theStatus.Static != "" {

		urlString := theStatus.Static
		urlString = strings.TrimSuffix(urlString, "/")
		// proxy
		if r.RequestURI == "/" {
			urlString += "/index.html"
		} else {
			urlString += r.RequestURI
		}
		fmt.Println("subdomain serving static ", urlString)

		// url, err := url.Parse(urlString)
		// if err != nil {
		// 	fmt.Println("url.Parse error ", urlString)
		// 	http.Error(w, "url.Parse "+urlString, 500)
		// 	return

		//proxy := httputil.NewSingleHostReverseProxy(url)
		// proxy := &httputil.ReverseProxy{
		// 	Rewrite: func(r *httputil.ProxyRequest) {
		// 		r.SetURL(url)
		// 		r.Out.Host = r.In.Host // if desired
		// 	},
		// }
		// proxy.ServeHTTP(w, r)

		// it's a static file. Just do a get.
		// resp, err := http.Get(urlString)
		newRequest, err := http.NewRequest("GET", urlString, nil)
		if err != nil {
			fmt.Println("http.Get error ", urlString)
			http.Error(w, "http.Get "+urlString, 500)
			return
		}
		// copy headers
		for key, val := range r.Header {
			for i := 0; i < len(val); i++ {
				newRequest.Header.Add(key, val[i])
			}
		}
		resp, err := http.DefaultClient.Do(newRequest)
		if err != nil {
			fmt.Println("http.Get ", urlString)
			http.Error(w, "http.Get "+urlString, 500)
			return
		}

		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body) // what if it's huge?
		if err != nil {
			fmt.Println("http.Get io.ReadAllerror ", urlString)
			http.Error(w, "http.Get io.ReadAll "+urlString, 500)
			return
		}
		w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
		w.Write(body)
		return
	}
	fmt.Println("subdomain is online ", theStatus.Online)
	if !theStatus.Online {
		fmt.Println("subdomain is not online to reply ", subDomain)
		http.Error(w, "subdomain is not online to reply "+subDomain, 404)
		return
	}

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
	isDebg := false

	//fmt.Println("http header ", r.Header) // it's a map with Cookie
	// r.RequtURI is "/"
	// r.URL is "/"
	// write the header to a buffer
	firstLine := r.Method + " " + r.URL.String() + " " + r.Proto + "\n"
	// fmt.Println("first line", firstLine[0:len(firstLine)-2])
	if strings.Contains(firstLine, "debg=12345678") {
		isDebg = true
		fmt.Println("first line", firstLine[0:len(firstLine)-2])
	}
	buf := new(bytes.Buffer)
	buf.WriteString(firstLine)
	// transfer all the headers.
	// leave some out. We may have to add them back but they clog up the small microcontrollers. TODO:
	for key, val := range r.Header {
		if key == "Cookie" {
			continue // don't pass the cookie
		}
		if strings.HasPrefix(key, "X-") {
			continue
		}
		if strings.HasPrefix(key, "Sec-") {
			continue
		}
		if strings.HasPrefix(key, "User-Agent") {
			continue
		}
		for i := 0; i < len(val); i++ {
			tmp := key + ": " + val[i] + "\r\n"
			buf.WriteString(tmp)
		}
	}
	buf.WriteString("\r\n")
	// write the body to a buffer
	n, err := buf.Write(theBody)
	if err != nil || (n != len(theBody)) {
		http.Error(w, "http theBody write ", 500)
	}

	fmt.Println("http is request ", firstLine[0:len(firstLine)-2])

	pastWritesIndex := 0
	packetStruct := &RequestReplyStruct{}

	// packetStruct.originalRequest = buf.Bytes()
	packetStruct.firstLine = firstLine[0 : len(firstLine)-2]

	// we need to make a contact. TODO: reuse a contact for all these.
	// FIXMEL use ServiceContact for this
	// make a reply address
	// serialize the request
	// publish it.
	// wait for the response and put that into the w http.ResponseWriter
	// copy over the response headers
	// unsub the reply address
	// close the contact.

	packetsChan := make(chan packets.Interface, 100)
	contact := &ContactStruct{}
	// hook the real writer
	myWriter := &myWriterType{}
	myWriter.packets = packetsChan

	myWriter.myPipeReader, myWriter.myPipeWriter = io.Pipe() // this is the pipe that the packets will come in on

	// bufiowriter := bufio.NewWriter(myWriter)

	contact.SetWriter(myWriter) // myWriter)
	AddContactStruct(contact, contact, ex.Config)
	go func() {
		for {
			select {
			case <-contact.ClosedChannel:
				fmt.Println("http subdomain handler contact closed", contact.GetKey().Sig())
				return
			default:

				packet, err := packets.ReadPacket(myWriter)
				if err != nil || packet == nil {
					// the buffer only had a partial packet
					fmt.Println("ERROR packet read fail ", err)
					contact.DoClose(err)
					return
				}
				if isDebg {
					fmt.Println("http subdomain handler got packet ", packet.String())
				}
				packetsChan <- packet
			}
		}
	}()

	if isDebg {
		contact.LogMeVerbose = true
	}
	startTime := time.Now()
	fmt.Println("serving subdomain ", subDomain, "  of "+theHost+r.RequestURI, "con=", contact.GetKey().Sig())

	defer func() {
		fmt.Println("DONE subdomain ", subDomain, "con=", contact.GetKey().Sig(), "time=", time.Since(startTime))

		if isDebg {
			fmt.Println("contact normal close", contact.GetKey().Sig())
		}
		contact.DoClose(errors.New("normal close"))
	}()

	// can we dispense with the stupid connect packet? TODO:
	// we could set contact.token = "dummy" and
	// contact.nextBillingTime = now + 100 years
	// and dispense with the billing?

	connect := packets.Connect{}
	connect.SetOption("token", []byte(tokens.GetImpromptuGiantToken()))
	if isDebg {
		connect.SetOption("debg", []byte("12345678"))
	}
	err = PushPacketUpFromBottom(contact, &connect)
	if err != nil {
		fmt.Println("connect problems subdomain dial conn ", err)
		http.Error(w, err.Error(), 500)
		return
	}

	// Subscribe
	myRandomName := GetRandomB64String()
	subs := packets.Subscribe{}
	subs.Address.FromString(myRandomName)
	subs.Address.EnsureAddressIsBinary()
	//
	if isDebg {
		subs.SetOption("debg", []byte("12345678"))
		fmt.Println(" our address will be ", subs.Address.String())
	}
	err = PushPacketUpFromBottom(contact, &subs)
	_ = err

	// now we have to wait for the suback to come back
	haveSuback := false
	for !haveSuback {
		select {
		case <-contact.ClosedChannel:
			haveSuback = true
		case packet := <-packetsChan:
			// see if it's a suback
			// fmt.Println("waiting for suback on gotDataChan.TheChan got ", cmd.Sig())
			if packet == nil {
				fmt.Println("ERROR nil packet waiting for suback. Never happens.")
			} else {
				subcmd, ok := packet.(*packets.Subscribe)
				if !ok {
					fmt.Println("ERROR wrong packet waiting for suback  ")
				} else {
					if isDebg {
						fmt.Println("http handler have suback  ", subcmd.Sig())
					}
					haveSuback = true
				}
			}
			// we have to wait for the suback to come back
		case <-time.After(4 * time.Second):
			errMsg := "timed out waiting for suback reply " + firstLine[0:len(firstLine)-2]
			fmt.Println(errMsg)
			http.Error(w, errMsg, 500)
			return
		}
	}

	if buf.Len() > 60*1024 {
		// stream it
		fmt.Println("ERROR fixme: implement this streaming thing")
	} else {

		// just send it all at once in one Send
		pub := packets.Send{}

		// copy the options over
		parts := strings.Split(firstLine, "?") // ie GET /get/c?debg=12345678 HTTP/1
		if len(parts) > 1 {
			parts = strings.Split(parts[1], " ")
			parts = strings.Split(parts[0], "&")
			for _, part := range parts {
				kv := strings.Split(part, "=")
				if len(kv) == 2 {
					pub.SetOption(kv[0], []byte(kv[1]))
				}
			}
		}
		got, ok := pub.GetOption("debg")
		if ok && string(got) == "12345678" {
			isDebg = true
		}

		pub.Address.FromString(subDomain) // !!!!!
		pub.Source = subs.Address
		// fmt.Println(" our send addr is ", pub.Address.String()) // atw delete
		pub.Address.EnsureAddressIsBinary()
		// fmt.Println(" our send addr is ", pub.Address.String())  // atw delete
		// fmt.Println(" our return addr is ", pub.Source.String()) // atw delete
		pub.Payload = []byte("GET " + r.URL.String() + " HTTP/1.1\n\n")
		pub.Payload = buf.Bytes()

		if isDebg {
			fmt.Println("publish PushPacketUpFromBottom", pub.Sig())
		}
		err = PushPacketUpFromBottom(contact, &pub)
		_ = err
	}

	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "webserver doesn't support hijacking", http.StatusInternalServerError)
		return
	}
	hijackedconn, responseBuffer, err := hj.Hijack()
	_ = hijackedconn
	if err != nil {
		fmt.Println("hijack error  ", err)
	}
	defer func() {
		// trying to make the nginx ingress happy. It never is.
		// it keeps acting like it's not getting the whole response.
		fmt.Println("flushing hijack socket ", r.URL.String(), contact.GetKey().Sig(), time.Since(startTime))
		responseBuffer.Flush()
		time.Sleep(1 * time.Millisecond) // superstition ain't the way
		responseBuffer.Flush()
		r.Close = true // see above also
		err = hijackedconn.Close()
		if err != nil {
			fmt.Println("hijackedconn close error  ", err)
		}
	}()

	{ // The Receive-a-packet loop
		running := true
		//hadHeader := false
		theLengthWeNeed := 0
		theAmountWeGot := 0
		for running {
			select {
			case <-contact.ClosedChannel:
				// fmt.Println("contact closed")
				running = false
			case packet := <-packetsChan:

				if isDebg {
					fmt.Println("Receive-a-packet loop got ", packet.Sig())
				}
				_, ok := packet.(*packets.Subscribe)
				if ok {
					continue // excess subacks are expected. why?
				}
				switch v := packet.(type) {
				case *packets.Send:
					snd := v
					packetCountStr, ok := snd.GetOption("of")
					if ok {
						_ = packetCountStr
						// fmt.Println("packet count total= ", packetCountStr)
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
					// fmt.Println("packet count is ", packetCountStr)
					//if packetCount != packetsReceived {
					//	fmt.Println("we seem to have lost a PACKET:", packetCount, packetsReceived)
					//} pastWritesIndex
					// pad out the buffer
					for packetIncomingIndex >= len(packetStruct.replyParts) {
						pi := &pinfo{}
						packetStruct.replyParts = append(packetStruct.replyParts, *pi)
					}
					currentPayload := snd.Payload

					// fmt.Println("have http reply packet #", packetIncomingIndex, "for ", firstLine)
					if packetIncomingIndex == 0 {
						headerEndBytes := []byte("\r\n\r\n")
						headerPos := bytes.Index(snd.Payload, headerEndBytes)
						if headerPos <= 0 {
							fmt.Println("no header was found in first packet", string(snd.Payload))
						} else {
							// parse the header
							header := snd.Payload[0:headerPos]
							clStr := "Content-Length:"
							clPos := bytes.Index(header, []byte(clStr))
							if clPos <= 0 {
								fmt.Println("no Content-Length was found in the header", string(header))
							}
							hpart := header[clPos+len(clStr):]
							lineEndBytes := []byte("\r\n")
							endPos := bytes.Index(hpart, lineEndBytes)
							//fmt.Println("is this a number? ", hpart[0:endPos])
							cldigits := string(hpart[0:endPos])
							i, err := strconv.Atoi(strings.Trim(cldigits, " "))
							if err != nil {
								fmt.Println("ERROR finding Content-Length", string(hpart[0:endPos]))
							}
							// fmt.Println("theLengthWeNeed is ", i)
							theLengthWeNeed = i + len(header) + 4

							// we have to transfer the user options to the header
							// we insert the options onto the currentPayload
							// split the currentPayload into header and the rest
							headerStart := string(currentPayload[0:int(headerPos)]) // force a copy
							pastHeader := currentPayload[int(headerPos):]           // contains the \r\n\r\n, might contain some body
							keys, bvalues := snd.GetOptionKeys()
							values := make([]string, len(keys))
							for n := 0; n < len(keys); n++ {
								k := keys[n]
								v := bvalues[n]
								values[n] = string(v)
								// fmt.Println("Options k v ", k, values[n])
								_ = k
							}

							// fmt.Println("headerStart  ", string(headerStart))
							// fmt.Println("pastHeader  ", string(pastHeader))
							if len(keys) > 0 {
								headerStart += "\r\n"
								theLengthWeNeed += 2
							}
							// fmt.Println("headerStart 2 ", string(headerStart)+"\n\n")
							for i := 0; i < len(keys); i++ {
								k := keys[i]
								v := values[i]

								// fmt.Println("adding ", k, ":", string(v))
								headerStart += k
								theLengthWeNeed += len(k)
								//fmt.Println("headerStart 3 ", string(headerStart)+"\n\n")
								headerStart += ": "
								theLengthWeNeed += 2
								//fmt.Println("headerStart 4 ", string(headerStart)+"\n\n")

								// fmt.Println("addingvalue  ", string(values[i])+"\n\n")
								headerStart += v
								theLengthWeNeed += len(v)
								//fmt.Println("headerStart 5 ", string(headerStart)+"\n\n")
								if i < len(keys)-1 {
									headerStart += "\r\n"
									theLengthWeNeed += 2
								}
								// fmt.Println("headerStart 6 ", string(headerStart)+"\n\n")
							}
							// fmt.Println("headerStart  ", string(headerStart)+"\n\n")
							currentPayload = append([]byte(headerStart), pastHeader...)
							// fmt.Println("new payload is ", string(currentPayload)+"\n\n")
						}
					}
					packetStruct.replyParts[packetIncomingIndex].buff = currentPayload

					for { // loop over packetlist stuff we can write
						if pastWritesIndex >= len(packetStruct.replyParts) {
							break // at the end
						}
						nextPi := packetStruct.replyParts[pastWritesIndex]

						if isDebg {
							fmt.Println(contact.GetKey().Sig(), " got a reply payload packet index ", pastWritesIndex)
						}
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
						//fmt.Println("So far we have got", theAmountWeGot, " of ", theLengthWeNeed, "for", packetStruct.firstLine)
						if theAmountWeGot >= theLengthWeNeed {
							// fmt.Println("looks like we made it ! :")
							if isDebg {
								fmt.Println("Request complete", packetStruct.firstLine)
							}
							running = false
						}
						responseBuffer.Flush()
					}

				default:
					// no match. do nothing. panic?
					fmt.Println("got weird packet instead of publish ", reflect.TypeOf(packet))
					responseBuffer.Write([]byte("error got weird packet"))
					running = false
				}
			//
			case <-time.After(4567 * time.Millisecond): // sooner than nginx
				errMsg := "timed out waiting for reply (receiver offline)" + contact.GetKey().Sig() + " " + firstLine[0:len(firstLine)-2]
				fmt.Println(errMsg)
				// http.Error(responseBuffer, errMsg, 500)
				responseBuffer.WriteString(errMsg)
				running = false
			}
		}

		unsub := packets.Unsubscribe{}
		unsub.Address.FromString(myRandomName)
		if isDebg {
			unsub.SetOption("debg", []byte("12345678"))
		}
		err = PushPacketUpFromBottom(contact, &unsub)
		_ = err
	}
}
