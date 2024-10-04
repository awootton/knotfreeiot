package monitor_pod

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/awootton/knotfreeiot/packets"
	"github.com/awootton/knotfreeiot/tokens"
	"golang.org/x/crypto/nacl/box"
)

// example of commands:
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

type ThingContext struct {
	Topic string

	Password string
	PubStr   string
	PrivStr  string

	Host string //host and port. eg knotfree.net:8384

	//adminPassPhrase string
	AdminPubStr  string
	AdminPrivStr string

	//adminPassPhrase2 string
	AdminPubStr2  string
	AdminPrivStr2 string

	dummyString string

	fail  int
	count int

	CommandMap map[string]Command
	Index      int
	Token      string

	LogMeVerbose bool // a debugging thing
}

var TempInF = 46.0
var Humidity = 10.0

type Command struct {
	Execute       func(msg string, args []string, callContext interface{}) string
	CommandString string
	Description   string
	ArgCount      int
}

func MakeCommand(commandString string, description string, argCount int,
	execute func(msg string, args []string, callContext interface{}) string,
	ourMap map[string]Command) Command {
	cmd := Command{
		CommandString: commandString,
		Description:   description,
		ArgCount:      argCount,
		Execute:       execute,
	}
	ourMap[commandString] = cmd
	return cmd
}

func ServeGetTime(token string, c *ThingContext) { // use knotfree format

	//target_cluster := os.Getenv("TARGET_CLUSTER")

	c.count = 0
	c.fail = 0

	setupCommands(c)

	// var wg sync.WaitGroup
	// wg.Add(1)
	waitCount := 1

	go func() {

		connectCount := 0

		for { // connect loop forever

			servAddr := c.Host // target_cluster + ":8384"
			tcpAddr, err := net.ResolveTCPAddr("tcp", servAddr)
			if err != nil {
				println("had ResolveTCPAddr failed:", c.Topic, err.Error())
				c.fail++
				time.Sleep(10 * time.Second)
				continue // to connect loop
			}
			println("ServeGetTime Dialing ", c.Topic)
			conn, err := net.DialTCP("tcp", nil, tcpAddr)
			if err != nil {
				println("dial failed:", err.Error())
				time.Sleep(10 * time.Second)
				c.fail++
				continue // to connect loop
			}
			connect := &packets.Connect{}
			connect.SetOption("token", []byte(token))
			if c.LogMeVerbose {
				connect.SetOption("debg", []byte("12345678"))
			}
			err = connect.Write(conn)
			if err != nil {
				println("write C to server failed:", c.Topic, err.Error())
				conn.Close()
				time.Sleep(10 * time.Second)
				c.fail++
				continue // to connect loop
			}

			quitSubscribeLoop := make(chan interface{})

			go func() {
				// expiration time is 20 min.
				// resubscribe every 14+4 min to keep alive.
				for {

					println("monitor Subscribing:" + c.Topic)

					err := Subscribe(c, conn) // how do we know the conn is any good?

					if err != nil {
						println("subscribing ERROR:"+c.Topic, err)
						conn.Close() // we should get a ReadPacket error asap
						c.fail++
					}
					select {
					case <-time.After(14*time.Minute + time.Minute*time.Duration(4*rand.Float32())):
						println("subscribing resubscribe timeout:" + c.Topic)
					case res := <-quitSubscribeLoop:
						_ = res
						return
					}
				}
			}()

			fmt.Println("connected and subscribed and waiting..", c.Topic)

			for { // read cmd and respond loop
				p, err := packets.ReadPacket(conn) // blocks
				if err != nil {
					println("ReadPacket client err:", c.Topic, err.Error())
					conn.Close()
					c.fail++
					quitSubscribeLoop <- true
					time.Sleep(10 * time.Second)
					break // from read loop
				}
				if _, ok := p.(*packets.Subscribe); ok {
					// this is the suback and is normal
					fmt.Println("monitor has suback", c.Topic, p.Sig())
					waitCount = 0
					continue
				}
				sendme, err := digestPacket(p, c)
				if err != nil {
					println("digestPacket err:", c.Topic, err)
					conn.Close()
					c.fail++
					quitSubscribeLoop <- true
					break // from read loop
				}

				pub, ok := sendme.(*packets.Send)
				if ok {
					SpecialPrint(&pub.PacketCommon, func() {
						fmt.Println("serveThing reply ", c.Topic, strings.Split(string(pub.Payload), "\n")[0])
					})
				}

				err = sendme.Write(conn)
				if err != nil {
					println("send err:", c.Topic, err)
					conn.Close()
					c.fail++
					quitSubscribeLoop <- true
					break // from read loop
				}
				c.count++
			} // read loop
			connectCount++
		} // connect loop
	}()

	// wg.Wait() // return after we have suback.
	for waitCount > 0 {
		time.Sleep(10 * time.Millisecond)
	}
	fmt.Println("serveThing started", c.Topic)
}

