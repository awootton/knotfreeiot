package iot_test

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/awootton/knotfreeiot/iot"
	"github.com/awootton/knotfreeiot/monitor_pod"
	"github.com/awootton/knotfreeiot/packets"
	"github.com/awootton/knotfreeiot/tokens"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/nacl/box"
)

func TestReserveOneName(t *testing.T) {

	iot.InitMongEnv()
	iot.InitIotTables()

	ce := makeClusterWithServiceContact()
	sc := ce.PacketService
	_ = sc

	devicePublicKey := ce.PublicKeyTemp
	devicePublicKeyStr := base64.URLEncoding.EncodeToString(devicePublicKey[:])
	devicePublicKeyStr = strings.TrimRight(devicePublicKeyStr, "=")
	_ = devicePublicKeyStr

	passphrase := "a-person-passphrase"
	pubk, privk := tokens.GetBoxKeyPairFromPassphrase(passphrase)
	pubkStr := base64.URLEncoding.EncodeToString(pubk[:])
	pubkStr = strings.TrimRight(pubkStr, "=")
	fmt.Println("pubkStr", pubkStr)
	fmt.Println("privkStr", base64.URLEncoding.EncodeToString(privk[:]))

	// pubk, privk := tokens.GetBoxKeyPairFromPassphrase(string(passPhrase))
	// pubkStr := base64.URLEncoding.EncodeToString(pubk[:])
	// pubkStr = strings.TrimRight(pubkStr, "=")
	_ = privk
	fmt.Println("pubkStr", pubkStr)

	// keep using the same jwtid as before
	token, payload := tokens.GetImpromptuGiantTokenLocal(pubkStr, "plfdfo4ezlgclcumjtqkiwre")
	_ = token
	_ = payload

	name := "a-person-channel-one-name_iot"
	{
		name = strings.TrimSpace(name)
		fmt.Println("Reserving", name)

		nonceStr := []byte("dd4lh93s2qqw1cfkmbzokrch") // tokens.GetRandomB36String())
		nonce := new([24]byte)
		copy(nonce[:], nonceStr[:])

		timeStr := strconv.FormatInt(time.Now().Unix(), 10)
		{
			command := "reserve " + name + " " + token
			cmd := packets.Lookup{}
			cmd.Address.FromString(name)
			// fixme: serialize a struct instead of this?
			cmd.SetOption("cmd", []byte(command))
			cmd.SetOption("pubk", []byte(pubkStr))
			cmd.SetOption("nonc", nonce[:]) // raw nonce, binary
			// cmd.SetOption("jwtid", []byte(payload.JWTID))
			// cmd.SetOption("name", []byte(name))
			// should we pass the whole token?

			// we need to sign this
			payload := command + "#" + timeStr

			buffer := make([]byte, 0, (len(payload) + box.Overhead))
			sealed := box.Seal(buffer, []byte(payload), nonce, ce.PublicKeyTemp, &privk)
			cmd.SetOption("sealed", sealed)

			// send it
			reply, err := sc.GetPacketReplyLonger(&cmd, time.Duration(5555*time.Second))
			if err == nil {
				got := string(reply.(*packets.Send).Payload)
				want := "ok"
				if got != want {
					t.Error("reply got", got, "want", want)
					fmt.Println("reply got", got, "want", want)
				}
			} else {
				t.Error("reply err", err)
				fmt.Println("reply err", err)
			}
		}
	}
}

