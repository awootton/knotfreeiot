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

package mainhelpers

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"time"

	"crypto/rand"

	"github.com/awootton/knotfreeiot/tokens"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

// type pinfo struct {
// 	// these are the reply buffers
// 	buff []byte
// }
// type RequestReplyStruct struct {
// 	originalRequest []byte
// 	firstLine       string
// 	replyParts      []pinfo
// }

// type myWriterType struct {
// 	packets chan packets.Interface
// 	buffer  bytes.Buffer
// }

// func (mw *myWriterType) Write(p []byte) (n int, err error) {
// 	return mw.buffer.Write(p)
// }

// func HandleHttpSubdomainRequest(w http.ResponseWriter, r *http.Request, ex *iot.Executive, subDomain string) {

// 	clen := r.ContentLength
// 	if clen > 63*1024 {
// 		fmt.Println("http packet too long ")
// 		http.Error(w, "http packet too long ", 500)
// 		return
// 	}
// 	theBody := make([]byte, clen)
// 	if clen > 0 {
// 		n, err := r.Body.Read(theBody)
// 		if err != nil || (n != int(clen)) {

// 			http.Error(w, "http content read fail ", 500)
// 			return
// 		}
// 	}
// 	isDebg := false

// 	//fmt.Println("http header ", r.Header) // it's a map with Cookie
// 	// r.RequtURI is "/"
// 	// r.URL is "/"
// 	// write the header to a buffer
// 	firstLine := r.Method + " " + r.URL.String() + " " + r.Proto + "\n"
// 	// fmt.Println("first line", firstLine[0:len(firstLine)-2])
// 	if strings.Contains(firstLine, "debg=12345678") {
// 		isDebg = true
// 		fmt.Println("first line", firstLine[0:len(firstLine)-2])
// 	}
// 	buf := new(bytes.Buffer)
// 	buf.WriteString(firstLine)
// 	for key, val := range r.Header {
// 		if key == "Cookie" {
// 			continue // don't pass the cookie
// 		}
// 		for i := 0; i < len(val); i++ {
// 			tmp := key + ": " + val[i] + "\r\n"
// 			buf.WriteString(tmp)
// 		}
// 	}
// 	buf.WriteString("\r\n")
// 	// write the body to a buffer
// 	n, err := buf.Write(theBody)
// 	if err != nil || (n != len(theBody)) {
// 		http.Error(w, "http theBody write ", 500)
// 	}

// 	// fmt.Println("http is request ", firstLine[0:len(firstLine)-2])

// 	pastWritesIndex := 0
// 	packetStruct := &RequestReplyStruct{}

// 	packetStruct.originalRequest = buf.Bytes()
// 	packetStruct.firstLine = firstLine[0 : len(firstLine)-2]

// 	// we need to make a contact
// 	// make a reply address
// 	// serialize the request
// 	// publish it.
// 	// wait for the response and put that into the w http.ResponseWriter
// 	// copy over the response headers
// 	// unsub the reply address
// 	// close the contact.

// 	packetsChan := make(chan packets.Interface, 100)
// 	contact := &iot.ContactStruct{}
// 	// hook the real writer
// 	myWriter := &myWriterType{}
// 	myWriter.packets = packetsChan
// 	myWriter.buffer = bytes.Buffer{}
// 	contact.SetWriter(myWriter)
// 	iot.AddContactStruct(contact, contact, ex.Config)
// 	go func() {
// 		for {
// 			select {
// 			case <-contact.ClosedChannel:
// 				return
// 			default:
// 				packet, err := packets.ReadPacket(&myWriter.buffer)
// 				if err != nil || packet == nil {
// 					fmt.Println("ERROR nil packet in http handler", err)
// 					return
// 				}
// 				packetsChan <- packet
// 			}
// 		}
// 	}()

// 	if isDebg {
// 		contact.LogMeVerbose = true
// 	}

