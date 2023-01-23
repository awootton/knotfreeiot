package main

import (
	"encoding/base64"
	"fmt"
	"math/rand"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/awootton/knotfreeiot/packets"
	"github.com/awootton/knotfreeiot/tokens"
)

// Copyright 2022 Alan Tracey Wootton
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

func main() {

	target_cluster := os.Getenv("TARGET_CLUSTER")
	fmt.Println("target_cluster", target_cluster)

	token := os.Getenv("TOKEN")
	fmt.Println("token", token)

	fmt.Println("version 3")

	serveGetTime(token)

	publishTestTopic(token)

	for {
		fmt.Println("in monitor_pod")
		time.Sleep(600 * time.Second)
	}
}

var testtopicCount = 0

func publishTestTopic(token string) { // use knotfree format

	target_cluster := os.Getenv("TARGET_CLUSTER")

	go func() {
		for { // forever
			servAddr := target_cluster + ":8384"
			tcpAddr, err := net.ResolveTCPAddr("tcp", servAddr)
			if err != nil {
				println("ResolveTCPAddr failed:", err.Error())
				fail++
				continue
			}
			// println("testtopic Dialing ")
			conn, err := net.DialTCP("tcp", nil, tcpAddr)
			if err != nil {
				println("Dial failed:", err.Error())
				time.Sleep(10 * time.Second)
				fail++
				continue
			}
			connect := &packets.Connect{}
			connect.SetOption("token", []byte(token))
			err = connect.Write(conn)
			if err != nil {
				println("testtopic Write C to server failed:", err.Error())
				conn.Close()
				time.Sleep(10 * time.Second)
				fail++
				continue
			}

			now := time.Now()
			min := strconv.Itoa(now.Minute())
			sec := strconv.Itoa(now.Second())

			// message := "time " + string(nowbytes) + " count " + strconv.FormatInt(testtopicCount, 10)
			message := min + ":" + sec + " count " + strconv.Itoa(testtopicCount)
			testtopicCount++

			//fmt.Println("testtopic connected")
			topic := "testtopic"
			sub := &packets.Send{}
			sub.Address.FromString(topic)
			sub.Payload = []byte(message)
			sub.Source.FromString("random unwatched return address")
			sub.SetOption("helloKey", []byte("worldValue"))
			sub.Write(conn)
			if err != nil {
				println("Write testtopic failed:", err.Error())
			}
			conn.Close()
			time.Sleep(10 * time.Second)
		}
	}()
}

var count = 0
var fail = 0

