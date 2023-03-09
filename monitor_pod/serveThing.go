package monitor_pod

import (
	"encoding/base64"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"net"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/awootton/knotfreeiot/packets"
	"github.com/awootton/knotfreeiot/tokens"
	"golang.org/x/crypto/nacl/box"
)

// example:
// [erase eeprom] +1 erase all the settings with code KILLMENOW
// [favicon.ico] shortest png in the world
// [freemem] count of free memory
// [get admin hint] +1 first bytes from any admin keys we accept
// [get box time] how many ms to box a message
// [get local peers] list http servers on local net.
// [get long name] long name is unique over the world.
// [get pass] WiFi password
// [get pubk] +1 public key of this thing
// [get short name] short name aka hostname. Name on local net.
// [get ssid] WiFi ssid
// [get ssid list] list of local wifi nets
// [get token] shows ** if you have a token.
// [help] the description of every command
// [served] count of requests served since reboot
// [set long name] +1 set long name is unique over the world
// [set pass] +1 set WiFi pass
// [set short name] +1 set short name aka hostname. This will be the 'local.' name.
// [set ssid] +1 set WiFi ssid
// [set token] +1 set access token
// [status] WiFi status
// [uptime] time since last reboot.
// [version] mqtt5nano version

type FauxContext struct {
	Topic string

	password string
	pubStr   string
	privStr  string

	//adminPassPhrase string
	adminPubStr string
	//adminPrivStr    string

	//adminPassPhrase2 string
	adminPubStr2 string
	//adminPrivStr2    string

	dummyString string

	fail  int
	count int

	CommandMap map[string]Command
	Index      int
	Token      string
}

var tempInF = 46.0

func StartTempGetter() {
	go func() {
		for {
			tempInF = 46.0 // fixme get real temp
			time.Sleep(15 * time.Minute)
		}
	}()
}

type Command struct {
	execute       func(msg string, args []string) string
	commandString string
	description   string
	argCount      int
}

func MakeCommand(commandString string, description string,
	argCount int, execute func(msg string, args []string) string,
	ourMap map[string]Command) Command {
	cmd := Command{
		commandString: commandString,
		description:   description,
		argCount:      argCount,
		execute:       execute,
	}
	ourMap[commandString] = cmd
	return cmd
}