func Subscribe(c *ThingContext, conn net.Conn) error {
	sub := &packets.Subscribe{}
	sub.Address.FromString(c.Topic)
	if c.LogMeVerbose {
		sub.SetOption("debg", []byte("12345678"))
	}
	err := sub.Write(conn)
	if err != nil {
		println("write topic subscribe failed:"+c.Topic, err.Error())
		// conn.Close() // if it fails here it will also fail below and reset.
		interval := time.Duration(100 + int(rand.Float32()*100))
		time.Sleep(interval * time.Second)
		c.fail++
		return err // go to top?
	}
	return nil
}

func digestPacket(p packets.Interface, c *ThingContext) (packets.Interface, error) {

	// println("received:", p.String())
	pub, ok := p.(*packets.Send)
	if !ok {
		println("expected a send aka publish:", c.Topic, p.Sig())
		c.fail++
		time.Sleep(1 * time.Second)
		return nil, errors.New("expected a send aka publish" + c.Topic)
	}

	message := string(pub.Payload)

	SpecialPrint(&pub.PacketCommon, func() {
		fmt.Print("monitor ", c.Topic, " got ", pub.Sig())
	})

	isHttp := false
	if strings.HasPrefix(message, `GET /`) {
		isHttp = true
		lines := strings.Split(message, "\n")
		if len(lines) < 1 {
			c.fail++
			return nil, errors.New("bad http request" + c.Topic)
		}
		getline := lines[0]
		getparts := strings.Split(getline, " ")
		if len(getparts) != 3 {
			c.fail++
			return nil, errors.New("expected 3 parts to http request " + c.Topic)
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
					return nil, errors.New("expected 2 parts to arg " + c.Topic + " " + arg + "")
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
		SpecialPrint(&pub.PacketCommon, func() {
			fmt.Println("http command is ", strings.Split(message, "\n")[0])
		})
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
			if strings.HasPrefix(c.AdminPubStr, string(admn)) {
				adminPublic = c.AdminPubStr
			} else if strings.HasPrefix(c.AdminPubStr2, string(admn)) {
				adminPublic = c.AdminPubStr2
			} else {
				hadError = "no matching admin key found" + c.Topic
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
			devicePrivateKeyTmp, err := base64.RawURLEncoding.DecodeString(c.PrivStr)
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
					//fmt.Println("decrypted command is ", message)
					SpecialPrint(&pub.PacketCommon, func() {
						fmt.Println("decrypted command is ", strings.Split(message, "\n")[0])
					})

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
		reply = "command Error: " + hadError
	} else {

		args := make([]string, 0, 10)

		// this doesn't work right with command with args
		// like 'set some text abc' FIXME! atw
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
		if strings.Contains(cmd.Description, "🔓") {
			reply = cmd.Execute(message, args, nil)
		} else {
			if !hadEncryption {
				reply = "Error: this command requires encryption"
				c.fail++
			} else {
				reply = cmd.Execute(message, args, nil)
			}
		}
	}
	nonc, ok := pub.GetOption("nonc")
	if nonc == nil || !ok {
		hadError = "Error: no nonce"
	}

	if hadError == "" && !strings.Contains(cmd.Description, "🔓") {
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
		if strings.HasPrefix(c.AdminPubStr, string(admn)) {
			adminPublic = c.AdminPubStr
		} else if strings.HasPrefix(c.AdminPubStr2, string(admn)) {
			adminPublic = c.AdminPubStr2
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
		devicePrivateKeyTmp, err := base64.RawURLEncoding.DecodeString(c.PrivStr)
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
		SpecialPrint(&pub.PacketCommon, func() {
			fmt.Println("encrypted reply is ", c.Topic, reply, "nonce", string(nonc))
		})
	}

	if isHttp {
		sttrbuf := strings.Builder{}
		sttrbuf.WriteString("HTTP/1.1 200 OK\r\n")
		sttrbuf.WriteString("Content-Length: ")
		sttrbuf.WriteString(strconv.FormatInt(int64(len(reply)), 10))
		sttrbuf.WriteString("\r\n")
		sttrbuf.WriteString("Content-Type: text/plain\r\n")
		sttrbuf.WriteString("Access-Control-Allow-Origin: *\r\n")
		sttrbuf.WriteString("access-control-expose-headers: nonc\r\n")
		// ?? atw fixme: make nginx-ingress happy
		sttrbuf.WriteString("Connection: Closed\r\n") // ?? atw fixme: make nginx-ingress happy
		// sttrbuf.WriteString("nonc: " + string(nonc) + "\r\n") // this might be redundant
		sttrbuf.WriteString("\r\n")
		sttrbuf.WriteString(reply)
		reply = sttrbuf.String()
	}

	// fmt.Println("monitor reply ", reply)

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
			err = sub.Write(conn)
			if err != nil {
				println("write testtopic failed:", err.Error())
				fail++
			}
			conn.Close()
			time.Sleep(10 * time.Second)
		}
	}()
}

func SetupKeys(c *ThingContext) {

	c.Password = "testString123"

	pubk, privk := tokens.GetBoxKeyPairFromPassphrase(c.Password)
	c.PubStr = base64.RawURLEncoding.EncodeToString(pubk[:])
	c.PrivStr = base64.RawURLEncoding.EncodeToString(privk[:])

	adminPassPhrase := "myFamousOldeSaying"
	pubk, privk = tokens.GetBoxKeyPairFromPassphrase(adminPassPhrase)
	c.AdminPubStr = base64.RawURLEncoding.EncodeToString(pubk[:])
	c.AdminPrivStr = base64.RawURLEncoding.EncodeToString(privk[:])

	adminPassPhrase2 := "myFamousOldeSaying2"
	pubk, privk = tokens.GetBoxKeyPairFromPassphrase(adminPassPhrase2)
	c.AdminPubStr2 = base64.RawURLEncoding.EncodeToString(pubk[:])
	c.AdminPrivStr2 = base64.RawURLEncoding.EncodeToString(privk[:])

	c.dummyString = "none"
}

func setupCommands(c *ThingContext) {

	SetupKeys(c)

	//
	// pubk, privk := tokens.GetBoxKeyPairFromPassphrase(c.password)
	// c.pubStr = base64.RawURLEncoding.EncodeToString(pubk[:])
	// c.privStr = base64.RawURLEncoding.EncodeToString(privk[:])

	// // adminPassPhrase := "myFamousOldeSaying"
	// pubk, privk = tokens.GetBoxKeyPairFromPassphrase(adminPassPhrase)
	// c.adminPubStr = base64.RawURLEncoding.EncodeToString(pubk[:])
	// //c.adminPrivStr = base64.RawURLEncoding.EncodeToString(privk[:])

	// adminPassPhrase2 := "myFamousOldeSaying2"
	// pubk, privk = tokens.GetBoxKeyPairFromPassphrase(adminPassPhrase2)
	// c.adminPubStr2 = base64.RawURLEncoding.EncodeToString(pubk[:])
	// //c.adminPrivStr2 = base64.RawURLEncoding.EncodeToString(privk[:])

	// c.dummyString = "none"

	MakeCommand("get time",
		"seconds since 1970🔓", 0,
		func(msg string, args []string, callContext interface{}) string { //
			sec := time.Now().UnixMilli() / 1000
			secStr := strconv.FormatInt(sec, 10)
			return secStr
		}, c.CommandMap)

	MakeCommand("get c",
		"temperature in °C🔓", 0,
		func(msg string, args []string, callContext interface{}) string {
			tmp := (TempInF - 32) * 5 / 9
			tmp = math.Floor(tmp*100) / 100.0
			str := strconv.FormatFloat(tmp, 'f', 2, 64)
			return str + "°C"
		}, c.CommandMap)

	MakeCommand("get f",
		"temperature in °F🔓", 0,
		func(msg string, args []string, callContext interface{}) string {
			str := strconv.FormatFloat(TempInF, 'f', 2, 64)
			return str + "°F"
		}, c.CommandMap)

	MakeCommand("get humidity",
		"humidity in %🔓", 0,
		func(msg string, args []string, callContext interface{}) string {
			str := strconv.FormatFloat(Humidity, 'f', 2, 64)
			return str + "%"
		}, c.CommandMap)

	MakeCommand("get random",
		"returns a random integer", 0,
		func(msg string, args []string, callContext interface{}) string {
			tmp := rand.Uint32()
			secStr := strconv.FormatInt(int64(tmp), 10)
			return secStr
		}, c.CommandMap)
	MakeCommand("get count",
		"how many served since reboot", 0,
		func(msg string, args []string, callContext interface{}) string {
			countStr := strconv.FormatInt(int64(c.count), 10)
			return countStr
		}, c.CommandMap)

	MakeCommand("get fail",
		"how many requests were bad since reboot", 0,
		func(msg string, args []string, callContext interface{}) string {
			secStr := strconv.FormatInt(int64(c.fail), 10)
			return secStr
		}, c.CommandMap)

	MakeCommand("get pubk",
		"device public key 🔓", 0,
		func(msg string, args []string, callContext interface{}) string {
			return c.PubStr
		}, c.CommandMap)
	MakeCommand("get admin hint",
		"the first chars of the admin public keys🔓", 0,
		func(msg string, args []string, callContext interface{}) string {
			return c.AdminPubStr[0:8] + " " + c.AdminPubStr2[0:8]
		}, c.CommandMap)

	MakeCommand("get short name",
		"the local name🔓", 0,
		func(msg string, args []string, callContext interface{}) string {
			return "time"
		}, c.CommandMap)
	MakeCommand("get long name",
		"the global name", 0,
		func(msg string, args []string, callContext interface{}) string {
			return c.Topic
		}, c.CommandMap)
	MakeCommand("favicon.ico",
		"returns a green square png🔓", 0,
		func(msg string, args []string, callContext interface{}) string {
			return string(GreenSquare)
		}, c.CommandMap)
	MakeCommand("get some text",
		"return the saved text", 0,
		func(msg string, args []string, callContext interface{}) string {
			return c.dummyString
		}, c.CommandMap)
	MakeCommand("set some text",
		"save some text", 1,
		func(msg string, args []string, callContext interface{}) string {
			s := msg[len("set some text"):]
			s = strings.Trim(s, " ")
			c.dummyString = s
			return "ok"
		}, c.CommandMap)
	MakeCommand("version",
		"info about this thing", 0,
		func(msg string, args []string, callContext interface{}) string {
			return "v0.1.5"
		}, c.CommandMap)
	MakeCommand("help",
		"lists all commands. 🔓 means no encryption required", 0,
		func(msg string, args []string, callContext interface{}) string {
			s := ""
			keys := make([]string, 0, len(c.CommandMap)) //  maps.Keys(c.CommandMap)
			for k := range c.CommandMap {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				command := c.CommandMap[k]
				argCount := ""
				if command.ArgCount > 0 {
					argCount = " +" + strconv.FormatInt(int64(command.ArgCount), 10)
				}
				s += "[" + k + "]" + argCount + " " + command.Description + "\n"
			}
			return s
		}, c.CommandMap)
	MakeCommand("get token",
		"info about the token", 0,
		func(msg string, args []string, callContext interface{}) string {
			parts := strings.Split(c.Token, ".")
			if len(parts) != 3 {
				return "error: invalid token"
			}
			payloadB64 := parts[1]
			payload, err := base64.RawURLEncoding.DecodeString(payloadB64)
			if err != nil {
				return "error: " + err.Error()
			}
			return string(payload)
		}, c.CommandMap)
}

func getFromReno(cmd string) string {

	// read passphrase from ~/atw/renoIotpass.txt
	home, _ := os.UserHomeDir()
	fname := home + "/atw/renoIotpass.txt"
	fmt.Println("getFromReno fname", fname)

	tmp, err := os.ReadFile(fname)
	if err != nil {
		fmt.Println("TestGetIotResponseReno err", err)
		return "111"
	}
	passphrase := strings.TrimSpace(string(tmp))

	fmt.Println("getFromReno passphrase", len(passphrase))

	c := ThingContext{}

	c.Password = "demo-Device"

	pubk, privk := tokens.GetBoxKeyPairFromPassphrase(c.Password)
	c.PubStr = base64.RawURLEncoding.EncodeToString(pubk[:])
	c.PrivStr = base64.RawURLEncoding.EncodeToString(privk[:])
	_ = c.PrivStr
	c.PubStr = "iP8H8BJAvNsac3rI2SFXvGiHmDqZV3vxFFLEWE-8bnE"

	// c.PrivStr = base64.RawURLEncoding.EncodeToString(privk[:])

	adminPassPhrase := strings.TrimSpace(passphrase)
	pubk, privk = tokens.GetBoxKeyPairFromPassphrase(adminPassPhrase)
	c.AdminPubStr = base64.RawURLEncoding.EncodeToString(pubk[:])
	c.AdminPrivStr = base64.RawURLEncoding.EncodeToString(privk[:])

	server := "knotfree.io"
	thing := "demo-small-window-allow-should-engine"

	r := GetIotResponse(server, thing, cmd, c.PubStr, c.AdminPrivStr, c.AdminPubStr)

	fmt.Println("getFromReno r", r)
	return r
}

func ReplaceTempInF() {

	temp := getFromReno("get f")
	hum := getFromReno("get humidity")
	temp = temp[:len(temp)-3]
	hum = hum[:len(hum)-1]

	TempInF, _ = strconv.ParseFloat(temp, 64)
	Humidity, _ = strconv.ParseFloat(hum, 64)
	fmt.Println("replaceTempInF TempInF", TempInF, "Humidity", Humidity)
}

// GetIotResponse queries a thing for a response via http
// TODO: make a packet version of this. And Python version, etc.
func GetIotResponse(server string, thing string, cmd string, devicepubk string, adminprivk string, adminpubk string) string {
	result := "failed to get response"

	url := "http://" + thing + "." + server + "/"

	// if devicepubk then we need to encrypt the request
	if len(devicepubk) != 0 && len(adminprivk) != 0 {

		publicKeyBinary, err := base64.RawURLEncoding.DecodeString(devicepubk)
		if err != nil {
			fmt.Println("GetIotResponse err", err)
			return "error GetIotResponse:" + err.Error()
		}
		adminPrivateKeyBinary, err := base64.RawURLEncoding.DecodeString(adminprivk)
		if err != nil {
			fmt.Println("GetIotResponse err", err)
			return "error GetIotResponse:" + err.Error()
		}

		command := cmd
		payload := command + "#" + strconv.FormatInt(time.Now().Unix(), 10)
		nonceStr := tokens.GetRandomB36String()
		nonce := new([24]byte)
		copy(nonce[:], nonceStr[:])

		publicKeyBuffer := new([32]byte)
		copy(publicKeyBuffer[:], publicKeyBinary[:])
		adminPrivateKeyBuffer := new([32]byte)
		copy(adminPrivateKeyBuffer[:], adminPrivateKeyBinary[:])

		buffer := make([]byte, 0, (len(payload) + box.Overhead))
		sealed := box.Seal(buffer, []byte(payload), nonce, publicKeyBuffer, adminPrivateKeyBuffer)
		if len(sealed) == 0 {
			fmt.Println("GetIotResponse box fail")
			return "error GetIotResponse: box fail"
		}
		url += "=" + base64.RawURLEncoding.EncodeToString(sealed)
		url += "?nonc=" + nonceStr
		url += "&admn=" + adminpubk[0:8]

	} else {
		cmd = strings.ReplaceAll(cmd, " ", "/")
		url += cmd // in plain text
	}

	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("GetIotResponse err", err)
		return "error GetIotResponse:" + err.Error()
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("GetIotResponse err", err)
		return "error GetIotResponse:" + err.Error()
	}
	result = string(body)
	fmt.Println("GetIotResponse body", result)

	// if devicepubk then we need to decrypt the response
	if result[0] == '=' {
		// decrypt the response
		publicKeyBinary, err := base64.RawURLEncoding.DecodeString(devicepubk)
		if err != nil {
			fmt.Println("GetIotResponse err", err)
			return "error GetIotResponse:" + err.Error()
		}
		adminPrivateKeyBinary, err := base64.RawURLEncoding.DecodeString(adminprivk)
		if err != nil {
			fmt.Println("GetIotResponse err", err)
			return "error GetIotResponse:" + err.Error()
		}

		nonceStr := resp.Request.URL.Query().Get("nonc")
		nonce := new([24]byte)
		copy(nonce[:], nonceStr[:])

		publicKeyBuffer := new([32]byte)
		copy(publicKeyBuffer[:], publicKeyBinary[:])
		adminPrivateKeyBuffer := new([32]byte)
		copy(adminPrivateKeyBuffer[:], adminPrivateKeyBinary[:])

		sealed, err := base64.RawURLEncoding.DecodeString(result[1:]) // skip the '='
		if err != nil {
			fmt.Println("GetIotResponse err", err)
			return "error GetIotResponse:" + err.Error()
		}
		opened, ok := box.Open(nil, sealed, nonce, publicKeyBuffer, adminPrivateKeyBuffer)
		if !ok {
			fmt.Println("GetIotResponse box fail")
			return "error GetIotResponse: box fail"
		}
		result = string(opened)
		fmt.Println("GetIotResponse opened", result)
		// split the time off of it.
		parts := strings.Split(result, "#")
		if len(parts) < 2 {
			fmt.Println("GetIotResponse err", "no time")
			return "error GetIotResponse no time"
		}
		result = parts[0]
		now := time.Now().Unix()
		t, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			fmt.Println("GetIotResponse err", err)
			return "error GetIotResponse:" + err.Error()
		}
		delta := now - t
		if delta < 0 {
			delta = -delta
		}
		if delta > 10 {
			fmt.Println("GetIotResponse err", "time too old")
			return "error GetIotResponse: time too old"
		}
	}
	return result
}

func SpecialPrint(p *packets.PacketCommon, fn func()) {
	val, ok := p.GetOption("debg")
	if ok && (string(val) == "[12345678]" || string(val) == "12345678") {
		fn()
	}
}

// it's a green square PNG.
var GreenSquare = []byte{137, 80, 78, 71, 13, 10, 26, 10, 0, 0, 0, 13, 73, 72, 68,
	82, 0, 0, 0, 16, 0, 0, 0, 16, 8, 6, 0, 0, 0, 31, 243,
	255, 97, 0, 0, 0, 26, 73, 68, 65, 84, 120, 218, 99, 84, 106, 209,
	255, 207, 64, 1, 96, 28, 53, 96, 212, 128, 81, 3, 134, 139, 1, 0,
	239, 170, 29, 81, 139, 188, 27, 125, 0, 0, 0, 0, 73, 69, 78, 68,
	174, 66, 96, 130}

// Copyright 2022,2023,2024 Alan Tracey Wootton
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