// 	fmt.Println("serving subdomain ", subDomain, "  of "+r.Host+r.RequestURI, "con=", contact.GetKey().Sig())

// 	defer func() {
// 		if isDebg {
// 			fmt.Println("contact normal close", contact.GetKey().Sig())
// 		}
// 		contact.DoClose(errors.New("normal close"))
// 	}()

// 	// can we dispense with the stupid connect packet? TODO:
// 	// we could set contact.token = "dummy" and
// 	// contact.nextBillingTime = now + 100 years
// 	// and dispense with the billing?

// 	connect := packets.Connect{}
// 	connect.SetOption("token", []byte(tokens.GetImpromptuGiantToken()))
// 	if isDebg {
// 		connect.SetOption("debg", []byte("12345678"))
// 	}
// 	err = iot.PushPacketUpFromBottom(contact, &connect)
// 	if err != nil {
// 		fmt.Println("connect problems subdomain dial conn ", err)
// 		http.Error(w, err.Error(), 500)
// 		return
// 	}

// 	// Subscribe
// 	myRandomAddress := GetRandomB64String()
// 	subs := packets.Subscribe{}
// 	subs.Address.FromString(myRandomAddress)
// 	subs.Address.EnsureAddressIsBinary()
// 	//fmt.Println(" our return addr will be ", subs.Address.String())
// 	if isDebg {
// 		subs.SetOption("debg", []byte("12345678"))
// 	}
// 	err = iot.PushPacketUpFromBottom(contact, &subs)
// 	_ = err

// 	// now we have to wait for the suback to come back
// 	haveSuback := false
// 	for !haveSuback {
// 		select {
// 		case <-contact.ClosedChannel:
// 			break
// 		case packet := <-packetsChan:
// 			// see if it's a suback
// 			// fmt.Println("waiting for suback on gotDataChan.TheChan got ", cmd.Sig())
// 			if packet == nil {
// 				fmt.Println("ERROR nil packet waiting for suback. Never happens.")
// 			} else {
// 				subcmd, ok := packet.(*packets.Subscribe)
// 				if !ok {
// 					fmt.Println("ERROR wrong packet waiting for suback  ")
// 				} else {
// 					if isDebg {
// 						fmt.Println("http handler have suback  ", subcmd.Sig())
// 					}
// 					haveSuback = true
// 				}
// 			}
// 			// we have to wait for the suback to come back
// 		case <-time.After(4 * time.Second):
// 			errMsg := "timed out waiting for suback reply " + firstLine[0:len(firstLine)-2]
// 			fmt.Println(errMsg)
// 			http.Error(w, errMsg, 500)
// 			return
// 		}
// 	}

// 	if buf.Len() > 60*1024 {
// 		// stream it
// 		fmt.Println("ERROR fixme: implement this streaming thing")
// 	} else {

// 		// just send it all at once in one Send
// 		pub := packets.Send{}

// 		// copy the options over
// 		parts := strings.Split(firstLine, "?") // ie GET /get/c?debg=12345678 HTTP/1
// 		if len(parts) > 1 {
// 			parts = strings.Split(parts[1], " ")
// 			parts = strings.Split(parts[0], "&")
// 			for _, part := range parts {
// 				kv := strings.Split(part, "=")
// 				if len(kv) == 2 {
// 					pub.SetOption(kv[0], []byte(kv[1]))
// 				}
// 			}
// 		}
// 		got, ok := pub.GetOption("debg")
// 		if ok && string(got) == "12345678" {
// 			isDebg = true
// 		}

// 		pub.Address.FromString(subDomain) // !!!!!
// 		pub.Source = subs.Address
// 		//fmt.Println(" our send addr is ", pub.Address.String())
// 		pub.Address.EnsureAddressIsBinary()
// 		//fmt.Println(" our send addr is ", pub.Address.String())
// 		//fmt.Println(" our return addr is ", pub.Source.String())
// 		//pub.Payload = []byte("GET " + r.URL.String() + " HTTP/1.1\n\n")
// 		pub.Payload = buf.Bytes()