// TestServiceContactTCP_prod will sometime fail the first time it is run.
func TestServiceContactTCP_prod(t *testing.T) {

	address := "knotfree.io:8384"
	token, _ := tokens.GetImpromptuGiantTokenLocal("", "")
	sc, err := iot.StartNewServiceContactTcp(address, token)
	check(err)

	// time.Sleep(5 * time.Second)

	fmt.Println("ServiceContactTcp_prod test start 1")
	fmt.Println("ServiceContactTcp_prod test start 1")
	fmt.Println("ServiceContactTcp_prod test start 1")
	fmt.Println("ServiceContactTcp_prod test start 1")

	name := "a-person-channel_iot"
	{
		command := "get option A"
		cmd := packets.Lookup{}
		cmd.Address.FromString(name)
		cmd.SetOption("cmd", []byte(command))

		// send it
		reply, err := sc.Get(&cmd)
		if err == nil {
			got := string(reply.(*packets.Send).Payload)
			fmt.Println("reply 1 got", got)
			want := "216.128.128.195"
			if got != want {
				t.Error("reply got", got, "want", want)
				fmt.Println("reply got", got, "want", want)
			}
		} else {
			t.Error("reply err", err)
			fmt.Println("reply err", err)
		}
	}

	// time.Sleep(5 * time.Second)

	fmt.Println("ServiceContactTcp_prod test start 2")
	fmt.Println("ServiceContactTcp_prod test start 2")
	fmt.Println("ServiceContactTcp_prod test start 2")
	fmt.Println("ServiceContactTcp_prod test start 2")

	starttime := time.Now()
	for i := 0; i < 10; i++ {
		command := "get option A"
		cmd := packets.Lookup{}
		cmd.Address.FromString(name)
		cmd.SetOption("cmd", []byte(command))

		// send it
		reply, err := sc.Get(&cmd)
		if err == nil {
			got := string(reply.(*packets.Send).Payload)
			// fmt.Println("reply 2 got", got)
			want := "216.128.128.195"
			if got != want {
				t.Error("reply got", got, "want", want)
				fmt.Println("reply got", got, "want", want)
			}
		} else {
			t.Error("reply err", err)
			fmt.Println("reply err", err)
		}
	}
	elapsed := time.Since(starttime)
	fmt.Println("ServiceContactTcp_prod test done", elapsed)
}

func TestGetA(t *testing.T) {

	iot.InitMongEnv()
	iot.InitIotTables()

	ce := makeClusterWithServiceContact()
	sc := ce.PacketService

	// make an internet name
	name := "a-person-channel_iot"
	{
		command := "get option A"
		cmd := packets.Lookup{}
		cmd.Address.FromString(name)
		cmd.SetOption("cmd", []byte(command))
		// cmd.SetOption("pubk", []byte(pubkStr))
		// cmd.SetOption("nonc", nonce[:]) // raw nonce, binary
		// cmd.SetOption("jwtid", []byte(payload.JWTID))
		// should we pass the whole token?

		// we need to sign this
		// payload := command + "#" + timeStr

		// buffer := make([]byte, 0, (len(payload) + box.Overhead))
		// sealed := box.Seal(buffer, []byte(payload), nonce, devicePublicKey, &privk)
		// cmd.SetOption("sealed", sealed)

		// send it
		reply, err := sc.GetPacketReply(&cmd)
		if err == nil {
			got := string(reply.(*packets.Send).Payload)
			want := "216.128.128.195"
			if got != want {
				t.Error("reply got", got, "want", want)
				fmt.Println("reply got", got, "want", want)
			}
		} else {
			t.Error("reply err", err)
			fmt.Println("reply err", err)
		}
	}
	// time.Sleep(1000 * time.Second)
}

