package iot

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	mathrand "math/rand"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/awootton/knotfreeiot/packets"
	"github.com/awootton/knotfreeiot/tokens"
)

// GetRandomB64String is a slower, more cryptographically secure random string generator. 18 bytes is 144 bits, which is more than enough for a session key.
// We use them for the sub of a service-contact, which happens infrequently.
func GetRandomB64String() string {
	var tmp [18]byte
	rand.Read(tmp[:])
	return base64.RawURLEncoding.EncodeToString(tmp[:])
}

const anArrayOf64Chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789QZ"

// GetRandomStringOf28 returns a random string of 28 NOT base64 characters.
// much faster. We use these for session key/val and have to make one up for every packet sent.
// 28 bytes is 224 bits, which is more than enough for a session key (it keeps telling this lie, sigh).
func GetRandomStringOf28() [28]byte {

	// go faster, be less cryptographically secure. 28 bytes is 224 bits, which is more than enough for a session key.
	// silly AI, it's 6 * 28 = 168 bits, not 224.
	var destination [28]byte

	bigint := mathrand.Uint64()
	for i := 0; i < 10; i++ { // 10, really? lol.
		destination[i] = anArrayOf64Chars[bigint&0x3F]
		bigint >>= 6
	}
	bigint = mathrand.Uint64()
	for i := 10; i < 20; i++ {
		destination[i] = anArrayOf64Chars[bigint&0x3F]
		bigint >>= 6
	}
	bigint = mathrand.Uint64()
	for i := 20; i < 28; i++ {
		destination[i] = anArrayOf64Chars[bigint&0x3F]
		bigint >>= 6
	}
	return destination
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

	// log.Println("myWriterType write:", printBinary(p))

	n, err = w.myPipeWriter.Write(p)
	if err != nil {
		log.Println("myWriterType write err:", err)
	}
	return n, err
}

func (w *myWriterType) Read(p []byte) (n int, err error) {
	n, err = w.myPipeReader.Read(p)
	if err != nil {
		log.Println("myWriterType read err:", err)
	}
	//	log.Println("myWriterType read:", printBinary(p[0:n]))
	return n, err
}

var servedCount int