// 		// fmt.Println("publish  PushPacketUpFromBottom") // , string(pub.Payload))
// 		err = iot.PushPacketUpFromBottom(contact, &pub)
// 		_ = err
// 	}

// 	hj, ok := w.(http.Hijacker)
// 	if !ok {
// 		http.Error(w, "webserver doesn't support hijacking", http.StatusInternalServerError)
// 		return
// 	}
// 	conn, responseBuffer, err := hj.Hijack()
// 	if err != nil {
// 		fmt.Println("hijack error  ", err)
// 	}
// 	defer func() {
// 		// fmt.Println("closing hijack socket " + r.URL.String() + "\n")
// 		conn.Close()
// 	}()

// 	{ // The Receive-a-packet loop
// 		running := true
// 		//hadHeader := false
// 		theLengthWeNeed := 0
// 		theAmountWeGot := 0
// 		for running {
// 			select {
// 			case <-contact.ClosedChannel:
// 				//fmt.Println("contact closed")
// 				running = false
// 				break
// 			case packet := <-packetsChan:

// 				//fmt.Println("Receive-a-packet loop got ", cmd.Sig())
// 				_, ok := packet.(*packets.Subscribe)
// 				if ok {
// 					continue // excess subacks are expected. why?
// 				}
// 				switch v := packet.(type) {
// 				case *packets.Send:
// 					snd := v
// 					packetCountStr, ok := snd.GetOption("of")
// 					if ok {
// 						fmt.Println("packet count total= ", packetCountStr)
// 						// we have the last packet.
// 						running = false
// 						break
// 					}
// 					packetCountStr, ok = snd.GetOption("indx")
// 					if !ok {
// 						packetCountStr = []byte("0")
// 					}
// 					if packetCountStr[0] == '[' { // some idiot wrapped it in []
// 						packetCountStr = packetCountStr[1:]
// 					}
// 					if packetCountStr[len(packetCountStr)-1] == ']' {
// 						packetCountStr = packetCountStr[0 : len(packetCountStr)-1]
// 					}
// 					packetIncomingIndex, _ := strconv.Atoi(string(packetCountStr))
// 					//fmt.Println("packet count is ", packetCount)
// 					//if packetCount != packetsReceived {
// 					//	fmt.Println("we seem to have lost a PACKET:", packetCount, packetsReceived)
// 					//} pastWritesIndex
// 					// pad out the buffer
// 					for packetIncomingIndex >= len(packetStruct.replyParts) {
// 						pi := &pinfo{}
// 						packetStruct.replyParts = append(packetStruct.replyParts, *pi)
// 					}
// 					//packetStruct.replyParts[packetIncomingIndex].buff = snd.Payload
// 					currentPayload := snd.Payload

// 					// fmt.Println("have http reply packet #", packetIncomingIndex, "for ", firstLine)
// 					if packetIncomingIndex == 0 {
// 						headerEndBytes := []byte("\r\n\r\n")
// 						headerPos := bytes.Index(snd.Payload, headerEndBytes)
// 						if headerPos <= 0 {
// 							fmt.Println("no header was found in first packet")
// 						} else {
// 							// parse the header
// 							header := snd.Payload[0:headerPos]
// 							clStr := "Content-Length:"
// 							clPos := bytes.Index(header, []byte(clStr))
// 							if clPos <= 0 {
// 								fmt.Println("no Content-Length was found in first packet")
// 							}
// 							hpart := header[clPos+len(clStr):]
// 							lineEndBytes := []byte("\r\n")
// 							endPos := bytes.Index(hpart, lineEndBytes)
// 							//fmt.Println("is this a number? ", hpart[0:endPos])
// 							cldigits := string(hpart[0:endPos])
// 							i, err := strconv.Atoi(strings.Trim(cldigits, " "))
// 							if err != nil {
// 								fmt.Println("ERROR finding Content-Length", hpart[0:endPos])
// 							}
// 							// fmt.Println("theLengthWeNeed is ", i)
// 							theLengthWeNeed = i + len(header) + 4

