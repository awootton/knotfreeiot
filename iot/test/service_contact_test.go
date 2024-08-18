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
	"golang.org/x/crypto/nacl/box"
)

func TestServiceContactTCP_prod(t *testing.T) {

	address := "knotfree.io:8384"
	token, _ := tokens.GetImpromptuGiantTokenLocal("", "")
	sc, err := iot.StartNewServiceContactTcp(address, token)
	check(err)

	time.Sleep(5 * time.Second)

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

	time.Sleep(15 * time.Second)

	{
		command := "get option A"
		cmd := packets.Lookup{}
		cmd.Address.FromString(name)
		cmd.SetOption("cmd", []byte(command))

		// send it
		reply, err := sc.Get(&cmd)
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

	fmt.Println("ServiceContactTcp_prod test done")
}

func TestGetA(t *testing.T) {

	iot.InitMongEnv()
	iot.InitIotTables()

	ce := makeClusterWithServiceContact()
	sc := ce.ServiceContact

	// devicePublicKey := ce.PublicKeyTemp
	// devicePublicKeyStr := base64.URLEncoding.EncodeToString(devicePublicKey[:])
	//devicePublicKeyStr = strings.TrimRight(devicePublicKeyStr, "=")

	//make a person
	// passphrase := "a-person-passphrase"
	// pubk, privk := tokens.GetBoxKeyPairFromPassphrase(passphrase)
	// pubkStr := base64.URLEncoding.EncodeToString(pubk[:])
	// pubkStr = strings.TrimRight(pubkStr, "=")

	// make an internet name
	name := "a-person-channel_iot"

	// token, payload := tokens.GetImpromptuGiantTokenLocal(pubkStr)
	// _ = token
	// // let's make a reserved subscription

	// nonceStr := []byte(tokens.GetRandomB36String())
	// nonce := new([24]byte)
	// copy(nonce[:], nonceStr[:])

	// timeStr := strconv.FormatInt(time.Now().Unix(), 10)
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
		// payload := command + " " + timeStr

		// buffer := make([]byte, 0, (len(payload) + box.Overhead))
		// sealed := box.Seal(buffer, []byte(payload), nonce, devicePublicKey, &privk)
		// cmd.SetOption("sealed", sealed)

		// send it
		reply, err := sc.Get(&cmd)
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

func TestReserve(t *testing.T) {

	iot.InitMongEnv()
	iot.InitIotTables()

	ce := makeClusterWithServiceContact()
	sc := ce.ServiceContact

	devicePublicKey := ce.PublicKeyTemp
	// devicePublicKeyStr := base64.URLEncoding.EncodeToString(devicePublicKey[:])
	//devicePublicKeyStr = strings.TrimRight(devicePublicKeyStr, "=")

	//make a person
	passphrase := "a-person-passphrase"
	pubk, privk := tokens.GetBoxKeyPairFromPassphrase(passphrase)
	pubkStr := base64.URLEncoding.EncodeToString(pubk[:])
	pubkStr = strings.TrimRight(pubkStr, "=")

	// make an internet name
	name := "a-person-channel_iot"

	token, payload := tokens.GetImpromptuGiantTokenLocal(pubkStr, "")
	_ = token
	// let's make a reserved subscription

	nonceStr := []byte(tokens.GetRandomB36String())
	nonce := new([24]byte)
	copy(nonce[:], nonceStr[:])

	timeStr := strconv.FormatInt(time.Now().Unix(), 10)
	{
		command := "reserve"
		cmd := packets.Lookup{}
		cmd.Address.FromString(name)
		// fixme: serialize a struct instead of this
		cmd.SetOption("cmd", []byte(command))
		cmd.SetOption("pubk", []byte(pubkStr))
		cmd.SetOption("nonc", nonce[:]) // raw nonce, binary
		cmd.SetOption("jwtid", []byte(payload.JWTID))
		cmd.SetOption("name", []byte(name))
		// should we pass the whole token?

		// we need to sign this
		payload := command + " " + timeStr

		buffer := make([]byte, 0, (len(payload) + box.Overhead))
		sealed := box.Seal(buffer, []byte(payload), nonce, devicePublicKey, &privk)
		cmd.SetOption("sealed", sealed)

		// send it
		reply, err := sc.Get(&cmd)
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
		command := "set option WEB get-unix-time.knotfree.net"
		cmd := packets.Lookup{}
		cmd.Address.FromString(name)
		cmd.SetOption("cmd", []byte(command))
		cmd.SetOption("pubk", []byte(pubkStr))
		cmd.SetOption("nonc", nonce[:]) // raw nonce, binary
		cmd.SetOption("jwtid", []byte(payload.JWTID))
		// should we pass the whole token?

		// we need to sign this
		payload := command + " " + timeStr

		buffer := make([]byte, 0, (len(payload) + box.Overhead))
		sealed := box.Seal(buffer, []byte(payload), nonce, devicePublicKey, &privk)
		cmd.SetOption("sealed", sealed)

		// send it
		reply, err := sc.Get(&cmd)
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
		command := "set option A 216.128.128.195"
		cmd := packets.Lookup{}
		cmd.Address.FromString(name)
		cmd.SetOption("cmd", []byte(command))
		cmd.SetOption("pubk", []byte(pubkStr))
		cmd.SetOption("nonc", nonce[:]) // raw nonce, binary
		cmd.SetOption("jwtid", []byte(payload.JWTID))
		// should we pass the whole token?

		// we need to sign this
		payload := command + " " + timeStr

		buffer := make([]byte, 0, (len(payload) + box.Overhead))
		sealed := box.Seal(buffer, []byte(payload), nonce, devicePublicKey, &privk)
		cmd.SetOption("sealed", sealed)

		// send it
		reply, err := sc.Get(&cmd)
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
	sc := ce.ServiceContact

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
		command := "exists"
		cmd := packets.Lookup{}
		cmd.Address.FromString("get-unix-time")
		cmd.SetOption("cmd", []byte(command))
		cmd.SetOption("pubk", []byte(pubkStr))
		cmd.SetOption("nonc", nonce[:]) // raw nonce

		// we need to sign this
		payload := command + " " + timeStr

		buffer := make([]byte, 0, (len(payload) + box.Overhead))
		sealed := box.Seal(buffer, []byte(payload), nonce, devicePublicKey, &privk)
		cmd.SetOption("sealed", sealed)

		// send it
		reply, err := sc.Get(&cmd)
		if err == nil {
			got := string(reply.(*packets.Send).Payload)
			want := "false"
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
		command := "get pubk"
		cmd := packets.Lookup{}
		cmd.Address.FromString(name)
		cmd.SetOption("cmd", []byte(command))
		cmd.SetOption("pubk", []byte(pubkStr))
		cmd.SetOption("nonc", nonce[:]) // raw nonce

		// we need to sign this
		payload := command + " " + timeStr

		buffer := make([]byte, 0, (len(payload) + box.Overhead))
		sealed := box.Seal(buffer, []byte(payload), nonce, devicePublicKey, &privk)
		cmd.SetOption("sealed", sealed)

		// send it
		reply, err := sc.Get(&cmd)
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

	{
		command := "exists"
		cmd := packets.Lookup{}
		cmd.Address.FromString("get-unix-time")
		cmd.SetOption("cmd", []byte(command))
		cmd.SetOption("pubk", []byte(pubkStr))
		cmd.SetOption("nonc", nonce[:]) // raw nonce

		// we need to sign this
		payload := command + " " + timeStr

		buffer := make([]byte, 0, (len(payload) + box.Overhead))
		sealed := box.Seal(buffer, []byte(payload), nonce, devicePublicKey, &privk)
		cmd.SetOption("sealed", sealed)

		// send it
		reply, err := sc.Get(&cmd)
		if err == nil {
			got := string(reply.(*packets.Send).Payload)
			want := "true"
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
	sc := ce.ServiceContact

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
	reply, err = sc.Get(&msg)
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
	ce.ServiceContact, err = iot.StartNewServiceClient(ce.Aides[0])
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
		t.Error("SendPacket returned wanted timeout", string(reply.(*packets.Send).Payload))
	} else {
		fmt.Println("SendPacket returned", string(reply.(*packets.Send).Payload))
	}
	fmt.Println("ServiceContactTcp test done")
}

func TestServiceContactTCP_DNS(t *testing.T) {

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
			t.Error("SendPacket returned wanted timeout", string(reply.(*packets.Send).Payload))
		} else {
			fmt.Println("SendPacket returned", string(reply.(*packets.Send).Payload))
		}
	}
	// time.Sleep(10 * time.Second)
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
			t.Error("SendPacket returned wanted timeout", string(reply.(*packets.Send).Payload))
		} else {
			fmt.Println("SendPacket returned", string(reply.(*packets.Send).Payload))
		}
	}

	fmt.Println("ServiceContactTcp test done")
}