// TestReserve tests the whole reserve a subscription process
// with 3 names. I am concerned about the time it takes to run.
func TestReserve(t *testing.T) {

	iot.InitMongEnv()
	iot.InitIotTables()

	ce := makeClusterWithServiceContact()
	sc := ce.PacketService

	devicePublicKey := ce.PublicKeyTemp
	// devicePublicKeyStr := base64.URLEncoding.EncodeToString(devicePublicKey[:])
	//devicePublicKeyStr = strings.TrimRight(devicePublicKeyStr, "=")

	//make a person
	passphrase := "atwadmin"
	pubk, privk := tokens.GetBoxKeyPairFromPassphrase(passphrase)
	pubkStr := base64.URLEncoding.EncodeToString(pubk[:])
	pubkStr = strings.TrimRight(pubkStr, "=")
	fmt.Println("pubkStr", pubkStr)

	passphrase = "a-person-passphrase"
	pubk, privk = tokens.GetBoxKeyPairFromPassphrase(passphrase)
	pubkStr = base64.URLEncoding.EncodeToString(pubk[:])
	pubkStr = strings.TrimRight(pubkStr, "=")
	fmt.Println("pubkStr", pubkStr)
	token, payload := tokens.GetImpromptuGiantTokenLocal(pubkStr, "")

	// make an internet name
	names := []string{"a-person-channel_iot", "a-person-channel_vr", "a-person-channel_pod"}
	for i := 0; i < 10; i++ {
		names = append(names, fmt.Sprintf("a-person-channel-%d", i))
	}
	i := -1
	for _, name := range names {
		i++

		// let's make a reserved subscription
		nonceStr := []byte(tokens.GetRandomB36String())
		nonce := new([24]byte)
		copy(nonce[:], nonceStr[:])

		timeStr := strconv.FormatInt(time.Now().Unix(), 10)
		{
			command := "reserve " + name + " " + token
			cmd := packets.Lookup{}
			cmd.Address.FromString(name)
			// fixme: serialize a struct instead of this
			cmd.SetOption("cmd", []byte(command))
			cmd.SetOption("pubk", []byte(pubkStr))
			cmd.SetOption("nonc", nonce[:]) // raw nonce, binary
			//cmd.SetOption("jwtid", []byte(payload.JWTID))
			// cmd.SetOption("name", []byte(name))
			// should we pass the whole token?

			// we need to sign this
			payload := command + "#" + timeStr

			buffer := make([]byte, 0, (len(payload) + box.Overhead))
			sealed := box.Seal(buffer, []byte(payload), nonce, devicePublicKey, &privk)
			cmd.SetOption("sealed", sealed)

			// send it
			reply, err := sc.GetPacketReply(&cmd)
			if err == nil {
				got := string(reply.(*packets.Send).Payload)
				want := "ok"
				if got != want {
					t.Error("reply got", got, "want", want)
					fmt.Println("reply got", got, "want", want)
				}
			} else {
				t.Error("reply err", err)
				fmt.Println("reply err", err)
			}
		}
		{
			command := "set option web get-unix-time.knotfree.net"
			cmd := packets.Lookup{}
			cmd.Address.FromString(name)
			cmd.SetOption("cmd", []byte(command))
			cmd.SetOption("pubk", []byte(pubkStr))
			cmd.SetOption("nonc", nonce[:]) // raw nonce, binary
			cmd.SetOption("jwtid", []byte(payload.JWTID))
			// should we pass the whole token?

			timeStr = strconv.FormatInt(time.Now().Unix(), 10)

			// we need to sign this
			payload := command + "#" + timeStr

			buffer := make([]byte, 0, (len(payload) + box.Overhead))
			sealed := box.Seal(buffer, []byte(payload), nonce, devicePublicKey, &privk)
			cmd.SetOption("sealed", sealed)

			// send it
			reply, err := sc.GetPacketReply(&cmd)
			if err == nil {
				got := string(reply.(*packets.Send).Payload)
				want := "ok"
				if got != want {
					t.Error("reply got", got, "want", want)
					fmt.Println("reply got", got, "want", want)
				}
			} else {
				t.Error("reply err", err)
				fmt.Println("reply err", err)
			}
		}
		{
			command := "set option a 216.128.128.195"
			cmd := packets.Lookup{}
			cmd.Address.FromString(name)
			cmd.SetOption("cmd", []byte(command))
			cmd.SetOption("pubk", []byte(pubkStr))
			cmd.SetOption("nonc", nonce[:]) // raw nonce, binary
			cmd.SetOption("jwtid", []byte(payload.JWTID))
			// should we pass the whole token?

			timeStr = strconv.FormatInt(time.Now().Unix(), 10)

			// we need to sign this
			payload := command + "#" + timeStr

			buffer := make([]byte, 0, (len(payload) + box.Overhead))
			sealed := box.Seal(buffer, []byte(payload), nonce, devicePublicKey, &privk)
			cmd.SetOption("sealed", sealed)

			// send it
			reply, err := sc.GetPacketReply(&cmd)
			if err == nil {
				got := string(reply.(*packets.Send).Payload)
				want := "ok"
				if got != want {
					t.Error("reply got", got, "want", want)
					fmt.Println("reply got", got, "want", want)
				}
			} else {
				t.Error("reply err", err)
				fmt.Println("reply err", err)
			}
		}
		// FIXME: this doesn't work. we need to fix it.
		if i == 2 {
			command := "bulk option txt @ basetext bitcoin 3G2cGahYrRNXWbUQLtCaF8joHD3VJyS7hr dummy dummy"
			cmd := packets.Lookup{}
			cmd.Address.FromString(name)
			cmd.SetOption("cmd", []byte(command))
			cmd.SetOption("pubk", []byte(pubkStr))
			cmd.SetOption("nonc", nonce[:]) // raw nonce, binary
			cmd.SetOption("jwtid", []byte(payload.JWTID))
			// should we pass the whole token?

			timeStr = strconv.FormatInt(time.Now().Unix(), 10)

			// we need to sign this
			payload := command + "#" + timeStr

			buffer := make([]byte, 0, (len(payload) + box.Overhead))
			sealed := box.Seal(buffer, []byte(payload), nonce, devicePublicKey, &privk)
			cmd.SetOption("sealed", sealed)

			// send it
			reply, err := sc.GetPacketReply(&cmd)
			if err == nil {
				got := string(reply.(*packets.Send).Payload)
				want := "ok"
				if got != want {
					t.Error("reply got", got, "want", want)
					fmt.Println("reply got", got, "want", want)
				}
			} else {
				t.Error("reply err", err)
				fmt.Println("reply err", err)
			}
		}

	}
	// now fetch it. get info

	// and fetch from mongo.

	_ = sc
	_ = privk
	// time.Sleep(1000 * time.Second)
}