// 							// we have to transfer the user options to the header
// 							// we insert the options onto the currentPayload
// 							// split the currentPayload into header and the rest
// 							headerStart := string(currentPayload[0:int(headerPos)]) // force a copy
// 							pastHeader := currentPayload[int(headerPos):]           // contains the \r\n\r\n, might contain some body
// 							keys, bvalues := snd.GetOptionKeys()
// 							values := make([]string, len(keys))
// 							for n := 0; n < len(keys); n++ {
// 								k := keys[n]
// 								v := bvalues[n]
// 								values[n] = string(v)
// 								// fmt.Println("Options k v ", k, values[n])
// 								_ = k
// 							}

// 							// fmt.Println("headerStart  ", string(headerStart))
// 							// fmt.Println("pastHeader  ", string(pastHeader))
// 							if len(keys) > 0 {
// 								headerStart += "\r\n"
// 								theLengthWeNeed += 2
// 							}
// 							// fmt.Println("headerStart 2 ", string(headerStart)+"\n\n")
// 							for i := 0; i < len(keys); i++ {
// 								k := keys[i]
// 								v := values[i]

// 								// fmt.Println("adding ", k, ":", string(v))
// 								headerStart += k
// 								theLengthWeNeed += len(k)
// 								//fmt.Println("headerStart 3 ", string(headerStart)+"\n\n")
// 								headerStart += ": "
// 								theLengthWeNeed += 2
// 								//fmt.Println("headerStart 4 ", string(headerStart)+"\n\n")

// 								// fmt.Println("addingvalue  ", string(values[i])+"\n\n")
// 								headerStart += v
// 								theLengthWeNeed += len(v)
// 								//fmt.Println("headerStart 5 ", string(headerStart)+"\n\n")
// 								if i < len(keys)-1 {
// 									headerStart += "\r\n"
// 									theLengthWeNeed += 2
// 								}
// 								// fmt.Println("headerStart 6 ", string(headerStart)+"\n\n")
// 							}
// 							// fmt.Println("headerStart  ", string(headerStart)+"\n\n")
// 							currentPayload = append([]byte(headerStart), pastHeader...)
// 							// fmt.Println("new payload is ", string(currentPayload)+"\n\n")
// 						}
// 					}
// 					packetStruct.replyParts[packetIncomingIndex].buff = currentPayload

// 					for { // loop over packetlist stuff we can write
// 						if pastWritesIndex >= len(packetStruct.replyParts) {
// 							break // at the end
// 						}
// 						nextPi := packetStruct.replyParts[pastWritesIndex]

// 						if isDebg {
// 							fmt.Println(contact.GetKey().Sig(), " got a reply payload packet index ", pastWritesIndex)
// 						}
// 						n, err := responseBuffer.Write(nextPi.buff)
// 						pastWritesIndex += 1
// 						theAmountWeGot += len(nextPi.buff)
// 						if err != nil {
// 							fmt.Println("got a reply write err:", err)
// 							running = false
// 							break
// 						}
// 						if n != len(nextPi.buff) {
// 							fmt.Println("writing len wanted, needed:", len(nextPi.buff), n)
// 						}
// 						//fmt.Println("So far we have got", theAmountWeGot, " of ", theLengthWeNeed, "for", packetStruct.firstLine)
// 						if theAmountWeGot >= theLengthWeNeed {
// 							// fmt.Println("looks like we made it ! :")
// 							responseBuffer.Flush()
// 							// push a close packet or something
// 							// close the connection -- below

// 							fmt.Println("Request complete", packetStruct.firstLine)
// 							running = false
// 						}
// 						//responseBuffer.Flush()
// 					}

