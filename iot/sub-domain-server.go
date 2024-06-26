package iot

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math/rand"
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
	originalRequest []byte
	firstLine       string
	replyParts      []pinfo
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

func HandleHttpSubdomainRequest(w http.ResponseWriter, r *http.Request, ex *Executive, subDomain string) {

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
	for key, val := range r.Header {
		if key == "Cookie" {
			continue // don't pass the cookie
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

	fmt.Println("serving subdomain ", subDomain, "  of "+r.Host+r.RequestURI, "con=", contact.GetKey().Sig())

	defer func() {
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
	conn, responseBuffer, err := hj.Hijack()
	if err != nil {
		fmt.Println("hijack error  ", err)
	}
	defer func() {
		// fmt.Println("closing hijack socket " + r.URL.String() + "\n")
		conn.Close() // closes contact.ClosedChannel
		contact.DoClose(nil)
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
							responseBuffer.Flush()
							// push a close packet or something
							// close the connection -- below

							if isDebg {
								fmt.Println("Request complete", packetStruct.firstLine)
							}
							running = false
						}
						//responseBuffer.Flush()
					}

				default:
					// no match. do nothing. panic?
					fmt.Println("got weird packet instead of publish ", reflect.TypeOf(packet))
					w.Write([]byte("error got weird packet"))
					running = false
				}
			// is this the only way to know that we're done??
			case <-time.After(5 * time.Second):
				errMsg := "timed out waiting for html reply " + contact.GetKey().Sig() + " " + firstLine[0:len(firstLine)-2]
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
		unsub.Address.FromString(myRandomName)
		if isDebg {
			unsub.SetOption("debg", []byte("12345678"))
		}
		err = PushPacketUpFromBottom(contact, &unsub)
		_ = err
		// defered contact.DoClose(nil)
	}
}