// this soiuld be a bulk set if we impleme ted it
func XxxxTestSetLongOption(t *testing.T) {

	iot.InitMongEnv()
	iot.InitIotTables()

	ce := makeClusterWithServiceContact()
	sc := ce.PacketService

	devicePublicKey := ce.PublicKeyTemp
	// devicePublicKeyStr := base64.URLEncoding.EncodeToString(devicePublicKey[:])
	//devicePublicKeyStr = strings.TrimRight(devicePublicKeyStr, "=")

	//make a person
	passphrase := "atwadmin"
	pubk, privk := tokens.GetBoxKeyPairFromPassphrase(passphrase)
	pubkStr := base64.URLEncoding.EncodeToString(pubk[:])
	pubkStr = strings.TrimRight(pubkStr, "=")
	fmt.Println("pubkStr", pubkStr)

	passphrase = "a-person-passphrase"
	pubk, privk = tokens.GetBoxKeyPairFromPassphrase(passphrase)
	pubkStr = base64.URLEncoding.EncodeToString(pubk[:])
	pubkStr = strings.TrimRight(pubkStr, "=")
	fmt.Println("pubkStr", pubkStr)
	token, payload := tokens.GetImpromptuGiantTokenLocal(pubkStr, "")
	_ = token

	// make an internet name
	names := []string{"a-person-channel_pod"}
	i := -1
	for _, name := range names {
		i++

		// let's make a reserved subscription

		nonceStr := []byte(tokens.GetRandomB36String())
		nonce := new([24]byte)
		copy(nonce[:], nonceStr[:])

		{
			command := "set option txt @ basetext bitcoin 3G2cGahYrRNXWbUQLtCaF8joHD3VJyS7hr dummy dummy"
			cmd := packets.Lookup{}
			cmd.Address.FromString(name)
			cmd.SetOption("cmd", []byte(command))
			cmd.SetOption("pubk", []byte(pubkStr))
			cmd.SetOption("nonc", nonce[:]) // raw nonce, binary
			cmd.SetOption("jwtid", []byte(payload.JWTID))
			// should we pass the whole token?

			timeStr := strconv.FormatInt(time.Now().Unix(), 10)

			// we need to sign this
			payload := command + "#" + timeStr

			buffer := make([]byte, 0, (len(payload) + box.Overhead))
			sealed := box.Seal(buffer, []byte(payload), nonce, devicePublicKey, &privk)
			cmd.SetOption("sealed", sealed)

			// send it
			reply, err := sc.GetPacketReply(&cmd)
			if err == nil {
				got := string(reply.(*packets.Send).Payload)
				want := "ok"
				if got != want {
					t.Error("reply got", got, "want", want)
					fmt.Println("reply got", got, "want", want)
				}
			} else {
				t.Error("reply err", err)
				fmt.Println("reply err", err)
			}
		}
	}
	// now fetch it. get info

	// and fetch from mongo.

	_ = sc
	_ = privk
	// time.Sleep(1000 * time.Second)
}