func HandleHttpSubdomainRequest(w http.ResponseWriter, r *http.Request, ex *Executive, subDomain string, theHost string) {

	r.Close = true // close when done? // see below

	// see if it even exists
	look := packets.Lookup{}
	look.Address.FromString(subDomain)
	look.SetOption("cmd", []byte("proxy-status"))
	gotPacket, err := ex.ce.GetPacketService().GetPacketReply(&look)
	if err != nil {
		log.Println("PacketService error ", err)
		http.Error(w, "PacketService error "+err.Error(), 500)
		return
	}
	theStatus := ProxyStatusReturnType{}
	gotPacketStr := string(gotPacket.(*packets.Send).Payload)
	if gotPacketStr[0] != '{' { // usually "error:" strings.HasPrefix("error", gotPacketStr) {
		// log.Println("PacketService error ", gotPacketStr)
		// http.Error(w, "HandleHttpSubdomainRequest "+gotPacketStr, 500)
		// return
	} else {
		err = json.Unmarshal([]byte(gotPacketStr), &theStatus)
		if err != nil {
			log.Println("ProxyStatusReturnType json unmarshal error ", err, []byte(gotPacketStr))
			http.Error(w, "json unmarshal error "+err.Error(), 500)
			return
		}
	}
	if theStatus.Exists {
		log.Println("subdomain DOES exist ", subDomain)
	} else {
		log.Println("subdomain does NOT exist ", subDomain)
	}

	if !theStatus.Exists {
		message := "This thing does not exist or is not currently online: " + subDomain
		log.Println(message)
		http.Error(w, message, 500)
		return
	}
	if theStatus.Static != "" {

		proxyName := theStatus.Static
		proxyName = strings.TrimSuffix(proxyName, "/")
		// proxy
		requestURI := r.RequestURI
		log.Println("subdomain serving static via HandleByProxy", proxyName, requestURI)
		HandleByProxy(w, r, ex, subDomain, theHost, proxyName, requestURI)

		return
	}
	if theStatus.Online {
		log.Println("subdomain IS online ", subDomain)
	} else {
		log.Println("subdomain NOT online ", subDomain)
	}
	if !theStatus.Online {
		message := "This thing is known but is not currently online: " + subDomain
		log.Println(message)
		http.Error(w, message, 404)
		return
	}

	clen := r.ContentLength
	if clen > 63*1024 {
		log.Println("http request packet too long ")
		http.Error(w, "http request packet too long ", 500)
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

	//log.Println("http header ", r.Header) // it's a map with Cookie
	// write the header to a buffer
	firstLine := r.Method + " " + r.URL.String() + " " + r.Proto + "\r\n"
	// log.Println("first line", firstLine[0:len(firstLine)-2])
	if strings.Contains(firstLine, "debg=12345678") {
		isDebg = true
		log.Println("first line", firstLine[0:len(firstLine)-2])
	}
	buf := new(bytes.Buffer)
	buf.WriteString(firstLine)
	// transfer all the headers into the packet.
	// leave some out?. We may have to add them back but they clog up the small microcontrollers. TODO:
	for key, val := range r.Header {
		// if key == "Cookie" {
		// 	continue // don't pass the cookie
		// }
		// if strings.HasPrefix(key, "X-") {
		// 	continue
		// }
		// if strings.HasPrefix(key, "Sec-") {
		// 	continue
		// }
		// if strings.HasPrefix(key, "User-Agent") {
		// 	continue
		// }
		for i := 0; i < len(val); i++ {
			tmp := key + ": " + val[i] + "\r\n"
			buf.WriteString(tmp)
		}
	}
	// add the host back
	buf.WriteString(fmt.Sprintf("Host: %s\r\n", theHost))
	buf.WriteString("\r\n")
	// write the body to the buffer
	n, err := buf.Write(theBody)
	if err != nil || (n != len(theBody)) {
		http.Error(w, "http theBody write ", 500)
	}
	// log.Println("http is requestbuff ", buf.String())
	log.Println("http is request ", firstLine[0:len(firstLine)-2])

	startTime := time.Now()
	_ = startTime
	log.Println("serving subdomain ", subDomain, "  of "+theHost+r.RequestURI)

	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "FATAL webserver doesn't support hijacking", http.StatusInternalServerError)
		return
	}
	hijackedconn, responseBuffer, err := hj.Hijack()
	_ = hijackedconn
	if err != nil {
		log.Println("hijack error  ", err)
		http.Error(w, "hijack error "+err.Error(), 500)
		return

	}

	defer func() {
		// trying to make the nginx ingress happy. It never is.
		// it keeps acting like it's not getting the whole response.
		// log.Println("flushing hijack socket ", r.URL.String(), contact.GetKey().Sig(), time.Since(startTime))
		responseBuffer.Flush()
		// time.Sleep(1 * time.Millisecond) // superstition ain't the way
		// responseBuffer.Flush()
		r.Close = true // see above also
		err = hijackedconn.Close()
		if err != nil {
			log.Println("hijackedconn close error  ", err)
		}
	}()

	if buf.Len() > 60*1024 {
		// stream it
		errMsg := ("ERROR fixme: implement a streaming thing")
		log.Println(errMsg)
		// http.Error(responseBuffer, errMsg, 500)
		responseBuffer.WriteString(errMsg)
		return
	}
	// just send it all at once in one Send
	pub := packets.Send{}

	// copy the ergs over into the option kv of the packet
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
	pub.Address.EnsureAddressIsBinary()
	pub.Payload = buf.Bytes()

	// pub.SetOption("count", []byte(strconv.FormatInt(int64(servedCount), 10)))

	if isDebg {
		log.Println("sub-domain sending", pub.Sig())
		var buffer bytes.Buffer
		pub.Write(&buffer)
		log.Println("sub-domain sending hex", hex.EncodeToString(buffer.Bytes()))
	}
	//err = PushPacketUpFromBottom(contact, &pub)
	// no, just no.
	//gotPackets, err := ex.ce.GetPacketService().GetPacketGroupReplyLonger(&pub, time.Duration(5*time.Second))
	gotPackets, err := ex.ce.GetPacketService().GetPacketReplyLonger(&pub, time.Duration(5*time.Second))
	if err != nil {
		errMsg := "Error. " + err.Error() // + contact.GetKey().Sig() + " " + firstLine[0:len(firstLine)-2]
		log.Println(errMsg)
		// http.Error(responseBuffer, errMsg, 500)
		responseBuffer.WriteString(errMsg)
		return
	}
	// if len(gotPackets) != 1 {
	// 	errMsg := "Error. More than one response" // + contact.GetKey().Sig() + " " + firstLine[0:len(firstLine)-2]
	// 	// FIXME: write tests and implement.
	// 	log.Println(errMsg)
	// 	// http.Error(responseBuffer, errMsg, 500)
	// 	responseBuffer.WriteString(errMsg)
	// 	return
	// }
	snd := gotPackets.(*packets.Send)
	builder := InsertAdditionalHeaders(snd.Payload, *snd)
	responseBuffer.Write(builder.Bytes())
	responseBuffer.Flush()

	servedCount++

}