func ServeGetTime(token string, c FauxContext) { // use knotfree format

	target_cluster := os.Getenv("TARGET_CLUSTER")

	c.count = 0
	c.fail = 0

	c.password = "testString123"
	pubk, privk := tokens.GetBoxKeyPairFromPassphrase(c.password)
	c.pubStr = base64.RawURLEncoding.EncodeToString(pubk[:])
	c.privStr = base64.RawURLEncoding.EncodeToString(privk[:])

	adminPassPhrase := "myFamousOldeSaying"
	pubk, privk = tokens.GetBoxKeyPairFromPassphrase(adminPassPhrase)
	c.adminPubStr = base64.RawURLEncoding.EncodeToString(pubk[:])
	//c.adminPrivStr = base64.RawURLEncoding.EncodeToString(privk[:])

	adminPassPhrase2 := "myFamousOldeSaying2"
	pubk, privk = tokens.GetBoxKeyPairFromPassphrase(adminPassPhrase2)
	c.adminPubStr2 = base64.RawURLEncoding.EncodeToString(pubk[:])
	//c.adminPrivStr2 = base64.RawURLEncoding.EncodeToString(privk[:])

	c.dummyString = "none"

	MakeCommand("get time",
		"seconds since 1970🔓", 0,
		func(msg string, args []string) string {
			sec := time.Now().UnixMilli() / 1000
			secStr := strconv.FormatInt(sec, 10)
			return secStr
		}, c.CommandMap)
	MakeCommand("get c",
		"temperature in C🔓", 0,
		func(msg string, args []string) string {
			tmp := (tempInF - 32) * 5 / 9
			tmp = math.Floor(tmp*100) / 100.0
			str := strconv.FormatFloat(tmp, 'f', 2, 64)
			return str + "°C"
		}, c.CommandMap)
	MakeCommand("get f",
		"temperature in F🔓", 0,
		func(msg string, args []string) string {
			str := strconv.FormatFloat(tempInF, 'f', 2, 64)
			return str + "°F"
		}, c.CommandMap)
	MakeCommand("get random",
		"returns a random integer", 0,
		func(msg string, args []string) string {
			tmp := rand.Uint32()
			secStr := strconv.FormatInt(int64(tmp), 10)
			return secStr
		}, c.CommandMap)
	MakeCommand("get count",
		"how many served since reboot", 0,
		func(msg string, args []string) string {
			countStr := strconv.FormatInt(int64(c.count), 10)
			return countStr
		}, c.CommandMap)

	MakeCommand("get fail",
		"how many requests were bad since reboot", 0,
		func(msg string, args []string) string {
			secStr := strconv.FormatInt(int64(c.fail), 10)
			return secStr
		}, c.CommandMap)

	MakeCommand("get pubk",
		"device public key 🔓", 0,
		func(msg string, args []string) string {
			return c.pubStr
		}, c.CommandMap)
	MakeCommand("get admin hint",
		"the first chars of the admin public keys🔓", 0,
		func(msg string, args []string) string {
			return c.adminPubStr[0:8] + " " + c.adminPubStr2[0:8]
		}, c.CommandMap)

	MakeCommand("get short name",
		"the local name🔓", 0,
		func(msg string, args []string) string {
			return "time"
		}, c.CommandMap)
	MakeCommand("get long name",
		"the global name", 0,
		func(msg string, args []string) string {
			return c.Topic
		}, c.CommandMap)
	MakeCommand("favicon.ico",
		"", 0,
		func(msg string, args []string) string {
			return string(GreenSquare)
		}, c.CommandMap)
	MakeCommand("get some text",
		"return the saved text", 0,
		func(msg string, args []string) string {
			return c.dummyString
		}, c.CommandMap)
	MakeCommand("set some text",
		"save some text", 1,
		func(msg string, args []string) string {
			s := msg[len("set some text"):]
			s = strings.Trim(s, " ")
			c.dummyString = s
			fmt.Println("dummy is set to ", s)
			return "ok"
		}, c.CommandMap)
	MakeCommand("version",
		"info about this thing", 0,
		func(msg string, args []string) string {
			return "v0.1.5"
		}, c.CommandMap)
	MakeCommand("help",
		"lists all commands. 🔓 means no encryption required", 0,
		func(msg string, args []string) string {
			s := ""
			keys := make([]string, 0, len(c.CommandMap)) //  maps.Keys(c.CommandMap)
			for k := range c.CommandMap {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				command := c.CommandMap[k]
				argCount := ""
				if command.argCount > 0 {
					argCount = " +" + strconv.FormatInt(int64(command.argCount), 10)
				}
				s += "[" + k + "]" + argCount + " " + command.description + "\n"
			}
			return s
		}, c.CommandMap)

	go func() {

		for { // forever
			servAddr := target_cluster + ":8384"
			tcpAddr, err := net.ResolveTCPAddr("tcp", servAddr)
			if err != nil {
				println("ResolveTCPAddr failed:", err.Error())
				c.fail++
				time.Sleep(10 * time.Second)
				continue
			}
			println("Dialing ")
			conn, err := net.DialTCP("tcp", nil, tcpAddr)
			if err != nil {
				println("Dial failed:", err.Error())
				time.Sleep(10 * time.Second)
				c.fail++
				continue
			}
			connect := &packets.Connect{}
			connect.SetOption("token", []byte(token))
			err = connect.Write(conn)
			if err != nil {
				println("Write C to server failed:", err.Error())
				conn.Close()
				time.Sleep(10 * time.Second)
				c.fail++
				continue
			}
			/// c.topic = "get-unix-time"
			subscribeCount := 0
			go func() {
				// contact expiration time is 20 min.
				// resubscribe every 15 min to keep alive.
				for {
					if subscribeCount > 0 {
						println("reSubscribing:" + c.Topic)
					} else {
						println("Subscribing:" + c.Topic)
					}

					sub := &packets.Subscribe{}
					sub.Address.FromString(c.Topic)
					sub.Write(conn)
					if err != nil {
						println("Write topic failed:"+c.Topic, err.Error())
						// conn.Close() // if it fails here it will also fail below and reset.
						interval := time.Duration(100 + int(rand.Float32()*100))
						time.Sleep(interval * time.Second)
						c.fail++
						break // go to top?
					}
					time.Sleep(15*time.Minute + time.Second*time.Duration(4*rand.Float32()))
				}
			}()

			fmt.Println("connected and subscribed and waiting..")
			// receive cmd and respond loop
			for {
				p, err := packets.ReadPacket(conn) // blocks
				if err != nil {
					println("client err:", err.Error())
					conn.Close()
					c.fail++
					time.Sleep(10 * time.Second)
					break
				}

				sendme, err := digestPacket(p, &c, c.CommandMap)
				if err != nil {
					break // continue
				}
				err = sendme.Write(conn)
				if err != nil {
					println("send err:", err)
					c.fail++
					break
				}
				c.count++
			}
		}
	}()
}