// TestSubs tests the whole reserve a subscription process
// and the lookup of the subscription.
// TODO: move this to a different file
func TestSubs(t *testing.T) {

	ce := makeClusterWithServiceContact()
	sc := ce.PacketService

	devicePublicKey := ce.PublicKeyTemp
	devicePublicKeyStr := base64.URLEncoding.EncodeToString(devicePublicKey[:])
	devicePublicKeyStr = strings.TrimRight(devicePublicKeyStr, "=")

	//make a person
	passphrase := "a-person-passphrase"
	pubk, privk := tokens.GetBoxKeyPairFromPassphrase(passphrase)
	pubkStr := base64.URLEncoding.EncodeToString(pubk[:])

	// make an internet name
	name := "a-person-channel"

	// let's make a reserved subscription

	nonceStr := []byte(tokens.GetRandomB36String())
	nonce := new([24]byte)
	copy(nonce[:], nonceStr[:])

	timeStr := strconv.FormatInt(time.Now().Unix(), 10)
	{
		theName := "get-unix-time"
		command := "exists"
		cmd := packets.Lookup{}
		cmd.Address.FromString(theName)
		cmd.SetOption("cmd", []byte(command))
		cmd.SetOption("pubk", []byte(pubkStr))
		cmd.SetOption("nonc", nonce[:]) // raw nonce

		// we need to sign this
		payload := command + "#" + timeStr

		buffer := make([]byte, 0, (len(payload) + box.Overhead))
		sealed := box.Seal(buffer, []byte(payload), nonce, devicePublicKey, &privk)
		cmd.SetOption("sealed", sealed)

		// send it
		reply, err := sc.GetPacketReplyLonger(&cmd, time.Duration(5555*time.Second))
		if err == nil {
			got := string(reply.(*packets.Send).Payload)
			want := `{"Exists":true,"Online":false}`
			if got != want {
				t.Error("reply got", got, "want", want)
				fmt.Println("reply got", got, "want", want)
			}
		} else {
			t.Error("reply err", err)
			fmt.Println("reply err", err)
		}
	}
	timeStr = strconv.FormatInt(time.Now().Unix(), 10)
	{
		command := "get pubk"
		cmd := packets.Lookup{}
		cmd.Address.FromString(name)
		cmd.SetOption("cmd", []byte(command))
		cmd.SetOption("pubk", []byte(pubkStr))
		cmd.SetOption("nonc", nonce[:]) // raw nonce

		// we need to sign this
		payload := command + "#" + timeStr

		buffer := make([]byte, 0, (len(payload) + box.Overhead))
		sealed := box.Seal(buffer, []byte(payload), nonce, devicePublicKey, &privk)
		cmd.SetOption("sealed", sealed)

		// send it
		reply, err := sc.GetPacketReplyLonger(&cmd, time.Duration(5555*time.Second))
		if err == nil {
			got := string(reply.(*packets.Send).Payload)
			want := devicePublicKeyStr
			if got != want {
				t.Error("reply got", got, "want", want)
				fmt.Println("reply got", got, "want", want)
			}
		} else {
			t.Error("reply err", err)
			fmt.Println("reply err", err)
		}
	}
	// start a monitor server
	startAServer("get-unix-time", "")
	timeStr = strconv.FormatInt(time.Now().Unix(), 10)
	{
		command := "exists"
		cmd := packets.Lookup{}
		cmd.Address.FromString("get-unix-time")
		cmd.SetOption("cmd", []byte(command))
		cmd.SetOption("pubk", []byte(pubkStr))
		cmd.SetOption("nonc", nonce[:]) // raw nonce

		// we need to sign this
		payload := command + "#" + timeStr

		buffer := make([]byte, 0, (len(payload) + box.Overhead))
		sealed := box.Seal(buffer, []byte(payload), nonce, devicePublicKey, &privk)
		cmd.SetOption("sealed", sealed)

		// send it
		reply, err := sc.GetPacketReply(&cmd)
		if err == nil {
			got := string(reply.(*packets.Send).Payload)
			// this time it's online
			want := `{"Exists":true,"Online":true}`
			if got != want {
				t.Error("reply got", got, "want", want)
				fmt.Println("reply got", got, "want", want)
			}
		} else {
			t.Error("reply err", err)
			fmt.Println("reply err", err)
		}
	}

	_ = sc
	_ = privk
}
func TestServiceContact(t *testing.T) {

	ce := makeClusterWithServiceContact()
	sc := ce.PacketService

	var reply packets.Interface
	reply = &packets.Send{}
	var err error

	msg := packets.Send{}
	msg.Address.FromString("get-unix-time")
	msg.Payload = []byte("get time")

	// this is the timeout test and it works but is kinda slow for a unit test.
	//  put this back: reply, err := sc.Get(&msg)
	// if err != nil {
	// 	fmt.Println("SendPacket returned error and that's good", err)
	// } else {
	// 	t.Error("SendPacket returned wanted timeout", string(reply.(*packets.Send).Payload))
	// 	fmt.Println("SendPacket returned", string(reply.(*packets.Send).Payload))
	// }
	// Now. Start the get-unix-time service.

	startAServer("get-unix-time", "")

	// c := monitor_pod.ThingContext{}
	// c.Topic = "get-unix-time"
	// c.CommandMap = make(map[string]monitor_pod.Command)
	// c.Index = 0
	// c.Token = tokens.GetImpromptuGiantTokenLocal()
	// c.LogMeVerbose = true
	// c.Host = "localhost" + ":8384" //
	// fmt.Println("monitor main c.Host", c.Host)
	// monitor_pod.ServeGetTime(c.Token, &c)

	// try it again.
	// msg := packets.Send{}
	// msg.Address.FromString("get-unix-time")
	// msg.Payload = []byte("get time")
	reply, err = sc.GetPacketReply(&msg)
	if err != nil {
		fmt.Println("SendPacket returned error and that's bad", err)
		t.Error("SendPacket returned wanted timeout", string(reply.(*packets.Send).Payload))
	} else {
		fmt.Println("SendPacket returned", string(reply.(*packets.Send).Payload))
	}

	// ce.Heartbeat(getTime()) // this is'nt working. we have no test for the timeouts.
	// localtime += 60 // sec
	// ce.Heartbeat(getTime())
	// localtime += 60 * 25
	// ce.Heartbeat(getTime())
	// ce.Heartbeat(getTime())

	fmt.Println("ServiceContact test done")

}