// 				default:
// 					// no match. do nothing. panic?
// 					fmt.Println("got weird packet instead of publish ", reflect.TypeOf(packet))
// 					w.Write([]byte("error got weird packet"))
// 					running = false
// 					break
// 				}
// 			// is this the only way to know that we're done??
// 			case <-time.After(5 * time.Second):
// 				errMsg := "timed out waiting for html reply " + contact.GetKey().Sig() + " " + firstLine[0:len(firstLine)-2]
// 				fmt.Println(errMsg)
// 				// http.Error(w, errMsg, 500)
// 				running = false
// 			}
// 		}

// 		//fmt.Println("closing html write ")
// 		responseBuffer.Flush()
// 		// un sub
// 		// close the contact
// 		unsub := packets.Unsubscribe{}
// 		unsub.Address.FromString(myRandomAddress)
// 		if isDebg {
// 			unsub.SetOption("debg", []byte("12345678"))
// 		}
// 		err = iot.PushPacketUpFromBottom(contact, &unsub)
// 		_ = err
// 		contact.DoClose(errors.New("normal close"))
// 	}
// }

// Makes a tokens.Medium token which is 32 connections
func MakeMedium32cToken() (string, tokens.KnotFreeTokenPayload) {

	// 20 connections is about Medium
	// see tokens.Medium

	// caller must do this:
	// tokens.LoadPublicKeys()
	// tokens.LoadPrivateKeys("~/atw/privateKeys4.txt")

	fmt.Println("in MakeMedium32cToken")

	// tokenRequest := &tokens.TokenRequest{}
	payload := tokens.KnotFreeTokenPayload{}
	payload.KnotFreeContactStats = tokens.GetTokenStatsAndPrice(tokens.Medium).Stats

	//payload.Connections = 20 // 2 // TODO: move into standard x-small token

	// a year - standard x-small
	payload.ExpirationTime = uint32(time.Now().Unix() + 60*60*24*365)

	//payload.Input = 1024  // 32 * 4  // TODO: move into standard x-small token
	// payload.Output = 1024 // 32 * 4 // TODO: move into standard x-small token

	payload.Issuer = tokens.GetPrivateKeyPrefix(0) // "_9sh"
	payload.JWTID = tokens.GetRandomB36String()
	nonce := payload.JWTID
	_ = nonce

	//payload.Subscriptions = 20 // TODO: move into standard x-small token

	//  Host:"building_bob_bottomline_boldness.knotfree2.com:8085"
	targetSite := "knotfree.net" // "gotohere.com"
	//if os.Getenv("KNOT_KUNG_FOO") == "atw" {
	// targetSite = "gotolocal.com"
	//}
	payload.URL = targetSite

	exp := payload.ExpirationTime
	// if exp > uint32(time.Now().Unix()+60*60*24*365) {
	// 	// more than a year in the future not allowed now.
	// 	exp = uint32(time.Now().Unix() + 60*60*24*365)
	// 	fmt.Println("had long token ", string(payload.JWTID)) // TODO: store in db
	// }

	cost := tokens.GetTokenStatsAndPrice(tokens.Medium).Price * 12 //tokens.CalcTokenPrice(&payload, uint32(time.Now().Unix()))
	jsonstr, _ := json.Marshal(payload)
	fmt.Println("token cost is "+fmt.Sprintf("%f", cost), string(jsonstr))

	// large32x := ScaleTokenPayload(&payload, 8*32)
	// cost = tokens.CalcTokenPrice(large32x, uint32(time.Now().Unix()))
	// jsonstr, _ = json.Marshal(large32x)
	// fmt.Println("token cost is "+fmt.Sprintf("%f", cost), string(jsonstr))

	signingKey := tokens.GetPrivateKeyWhole(0)
	bbb, err := tokens.MakeToken(&payload, []byte(signingKey))
	if err != nil {
		fmt.Println("Make32xLargeToken ", err)
	}
	exptime := time.Unix(int64(exp), 0)
	formatted := exptime.Format("Jan/_2/2006")

	giantToken := string(bbb)
	giantToken = "[32xlarge_token,expires:" + formatted + ",token:" + giantToken + "]"
	return giantToken, payload
}