func digestPacket(p packets.Interface,
	c *FauxContext,
	CommandMap map[string]Command) (packets.Interface, error) {

	// println("received:", p.String())
	pub, ok := p.(*packets.Send)
	if !ok {
		println("expected a send aka publish:", p.String())
		c.fail++
		// time.Sleep(10 * time.Second)
		return nil, errors.New("expected a send aka publish")
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
			c.fail++
			return nil, errors.New("bad http request")
		}
		getline := lines[0]
		getparts := strings.Split(getline, " ")
		if len(getparts) != 3 {
			c.fail++
			return nil, errors.New("expected 3 parts to http request")
		}
		// now we passed the headers
		message = getparts[1]

		mparts := strings.Split(message, "?")
		if len(mparts) > 1 {
			argparts := strings.Split(mparts[1], "&")
			for _, arg := range argparts {
				argparts2 := strings.Split(arg, "=")
				if len(argparts2) != 2 {
					c.fail++
					return nil, errors.New("expected 2 parts to arg")
				}
				argname := argparts2[0]
				argvalue := argparts2[1]
				tmp := make([]byte, len(argvalue))
				copy(tmp, argvalue)
				//fmt.Println("arg and val is ", argname, string(tmp))
				pub.SetOption(argname, []byte(argvalue)) // todo: copy inside of setoption
			}
		}
		pub.SetOption("monitorpod", []byte("rocks"))
		message = mparts[0]
		message = strings.ReplaceAll(message, "/", " ")
		message = strings.Trim(message, " ")
		fmt.Println("http command is ", message)
	}

	reply := ""
	hadEncryption := false
	hadError := ""

	if strings.HasPrefix(message, "=") { // it is base64 encoded ie encrypted
		emessage := message[1:]
		nonc, ok := pub.GetOption("nonc")
		admn, ok2 := pub.GetOption("admn")
		if nonc == nil || !ok || admn == nil || !ok2 {
			hadError = "no nonce or no admn"
			c.fail++
		} else {

			messageBytes, err := base64.RawURLEncoding.DecodeString(emessage)
			if err != nil {
				hadError = err.Error()
			}

			adminPublic := "none"
			if strings.HasPrefix(c.adminPubStr, string(admn)) {
				adminPublic = c.adminPubStr
			} else if strings.HasPrefix(c.adminPubStr2, string(admn)) {
				adminPublic = c.adminPubStr2
			} else {
				hadError = "no matching admin key found"
				c.fail++
			}

			adminPublicBytes := new([32]byte)
			adminPublicBytesTmp, err := base64.RawURLEncoding.DecodeString(adminPublic)
			if err != nil || len(adminPublicBytesTmp) != 32 {
				hadError = err.Error()
			} else {
				copy(adminPublicBytes[:], adminPublicBytesTmp[:])
			}

			devicePrivateKey := new([32]byte)
			devicePrivateKeyTmp, err := base64.RawURLEncoding.DecodeString(c.privStr)
			if err != nil || len(devicePrivateKeyTmp) != 32 {
				hadError = err.Error()
			} else {
				copy(devicePrivateKey[:], devicePrivateKeyTmp[:])
			}
			nonce := new([24]byte)
			copy(nonce[:], nonc[:])
			openbuffer := make([]byte, 0, (len(messageBytes))) // - box.Overhead
			opened, ok := box.Open(openbuffer, messageBytes, nonce, adminPublicBytes, devicePrivateKey)
			if !ok {
				hadError = "failed to decrypt"
				c.fail++
			} else {
				message = string(opened)
				mparts := strings.Split(message, "#")
				if len(mparts) > 1 {
					timestamp, err := strconv.ParseInt(mparts[1], 10, 64)
					if err != nil {
						hadError = "bad timestamp"
						c.fail++
					} else {
						now := time.Now().Unix()
						diff := now - timestamp
						if diff < 0 {
							diff = -diff
						}
						if diff > 30 {
							hadError = "timestamp too old"
							c.fail++
						}
					}
					message = mparts[0]
					message = strings.ReplaceAll(message, "/", " ")
					fmt.Println("decrypted command is ", message)

				} else {
					hadError = "missing timestamp"
					c.fail++
				}
				hadEncryption = true
			}
		}
	}

	cmd, ok := c.CommandMap["help"]
	_ = ok

	if hadError != "" {
		reply = "Error: " + hadError
	} else {

		args := make([]string, 0, 10)

		// this doesn't work right with command with args
		// like 'set some text abc'
		cmd, ok = c.CommandMap[message]
		if !ok { // try harder
			ok = false
			for k, v := range c.CommandMap {
				if strings.HasPrefix(message, k) {
					cmd = v
					ok = true
					break
				}
			}
		}
		if !ok {
			cmd = c.CommandMap["help"]
		}
		if strings.Contains(cmd.description, "🔓") {
			reply = cmd.execute(message, args)
		} else {
			if !hadEncryption {
				reply = "Error: this command requires encryption"
				c.fail++
			} else {
				reply = cmd.execute(message, args)
			}
		}
	}
	nonc, ok := pub.GetOption("nonc")
	if nonc == nil || !ok {
		hadError = "Error: no nonce"
	}

	if hadError == "" && !strings.Contains(cmd.description, "🔓") {
		// encrypt the reply

		admn, ok2 := pub.GetOption("admn")
		if admn == nil || !ok2 {
			hadError = "Error: no admn"
		}

		nonce := new([24]byte)
		copy(nonce[:], nonc[:])

		boxout := make([]byte, len(reply)+box.Overhead+99)
		boxout = boxout[:0]
		//use same nonce that was used for the message and is in the packet user args

		adminPublic := "none"
		if strings.HasPrefix(c.adminPubStr, string(admn)) {
			adminPublic = c.adminPubStr
		} else if strings.HasPrefix(c.adminPubStr2, string(admn)) {
			adminPublic = c.adminPubStr2
		} else {
			hadError = "no matching admin key found"
			c.fail++
		}

		adminPublicBytes := new([32]byte)
		adminPublicBytesTmp, err := base64.RawURLEncoding.DecodeString(adminPublic)
		if err != nil || len(adminPublicBytesTmp) != 32 {
			hadError = err.Error()
		} else {
			copy(adminPublicBytes[:], adminPublicBytesTmp[:])
		}

		devicePrivateKey := new([32]byte)
		devicePrivateKeyTmp, err := base64.RawURLEncoding.DecodeString(c.privStr)
		if err != nil || len(devicePrivateKeyTmp) != 32 {
			hadError = err.Error()
		} else {
			copy(devicePrivateKey[:], devicePrivateKeyTmp[:])
		}

		reply = reply + "#" + strconv.FormatInt(time.Now().Unix(), 10)

		sealed := box.Seal(boxout, []byte(reply), nonce, adminPublicBytes, devicePrivateKey)

		reply = "=" + base64.RawURLEncoding.EncodeToString(sealed)
		if hadError != "" {
			reply = "Error: " + hadError
		}

		fmt.Println("encrypted reply is ", reply, "nonce", string(nonc))
	}

	if isHttp {

		tmp := "HTTP/1.1 200 OK\r\n"
		tmp += "Content-Length: "
		tmp += strconv.FormatInt(int64(len(reply)), 10)
		tmp += "\r\n"
		tmp += "Content-Type: text/plain\r\n"
		tmp += "Access-Control-Allow-Origin: *\r\n"
		tmp += "access-control-expose-headers: nonc\r\n"
		tmp += "Connection: Closed\r\n"
		//tmp += "nonc: " + string(nonc) + "\r\n" // this might be redundant
		tmp += "\r\n"
		tmp += reply
		reply = tmp
	}

	sendme := &packets.Send{}
	sendme.Address = pub.Source
	sendme.Source = pub.Address
	sendme.Payload = []byte(reply)
	sendme.CopyOptions(&pub.PacketCommon) // this is very important. there's a nonce in here
	return sendme, nil
}

var testtopicCount = 0

func PublishTestTopic(token string) { // use knotfree format

	target_cluster := os.Getenv("TARGET_CLUSTER")

	fail := 0

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
				fail++
			}
			conn.Close()
			time.Sleep(10 * time.Second)
		}
	}()
}

// it's a green square PNG.
var GreenSquare = []byte{137, 80, 78, 71, 13, 10, 26, 10, 0, 0, 0, 13, 73, 72, 68,
	82, 0, 0, 0, 16, 0, 0, 0, 16, 8, 6, 0, 0, 0, 31, 243,
	255, 97, 0, 0, 0, 26, 73, 68, 65, 84, 120, 218, 99, 84, 106, 209,
	255, 207, 64, 1, 96, 28, 53, 96, 212, 128, 81, 3, 134, 139, 1, 0,
	239, 170, 29, 81, 139, 188, 27, 125, 0, 0, 0, 0, 73, 69, 78, 68,
	174, 66, 96, 130}

// Copyright 2022,2023 Alan Tracey Wootton
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