func startAServer(name string, personPubk string) {
	c := monitor_pod.ThingContext{}
	c.Topic = name //"get-unix-time"
	c.CommandMap = make(map[string]monitor_pod.Command)
	c.Index = 0
	c.Token, _ = tokens.GetImpromptuGiantTokenLocal(personPubk, "")
	c.LogMeVerbose = true
	c.Host = "localhost" + ":8384" //
	fmt.Println("monitor main c.Host", c.Host)
	monitor_pod.ServeGetTime(c.Token, &c)
}

func makeClusterWithServiceContact() *iot.ClusterExecutive {
	tokens.LoadPublicKeys()

	var err error
	localtime := starttime
	getTime := func() uint32 {
		return localtime
	}

	ce := iot.MakeSimplestCluster(getTime, true, 1, "")
	ce.PacketService, err = iot.StartNewServiceContact(ce.Aides[0])
	check(err)

	return ce
}

func TestServiceContactTCP(t *testing.T) {

	tokens.LoadPublicKeys()

	var err error
	localtime := starttime
	getTime := func() uint32 {
		return localtime
	}
	ce := iot.MakeSimplestCluster(getTime, true, 1, "")
	_ = ce

	address := "localhost:8384"
	token, _ := tokens.GetImpromptuGiantTokenLocal("", "")
	sc, err := iot.StartNewServiceContactTcp(address, token)
	check(err)

	var reply packets.Interface
	reply = &packets.Send{}

	msg := packets.Send{}
	msg.Address.FromString("get-unix-time")
	msg.Payload = []byte("get time")

	// Now. Start the get-unix-time service.

	startAServer("get-unix-time", "")

	reply, err = sc.Get(&msg)
	if err != nil {
		fmt.Println("SendPacket returned error and that's bad", err)
		t.Error("SendPacket returned timeout")
	} else {
		fmt.Println("SendPacket returned", string(reply.(*packets.Send).Payload))
	}
	fmt.Println("ServiceContactTcp test done")
}