const S3_BUCKET = "gotoherestatic"

// this works.

func TrySomeS3Stuff() {

	fmt.Println("in TrySomeS3Stuff")

	dirname, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}
	credsName := dirname + "/atw/credentials"
	fmt.Println("Using", dirname)

	// the default location ~/.aws/credentials is not mapped by k8s
	// we use this path:
	os.Setenv("AWS_SHARED_CREDENTIALS_FILE", credsName) // don't work
	//os.Setenv("AWS_CONFIG_FILE", credsName)

	sess, err := session.NewSession(&aws.Config{Region: aws.String("us-east-1")})

	fmt.Println("in TrySomeS3Stuff bottom   err ", err)

	ccc, err2 := sess.Config.Credentials.Get()
	fmt.Println("in TrySomeS3Stuff ccc ", ccc, err2)

	svc := s3.New(sess)

	input := &s3.ListBucketsInput{}
	result, err := svc.ListBuckets(input)
	_ = result
	fmt.Println("in TrySomeS3Stuff get bucket list  ", err, result)

	// S3_BUCKET aka gotoherestatic will be our cache.

	// let's amke a file
	fname := "testingFile" + getRandomString() + ".txt"
	sampleText := fname + " aaa nnn lpl tttaaa nnn lpl ttt aaa nnn lpl ttt aaa nnn lpl ttt aaa nnn lpl ttt aaa nnn lpl ttt aaa nnn lpl ttt  "

	AddBytesToS3(sess, fname, []byte(sampleText))

	req, _ := svc.GetObjectRequest(&s3.GetObjectInput{
		Bucket: aws.String(S3_BUCKET),
		Key:    aws.String(fname),
	})
	urlStr, err := req.Presign(15 * time.Minute)

	if err != nil {
		log.Println("Failed to sign request", err)
	}

	log.Println("The URL is", urlStr)

	fmt.Println("in TrySomeS3Stuff bottom ", fname)
}

func getRandomString() string {
	var tmp [16]byte
	rand.Read(tmp[:])
	return base64.RawURLEncoding.EncodeToString(tmp[:])
}

// func GetRandomB64String() string {
// 	var tmp [18]byte
// 	rand.Read(tmp[:])
// 	return base64.RawURLEncoding.EncodeToString(tmp[:])
// }

func AddBytesToS3(s *session.Session, destFileName string, buffer []byte) error {

	// Config settings: this is where you choose the bucket, filename, content-type etc.
	// of the file you're uploading.
	_, err := s3.New(s).PutObject(&s3.PutObjectInput{
		Bucket:             aws.String(S3_BUCKET),
		Key:                aws.String(destFileName),
		ACL:                aws.String("private"),
		Body:               bytes.NewReader(buffer),
		ContentLength:      aws.Int64(int64(len(buffer))),
		ContentType:        aws.String(http.DetectContentType(buffer)),
		ContentDisposition: aws.String("attachment"),
		//	ServerSideEncryption: aws.String("AES256"),
	})
	return err
}

// AddFileToS3 will upload a single file to S3, it will require a pre-built aws session
// and will set file info like content type and encryption on the uploaded file.
func AddFileFileToS3(s *session.Session, fileDir string) error {

	// Open the file for use
	file, err := os.Open(fileDir)
	if err != nil {
		return err
	}
	defer file.Close()

	// Get file size and read the file content into a buffer
	fileInfo, _ := file.Stat()
	var size int64 = fileInfo.Size()
	buffer := make([]byte, size)
	file.Read(buffer)

	// Config settings: this is where you choose the bucket, filename, content-type etc.
	// of the file you're uploading.
	_, err = s3.New(s).PutObject(&s3.PutObjectInput{
		Bucket:             aws.String(S3_BUCKET),
		Key:                aws.String(fileDir),
		ACL:                aws.String("private"),
		Body:               bytes.NewReader(buffer),
		ContentLength:      aws.Int64(int64(len(buffer))),
		ContentType:        aws.String(http.DetectContentType(buffer)),
		ContentDisposition: aws.String("attachment"),
		//ServerSideEncryption: aws.String("AES256"),
	})
	return err
}