func serveGetTime(token string) { // use knotfree format

	target_cluster := os.Getenv("TARGET_CLUSTER")

	passPhrase := "testString123"

	pubk, privk := tokens.GetBoxKeyPairFromPassphrase(passPhrase)
	pubStr := base64.RawURLEncoding.EncodeToString(pubk[:])
	privStr := base64.RawURLEncoding.EncodeToString(privk[:])
	_ = privStr

	adminPassPhrase := "myFamousOldeSaying"

	pubk, privk = tokens.GetBoxKeyPairFromPassphrase(adminPassPhrase)
	adminPubStr := base64.RawURLEncoding.EncodeToString(pubk[:])
	adminPrivStr := base64.RawURLEncoding.EncodeToString(privk[:])
	_ = adminPrivStr

	adminPassPhrase2 := "myFamousOldeSaying2"

	pubk, privk = tokens.GetBoxKeyPairFromPassphrase(adminPassPhrase2)
	adminPubStr2 := base64.RawURLEncoding.EncodeToString(pubk[:])
	adminPrivStr2 := base64.RawURLEncoding.EncodeToString(privk[:])
	_ = adminPrivStr2

	dummyString := "none"

	go func() {
		for { // forever
			servAddr := target_cluster + ":8384"
			tcpAddr, err := net.ResolveTCPAddr("tcp", servAddr)
			if err != nil {
				println("ResolveTCPAddr failed:", err.Error())
				fail++
				time.Sleep(10 * time.Second)
				continue
			}
			println("Dialing ")
			conn, err := net.DialTCP("tcp", nil, tcpAddr)
			if err != nil {
				println("Dial failed:", err.Error())
				time.Sleep(10 * time.Second)
				fail++
				continue
			}
			connect := &packets.Connect{}
			connect.SetOption("token", []byte(token))
			err = connect.Write(conn)
			if err != nil {
				println("Write C to server failed:", err.Error())
				conn.Close()
				time.Sleep(10 * time.Second)
				fail++
				continue
			}
			topic := "get-unix-time"
			sub := &packets.Subscribe{}
			sub.Address.FromString(topic)
			sub.Write(conn)
			if err != nil {
				println("Write topic failed:", err.Error())
				conn.Close()
				time.Sleep(10 * time.Second)
				fail++
				continue
			}
			fmt.Println("connected and subscribed and waiting..")
			// receive cmd and respond loop
			for {
				p, err := packets.ReadPacket(conn) // blocks
				if err != nil {
					println("client err:", err.Error())
					conn.Close()
					fail++
					time.Sleep(10 * time.Second)
					break
				}
				println("received:", p.String())
				pub, ok := p.(*packets.Send)
				if !ok {
					println("expected a send aka publish:", p.String())
					fail++
					// time.Sleep(10 * time.Second)
					continue
				}
				//fmt.Println("to ", string(pub.Address.String()))
				//fmt.Println("from ", string(pub.Source.String()))

				message := string(pub.Payload)
				// n, _ := pub.GetOption("nonce")
				// println("get-unix-time got:", message, string(n))

				isHttp := false
				if strings.HasPrefix(message, `GET /`) {
					isHttp = true
					lines := strings.Split(message, "\n")
					if len(lines) < 1 {
						fail++
						break
					}
					getline := lines[0]
					getparts := strings.Split(getline, " ")
					if len(getparts) != 3 {
						fail++
						continue
					}
					// now we passed the headers
					message = getparts[1]
					message = strings.ReplaceAll(message, "/", " ")
					message = strings.Trim(message, " ")
					fmt.Println("http command is ", message)
				}

				reply := ""
				needsEncryption := false
				if message == `get time` {
					sec := time.Now().UnixMilli() / 1000
					secStr := strconv.FormatInt(sec, 10)
					reply = secStr
				} else if message == `get random` {
					tmp := rand.Uint32()
					secStr := strconv.FormatInt(int64(tmp), 10)
					reply = secStr
				} else if message == `get count` {
					countStr := strconv.FormatInt(int64(count), 10)
					reply = countStr
					needsEncryption = true
				} else if message == `get fail` {
					countStr := strconv.FormatInt(int64(fail), 10)
					reply = countStr
					needsEncryption = true
				} else if message == `version` {
					reply = "v0.1.2"
				} else if message == "get pubk" {
					reply = pubStr
				} else if message == "get admin hint" {
					reply = adminPubStr[0:8] + " " + adminPubStr2[0:8]
				} else if message == "get dummy" {
					reply = dummyString
				} else if strings.HasPrefix(message, "set dummy") {
					s := message[len("set dummy"):]
					s = strings.Trim(s, " ")
					dummyString = s
					fmt.Println("dummy is set to ", s)
					reply = "ok"
				} else {
					reply += "[get time] unix time in seconds\n"
					reply += "[get count] how many served since reboot\n"
					reply += "[get fail] how many requests were bad since reboot\n"
					reply += "[get pubk] device public key\n"
					reply += "[get admin hint] the first chars of the admin public keys\n"
					reply += "[get dummy] an unused string\n"
					reply += "[set dummy] +1 set useless string\n"
					reply += "[get random] returns a random integer\n"
					reply += "[version] info on this device\n"
					reply += "[help] lists all commands\n"
				}
				if isHttp {

					tmp := "HTTP/1.1 200 OK\r\n"
					tmp += "Content-Length: "
					tmp += strconv.FormatInt(int64(len(reply)), 10)
					tmp += "\r\n"
					tmp += "Content-Type: text/plain\r\n"
					tmp += "Connection: Closed\r\n"
					tmp += "\r\n"
					tmp += reply
					reply = tmp
				}

				_ = needsEncryption
				sendme := &packets.Send{}
				sendme.Address = pub.Source
				sendme.Source = pub.Address
				sendme.Payload = []byte(reply)
				sendme.CopyOptions(&pub.PacketCommon) // this is very important. there's a nonce in here

				// println("get-unix-time reply to :", string(sendme.Address.String()))
				// fmt.Println("destination ", string(sendme.Address.String()))
				//  fmt.Println("source ", string(sendme.Source.String()))
				err = sendme.Write(conn)
				if err != nil {
					println("send err:", err)
					fail++
					break
				}
				count++
			}
		}
	}()
}
