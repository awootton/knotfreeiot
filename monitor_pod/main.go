package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/awootton/knotfreeiot/packets"
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

	serveTime(token)

	publistTestTopic(token)

	for {
		fmt.Println("in monitor_pod")
		time.Sleep(600 * time.Second)
	}
}

var testtopicCount = 0

func publistTestTopic(token string) { // use knotfree format

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
			println("testtopic Dialing ")
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

			fmt.Println("testtopic connected")
			topic := "testtopic"
			sub := &packets.Send{}
			sub.Address.FromString(topic)
			sub.Payload = []byte(message)
			sub.Source.FromString("random unwatched return address")
			sub.SetOption("hello", []byte("world"))
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

func serveTime(token string) { // use knotfree format

	target_cluster := os.Getenv("TARGET_CLUSTER")

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
				p, err := packets.ReadPacket(conn) // this better block
				if err != nil {
					println("client err:", err.Error())
					conn.Close()
					fail++
					time.Sleep(10 * time.Second)
					break
				}
				// println("received:", p.String())
				pub, ok := p.(*packets.Send)
				if !ok {
					println("expected a send aka publish:", p.String())
					fail++
					time.Sleep(10 * time.Second)
					break
				}
				//fmt.Println("to ", string(pub.Address.String()))
				//fmt.Println("from ", string(pub.Source.String()))

				message := string(pub.Payload)
				//println("client got:", message)
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
				if message == `get time` {
					sec := time.Now().UnixMilli() / 1000
					secStr := strconv.FormatInt(sec, 10)
					reply = secStr
				} else if message == `get count` {
					countStr := strconv.FormatInt(int64(count), 10)
					reply = countStr
				} else if message == `get fail` {
					countStr := strconv.FormatInt(int64(fail), 10)
					reply = countStr
				} else {
					reply += "[get time] returns the unix time in seconds\n"
					reply += "[get count] returns how many served since reboot\n"
					reply += "[get fail] returns how requests were bad since reboot\n"
					reply += "[help] returns this message\n"
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
				sendme := &packets.Send{}
				sendme.Address = pub.Source
				sendme.Source = pub.Address
				sendme.Payload = []byte(reply)

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

// delete me
func XXXserveTimeTextMode(token string) { // fails http and is awkward with help. works otherwise

	// using the text protocol on port 7465

	target_cluster := os.Getenv("TARGET_CLUSTER")

	go func() {
		for {
			servAddr := target_cluster + ":7465"
			tcpAddr, err := net.ResolveTCPAddr("tcp", servAddr)
			if err != nil {
				println("serveTime ResolveTCPAddr failed:", err.Error())
				continue
			}
			println("serveTime Dialing ")
			conn, err := net.DialTCP("tcp", nil, tcpAddr)
			if err != nil {
				println("serveTime Dial failed:", err.Error())
				continue
			}

			str := "C token " + token
			_, err = conn.Write([]byte(str + "\n"))
			if err != nil {
				println("serveTime C Write to server failed:", err.Error())
				continue
			}

			topic := "get-unix-time"
			str = "S " + topic
			_, err = conn.Write([]byte(str + "\n"))
			if err != nil {
				println("serveTime Write S to server failed:", err.Error())
				continue
			}

			println("serveTime subscribed, waiting for line... ")

			lineReader := bufio.NewReader(conn)
			for {
				str, err := lineReader.ReadString('\n')
				if err != nil {
					println("serveTime ReadString err:", err.Error())
					break
					// start over
				}
				if len(str) < 3 {
					continue
				}
				println("serveTime got:", str)
				// eg. [P,=xOZPbNiNsA_lM_6xJEwM1C7YmVMGlDpA,myaddresstopicchannel,"get time"]\n
				// or  [P,=xOZPbNiNsA_lM_6xJEwM1C7YmVMGlDpA,=NSIrmrHdo37keWaU1RP4MikldE4B-Vga,"GET /get/time HTTP/1.1
				// it's going  to come through without spaces

				if strings.HasPrefix(str, "[") {
					str = str[1:]
				} else {
					continue
				}
				if strings.HasSuffix(str, "]\n") {
					str = str[0 : len(str)-2]
				} else {
					//continue
				}
				parts := strings.Split(str, ",")
				fmt.Println(parts) // eg. [P,=xOZPbNiNsA_lM_6xJEwM1C7YmVMGlDpA,=NSIrmrHdo37keWaU1RP4MikldE4B-Vga,"GET /get/time HTTP/1.1
				if len(parts) != 4 {
					continue
				}
				isHttp := false
				if parts[0] != "P" { // it has to be a publish
					continue
				}
				ourAddress := parts[1]    // hashed by system : will always be =xOZPbNiNsA_lM_6xJEwM1C7YmVMGlDpA
				returnAddress := parts[2] // eg, in example, myaddresstopicchannel
				message := parts[3]       // eg, "get time" or "GET /get/time HTTP/1.1

				if strings.HasPrefix(message, `"GET /`) {
					isHttp = true
					for {
						headerLine, err := lineReader.ReadString('\n')
						if err != nil {
							println("serveTime headerLine err:", err.Error())
							break
						}
						if headerLine == "\n" {
							break
						}
					}
					getparts := strings.Split(message, " ")
					if len(getparts) != 3 {
						continue
					}
					// now we passed the headers
					message = getparts[1]
					message = strings.ReplaceAll(message, "/", " ")
					message = strings.Trim(message, " ")
					fmt.Println("http command is ", message)
				} else {
					// message is ok
					message = strings.Trim(message, `"`)
				}
				reply := "[get time] returns the unix time in seconds\n"
				reply += "[get count] returns how many served since reboot\n"
				reply += "[get fail] returns how requests were bad since reboot\n"
				reply += "[help] returns this message\n"
				if message == `get time` {
					sec := time.Now().UnixMilli() / 1000
					secStr := strconv.FormatInt(sec, 10)
					reply = secStr
				} else if message == `get count` {
					countStr := strconv.FormatInt(int64(count), 10)
					reply = countStr
				}
				_ = isHttp
				// make a reply
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
				lines := strings.Split(reply, "\n")
				for _, line := range lines {
					replyStr := "P " + returnAddress + " " + ourAddress + ` "` + line + `"\n`
					fmt.Println("get time reply: ", replyStr)
					_, err = conn.Write([]byte(replyStr + "\n"))
					if err != nil {
						println("serveTime Write err:", err.Error())
						break
					}
				}
				count++
			}
		}
	}()
}