func XXX_ScaleTokenPayload(token *tokens.KnotFreeTokenPayload, scale float64) *tokens.KnotFreeTokenPayload {
	scaled := tokens.KnotFreeTokenPayload{}

	scaled.ExpirationTime = token.ExpirationTime // unix seconds
	scaled.Issuer = token.Issuer                 // first 4 bytes (or more) of base64 public key of issuer
	scaled.JWTID = token.JWTID                   // a unique serial number for this Issuer

	scaled.KnotFreeContactStats.Input = token.KnotFreeContactStats.Input                 // bytes per sec
	scaled.KnotFreeContactStats.Output = token.KnotFreeContactStats.Output               // bytes per sec
	scaled.KnotFreeContactStats.Subscriptions = token.KnotFreeContactStats.Subscriptions // seconds per sec
	scaled.KnotFreeContactStats.Connections = token.KnotFreeContactStats.Connections     // limits on what we're allowed to do.

	scaled.URL = token.URL // address of the service eg. "knotfree.net" or knotfree0.com for localhost

	// the meat:
	scaled.KnotFreeContactStats.Input *= scale
	scaled.KnotFreeContactStats.Output *= scale
	scaled.KnotFreeContactStats.Subscriptions *= scale
	scaled.KnotFreeContactStats.Connections *= scale

	scaled.KnotFreeContactStats.Input = float64(math.Floor(float64(scaled.KnotFreeContactStats.Input)))
	scaled.KnotFreeContactStats.Output = float64(math.Floor(float64(scaled.KnotFreeContactStats.Output)))
	scaled.KnotFreeContactStats.Subscriptions = float64(math.Floor(float64(scaled.KnotFreeContactStats.Subscriptions)))
	scaled.KnotFreeContactStats.Connections = float64(math.Floor(float64(scaled.KnotFreeContactStats.Connections)))

	return &scaled
}

// This was an old attempt to cache the response from the server
// now the whole original request packet is in buf

// isCachable := strings.Contains(firstLine, "/static/") || strings.Contains(firstLine, "/images/")
// haveAlready, ok := servedMap[firstLine]
// if ok && isCachable {

// 	size := 0
// 	for _, databuf := range haveAlready.replyParts {
// 		size += len(databuf.buff)
// 	}

// 	fmt.Println("serving from cache ", firstLine, size)

// 	hj, ok := w.(http.Hijacker)
// 	if !ok {
// 		http.Error(w, "webserver doesn't support hijacking", http.StatusInternalServerError)
// 		return
// 	}
// 	conn, responseBuffer, err := hj.Hijack()
// 	if err != nil {
// 		fmt.Println("hijack error  ", err)
// 	}
// 	defer func() {
// 		fmt.Println("closing hijack socket from cache " + r.URL.String() + "\n\n")
// 		conn.Close()
// 	}()

// 	for i, databuf := range haveAlready.replyParts {
// 		n, err := responseBuffer.Write(databuf.buff[:])
// 		if err != nil {
// 			fmt.Println("responseBuffer.Write ERROR ", firstLine[0:len(firstLine)-2])
// 		}
// 		_ = i
// 		_ = n
// 	}
// 	return
// }

// if isCachable {
// 	size := 0
// 	for _, databuf := range packetStruct.replyParts {
// 		size += len(databuf.buff)
// 	}
// 	if size > 1500 {
// 		fmt.Println("adding to cache ", firstLine, size)
// 		servedMap[firstLine] = packetStruct
// 	}
// }