// InsertAdditionalHeaders inserts the optional packet key value pairs into the http header in the buffer.
func InsertAdditionalHeaders(buffer []byte, snd packets.Send) bytes.Buffer {

	var builder bytes.Buffer
	// find the end of the header
	pos := bytes.Index(buffer, []byte{0x0D, 0x0A, 0x0D, 0x0A})
	if pos == -1 {
		builder.Write(buffer)
	} else {
		builder.Write(buffer[:pos+2])
		keys, values := snd.GetOptionKeys()
		for i, key := range keys {
			val := values[i]
			builder.Write([]byte(key))
			builder.Write([]byte(": "))
			builder.Write([]byte(val))
			builder.Write([]byte("\r\n"))
		}
		builder.Write([]byte("\r\n"))
		builder.Write(buffer[pos+4:])
	}
	// str := builder.String()
	// log.Println("InsertAdditionalHeaders: ", str)
	// log.Println("InsertAdditionalHeaders from: ", string(buffer))
	return builder
}

func OldHandleHttpSubdomainRequest(w http.ResponseWriter, r *http.Request, ex *Executive, subDomain string, theHost string) {

	r.Close = true // close when done? // see below

	// see if it even exists
	look := packets.Lookup{}
	look.Address.FromString(subDomain)
	look.SetOption("cmd", []byte("proxy-status"))
	gotPacket, err := ex.ce.GetPacketService().GetPacketReply(&look)
	if err != nil {
		log.Println("PacketService error ", err)
		http.Error(w, "PacketService error "+err.Error(), 500)
		return
	}
	theStatus := ProxyStatusReturnType{}
	gotPacketStr := string(gotPacket.(*packets.Send).Payload)
	if gotPacketStr[0] != '{' { // usually "error:" strings.HasPrefix("error", gotPacketStr) {
		// log.Println("PacketService error ", gotPacketStr)
		// http.Error(w, "HandleHttpSubdomainRequest "+gotPacketStr, 500)
		// return
	} else {
		err = json.Unmarshal([]byte(gotPacketStr), &theStatus)
		if err != nil {
			log.Println("ProxyStatusReturnType json unmarshal error ", err, []byte(gotPacketStr))
			http.Error(w, "json unmarshal error "+err.Error(), 500)
			return
		}
	}
	if theStatus.Exists {
		log.Println("subdomain DOES exist ", subDomain)
	} else {
		log.Println("subdomain does NOT exist ", subDomain)
	}

	if !theStatus.Exists {
		message := "This thing does not exist or is not currently online: " + subDomain
		log.Println(message)
		http.Error(w, message, 500)
		return
	}
	if theStatus.Static != "" {

		proxyName := theStatus.Static
		proxyName = strings.TrimSuffix(proxyName, "/")
		// proxy
		requestURI := r.RequestURI
		log.Println("subdomain serving static via HandleByProxy", proxyName, requestURI)
		HandleByProxy(w, r, ex, subDomain, theHost, proxyName, requestURI)

		return
	}
	if theStatus.Online {
		log.Println("subdomain IS online ", subDomain)
	} else {
		log.Println("subdomain NOT online ", subDomain)
	}
	if !theStatus.Online {
		message := "This thing is known but is not currently online: " + subDomain
		log.Println(message)
		http.Error(w, message, 404)
		return
	}

	clen := r.ContentLength
	if clen > 63*1024 {
		log.Println("http packet too long ")
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

	//log.Println("http header ", r.Header) // it's a map with Cookie
	// r.RequtURI is "/"
	// r.URL is "/"
	// write the header to a buffer
	firstLine := r.Method + " " + r.URL.String() + " " + r.Proto + "\r\n"
	// log.Println("first line", firstLine[0:len(firstLine)-2])
	if strings.Contains(firstLine, "debg=12345678") {
		isDebg = true
		log.Println("first line", firstLine[0:len(firstLine)-2])
	}
	buf := new(bytes.Buffer)
	buf.WriteString(firstLine)
	// transfer all the headers.
	// leave some out. We may have to add them back but they clog up the small microcontrollers. TODO:
	for key, val := range r.Header {
		// if key == "Cookie" {
		// 	continue // don't pass the cookie
		// }
		// if strings.HasPrefix(key, "X-") {
		// 	continue
		// }
		// if strings.HasPrefix(key, "Sec-") {
		// 	continue
		// }
		// if strings.HasPrefix(key, "User-Agent") {
		// 	continue
		// }
		for i := 0; i < len(val); i++ {
			tmp := key + ": " + val[i] + "\r\n"
			buf.WriteString(tmp)
		}
	}
	// add the host back
	fmt.Fprintf(buf, "Host: %s\r\n", theHost)
	buf.WriteString("\r\n")
	// write the body to the buffer
	n, err := buf.Write(theBody)
	if err != nil || (n != len(theBody)) {
		http.Error(w, "http theBody write ", 500)
	}
	// add w.Header().Add("Access-Control-Allow-Origin", "*")
	buf.WriteString("Access-Control-Allow-Origin: *\r\n")

	// log.Println("http is requestbuff ", buf.String())
	log.Println("http is request ", firstLine[0:len(firstLine)-2])

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
				log.Println("http subdomain handler contact closed", contact.GetKey().Sig())
				return
			default:

				packet, err := packets.ReadPacket(myWriter)
				if err != nil || packet == nil {
					// the buffer only had a partial packet
					log.Println("ERROR packet read fail ", err)
					contact.DoClose(err)
					return
				}
				if isDebg {
					log.Println("http subdomain handler got packet ", packet.String())
				}
				packetsChan <- packet
			}
		}
	}()

	if isDebg {
		contact.LogMeVerbose = true
	}
	startTime := time.Now()
	log.Println("serving subdomain ", subDomain, "  of "+theHost+r.RequestURI, "con=", contact.GetKey().Sig())

	defer func() {
		log.Println("DONE subdomain ", subDomain, "con=", contact.GetKey().Sig(), "time=", time.Since(startTime))

		if isDebg {
			log.Println("contact normal close", contact.GetKey().Sig())
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
		log.Println("connect problems subdomain dial conn ", err)
		http.Error(w, err.Error(), 500)
		return
	}

	// Subscribe
	myRandomName := GetRandomB64String()
	subs := packets.Subscribe{}
	subs.Address.FromString(myRandomName)
	subs.Address.EnsureAddressIsBinary()
	// log.Println(" our return address will be ", hex.EncodeToString(subs.Address.Bytes))
	subs.SetOption("xxxx", []byte("TODO: reuse the same contact"))
	//
	if isDebg {
		subs.SetOption("debg", []byte("12345678"))
		log.Println("sub-domain our address will be ", subs.Address.String())
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
			// log.Println("waiting for suback on gotDataChan.TheChan got ", cmd.Sig())
			if packet == nil {
				log.Println("ERROR nil packet waiting for suback. Never happens.")
			} else {
				subcmd, ok := packet.(*packets.Subscribe)
				if !ok {
					log.Println("ERROR wrong packet waiting for suback  ")
				} else {
					if isDebg {
						log.Println("sub-domain http handler have suback  ", subcmd.Sig())
					}
					haveSuback = true
				}
			}
			// we have to wait for the suback to come back
		case <-time.After(4 * time.Second):
			errMsg := "timed out waiting for suback reply " + firstLine[0:len(firstLine)-2]
			log.Println(errMsg)
			http.Error(w, errMsg, 500)
			return
		}
	}

	if buf.Len() > 60*1024 {
		// stream it
		log.Println("ERROR fixme: implement this streaming thing")
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
		// log.Println(" our send addr is ", pub.Address.String())
		pub.Address.EnsureAddressIsBinary()
		// log.Println(" our send addr is ", pub.Address.String())
		// log.Println(" our return addr is ", pub.Source.String())
		// pub.Payload = []byte("GET " + r.URL.String() + " HTTP/1.1\n\n")
		pub.Payload = buf.Bytes()

		pub.SetOption("count", []byte(strconv.FormatInt(int64(servedCount), 10)))

		if isDebg {
			log.Println("sub-domain sending", servedCount, pub.Sig())
			var buffer bytes.Buffer
			pub.Write(&buffer)
			log.Println("sub-domain sending hex", hex.EncodeToString(buffer.Bytes()))
		}
		CheckSendPacket(&pub)
		err = PushPacketUpFromBottom(contact, &pub)
		_ = err
		servedCount++
	}

	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "FATAL webserver doesn't support hijacking", http.StatusInternalServerError)
		return
	}
	hijackedconn, responseBuffer, err := hj.Hijack()
	_ = hijackedconn
	if err != nil {
		log.Println("hijack error  ", err)
	}
	defer func() {
		// trying to make the nginx ingress happy. It never is.
		// it keeps acting like it's not getting the whole response.
		// log.Println("flushing hijack socket ", r.URL.String(), contact.GetKey().Sig(), time.Since(startTime))
		responseBuffer.Flush()
		time.Sleep(1 * time.Millisecond) // superstition ain't the way
		responseBuffer.Flush()
		r.Close = true // see above also
		err = hijackedconn.Close()
		if err != nil {
			log.Println("hijackedconn close error  ", err)
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
				// log.Println("contact closed")
				running = false
			case packet := <-packetsChan:

				if isDebg {
					log.Println("Receive-a-packet loop got ", packet.Sig())
				}
				_, ok := packet.(*packets.Subscribe)
				if ok {
					continue // excess subacks are expected. why?
				}
				switch v := packet.(type) {
				case *packets.Send:
					snd := v
					packetOfStr, hasOf := snd.GetOption("of")
					packetCountStr, hasIndx := snd.GetOption("indx")
					if !hasOf && !hasIndx {
						// just write the reply and we're done.
						// need to move options to the header TODO: FIXME: this is a bug
						builder := InsertAdditionalHeaders(snd.Payload, *snd)
						responseBuffer.Write(builder.Bytes())
						responseBuffer.Flush()
					} else {
						// this is all buggy crap
						// we USED get replies in parts. Now we get them all at once.
						// packetOfStr, ok := snd.GetOption("of")
						if hasOf {
							_ = packetOfStr
							// log.Println("packet count total= ", packetCountStr)
							// we have the last packet.
							running = false
							break
						}
						// packetCountStr, ok = snd.GetOption("indx")
						if !hasIndx {
							packetCountStr = []byte("0")
						}
						if packetCountStr[0] == '[' { // some idiot wrapped it in []
							packetCountStr = packetCountStr[1:]
						}
						if packetCountStr[len(packetCountStr)-1] == ']' {
							packetCountStr = packetCountStr[0 : len(packetCountStr)-1]
						}
						packetIncomingIndex, _ := strconv.Atoi(string(packetCountStr))
						// log.Println("packet count is ", packetCountStr)
						//if packetCount != packetsReceived {
						//	log.Println("we seem to have lost a PACKET:", packetCount, packetsReceived)
						//} pastWritesIndex
						// pad out the buffer
						for packetIncomingIndex >= len(packetStruct.replyParts) {
							pi := &pinfo{}
							packetStruct.replyParts = append(packetStruct.replyParts, *pi)
						}
						currentPayload := snd.Payload

						// log.Println("have http reply packet #", packetIncomingIndex, "for ", firstLine)
						// TODO: don't parse the reply packet at all. Just send it.
						// the sender will tell us when we have the last packet.
						// needs tests.
						if packetIncomingIndex == 0 {
							headerEndBytes := []byte("\r\n\r\n")
							headerPos := bytes.Index(snd.Payload, headerEndBytes)
							if headerPos <= 0 {
								log.Println("no header was found in first packet", string(snd.Payload))
							} else {
								// parse the header
								header := snd.Payload[0:headerPos]
								clStr := "content-length:"
								lowheader := strings.ToLower(string(header))
								clPos := bytes.Index([]byte(lowheader), []byte(clStr))
								if clPos <= 0 {
									log.Println("no Content-Length was found in the header", string(header))
								}
								hpart := header[clPos+len(clStr):]
								lineEndBytes := []byte("\r\n")
								endPos := bytes.Index(hpart, lineEndBytes)
								if endPos <= 0 {
									log.Println("no end of line was found in the header", string(hpart), lowheader)
									endPos = len(hpart)
									log.Println("is this a number? ", hpart[0:endPos])
								}
								cldigits := string(hpart[0:endPos])
								i, err := strconv.Atoi(strings.Trim(cldigits, " "))
								if err != nil {
									log.Println("ERROR finding Content-Length", string(hpart[0:endPos]))
								}
								// log.Println("theLengthWeNeed is ", i)
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
									// log.Println("Options k v ", k, values[n])
									_ = k
								}

								// log.Println("headerStart  ", string(headerStart))
								// log.Println("pastHeader  ", string(pastHeader))
								if len(keys) > 0 {
									headerStart += "\r\n"
									theLengthWeNeed += 2
								}
								// log.Println("headerStart 2 ", string(headerStart)+"\n\n")
								for i := 0; i < len(keys); i++ {
									k := keys[i]
									v := values[i]

									// log.Println("adding ", k, ":", string(v))
									headerStart += k
									theLengthWeNeed += len(k)
									//log.Println("headerStart 3 ", string(headerStart)+"\n\n")
									headerStart += ": "
									theLengthWeNeed += 2
									//log.Println("headerStart 4 ", string(headerStart)+"\n\n")

									// log.Println("addingvalue  ", string(values[i])+"\n\n")
									headerStart += v
									theLengthWeNeed += len(v)
									//log.Println("headerStart 5 ", string(headerStart)+"\n\n")
									if i < len(keys)-1 {
										headerStart += "\r\n"
										theLengthWeNeed += 2
									}
									// log.Println("headerStart 6 ", string(headerStart)+"\n\n")
								}
								// log.Println("headerStart  ", string(headerStart)+"\n\n")
								currentPayload = append([]byte(headerStart), pastHeader...)
								// log.Println("new payload is ", string(currentPayload)+"\n\n")
							}
						}
						packetStruct.replyParts[packetIncomingIndex].buff = currentPayload

						for { // loop over packetlist stuff we can write
							if pastWritesIndex >= len(packetStruct.replyParts) {
								break // at the end
							}
							nextPi := packetStruct.replyParts[pastWritesIndex]

							if isDebg {
								log.Println(contact.GetKey().Sig(), " got a reply payload packet index ", pastWritesIndex)
							}
							n, err := responseBuffer.Write(nextPi.buff)
							pastWritesIndex += 1
							theAmountWeGot += len(nextPi.buff)
							if err != nil {
								log.Println("got a reply write err:", err)
								running = false
								break
							}
							if n != len(nextPi.buff) {
								log.Println("writing len wanted, needed:", len(nextPi.buff), n)
							}
							//log.Println("So far we have got", theAmountWeGot, " of ", theLengthWeNeed, "for", packetStruct.firstLine)
							if theAmountWeGot >= theLengthWeNeed {
								// log.Println("looks like we made it ! :")
								if isDebg {
									log.Println("Request complete", packetStruct.firstLine)
								}
								running = false
							}
							responseBuffer.Flush()
						}
					}
				default:
					// no match. do nothing. panic?
					log.Println("got weird packet instead of publish ", reflect.TypeOf(packet))
					responseBuffer.Write([]byte("error got weird packet"))
					running = false
				}
			//
			case <-time.After(4567 * time.Millisecond): // sooner than nginx
				errMsg := "Error. Knotfree timed out waiting for reply (receiver offline)" // + contact.GetKey().Sig() + " " + firstLine[0:len(firstLine)-2]
				log.Println(errMsg)
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

// Copyright 2024.2026 Alan Tracey Wootton
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