func TestServiceContactTCP_DNS(t *testing.T) {

	tokens.LoadPublicKeys()
	iot.InitMongEnv()
	iot.InitIotTables()

	var err error
	localtime := starttime
	getTime := func() uint32 {
		return localtime
	}
	ce := iot.MakeSimplestCluster(getTime, true, 1, "")
	_ = ce

	address := "localhost:8384"
	token, _ := tokens.GetImpromptuGiantTokenLocal("", "")
	sc, err := iot.StartNewServiceContactTcp(address, token)
	check(err)

	var reply packets.Interface
	reply = &packets.Send{}
	{
		name := "a-person-channel_iot"
		command := "get option A"
		cmd := packets.Lookup{}
		cmd.Address.FromString(name)
		cmd.SetOption("cmd", []byte(command))
		// Now. Start the get-unix-time service.

		// startAServer("get-unix-time", "")

		reply, err = sc.Get(&cmd)
		if err != nil {
			fmt.Println("SendPacket returned error and that's bad", err)
			assert.Equal(t, 0, 1, "SendPacket returned error and that's bad"+err.Error())
		} else {
			fmt.Println("SendPacket returned", string(reply.(*packets.Send).Payload))
		}
	}
	// time.Sleep(10 * time.Second)
	for i := 0; i < 10; i++ {
		name := "a-person-channel_iot"
		command := "get option A"
		cmd := packets.Lookup{}
		cmd.Address.FromString(name)
		cmd.SetOption("cmd", []byte(command))
		// Now. Start the get-unix-time service.

		// startAServer("get-unix-time", "")

		reply, err = sc.Get(&cmd)
		if err != nil {
			fmt.Println("SendPacket returned error and that's bad", err)
			assert.Equal(t, 0, 1, "SendPacket returned timeout"+err.Error())
		} else {
			fmt.Println("SendPacket returned", string(reply.(*packets.Send).Payload))
		}
	}

	fmt.Println("ServiceContactTcp test done")
}
