package iot_test

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/awootton/knotfreeiot/iot"
	"github.com/awootton/knotfreeiot/packets"
	"github.com/awootton/knotfreeiot/tokens"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/nacl/box"
)

func TestNameApiDeleteName(t *testing.T) {

	ce := makeClusterWithServiceContact()
	iot.InitMongEnv()
	iot.InitIotTables()

	sc := ce.PacketService
	devicePublicKey := ce.PublicKeyTemp
	_ = devicePublicKey
	_ = sc
	fmt.Println("devicePublicKey", base64.URLEncoding.EncodeToString(devicePublicKey[:]))

	passphrase := "a-person-passphrase"
	pubk, privk := tokens.GetBoxKeyPairFromPassphrase(passphrase)
	pubkStr := base64.URLEncoding.EncodeToString(pubk[:])
	pubkStr = strings.TrimRight(pubkStr, "=")
	fmt.Println("pubkStr", pubkStr)
	fmt.Println("privkStr", base64.URLEncoding.EncodeToString(privk[:]))

	token := tokens.GetTest32xTokenwjwtid(pubkStr, "f9boplwyb2wxsjtkspnucrth")
	payload, err := tokens.ValidateToken(string(token))
	fmt.Println("payload", payload, err)

	newName := "test-new-name-delete-me"

	// hash it, just checking.
	cmd := packets.Lookup{}
	cmd.Address.FromString(newName)
	cmd.Address.EnsureAddressIsBinary() // 72 f3 39 ...
	var topicHash iot.HashType
	topicHash.InitFromBytes(cmd.Address.Bytes) // 8283027105011586995 ...
	var b [24]byte
	topicHash.GetBytes(b[:])     // // 72 f3 39 ...
	str1 := topicHash.ToBase64() // cvM5REMl17OdX1yZeH4fr7668Ucpn8uR
	var hash iot.HashType
	hash.HashString(newName)
	str2 := hash.ToBase64()
	if str1 != str2 {
		t.Error("expected", str2, "got", str1)
	}

	// add a new name with web-api
	{
		command := "reserve " + newName + " " + string(token)
		payload := command + "#" + strconv.FormatInt(time.Now().Unix(), 10)
		nonceStr := tokens.GetRandomB36String()
		nonce := new([24]byte)
		copy(nonce[:], nonceStr[:])

		buffer := make([]byte, 0, (len(payload) + box.Overhead))
		sealed := box.Seal(buffer, []byte(payload), nonce, devicePublicKey, &privk)

		// fmt.Println("sealed", base64.RawURLEncoding.EncodeToString(sealed))
		// fmt.Println("devicePublicKey", base64.RawURLEncoding.EncodeToString(devicePublicKey[:]))
		// fmt.Println("privk", base64.RawURLEncoding.EncodeToString(privk[:]))
		// fmt.Println("nonce", nonceStr)

		// sign it
		uri := "http://knotfree.com:8085/api1/nameService?"
		uri += "&sealed=" + base64.RawURLEncoding.EncodeToString(sealed)
		uri += "&cmd=" + strings.ReplaceAll(command, " ", "%20")
		uri += "&nonce=" + nonceStr
		uri += "&pubk=" + pubkStr // the owner pub key of the name
		uri += "&name=" + newName
		val := getVal(t, uri)

		fmt.Println("get-names returned ", val)

		if val != "ok" {
			t.Error("expected ok, got", val)
		}
	}

	// do the delete
	{
		command := "delete " + newName
		payload := command + "#" + strconv.FormatInt(time.Now().Unix(), 10)
		nonceStr := tokens.GetRandomB36String()
		nonce := new([24]byte)
		copy(nonce[:], nonceStr[:])

		buffer := make([]byte, 0, (len(payload) + box.Overhead))
		sealed := box.Seal(buffer, []byte(payload), nonce, devicePublicKey, &privk)

		// fmt.Println("sealed", base64.RawURLEncoding.EncodeToString(sealed))
		// fmt.Println("devicePublicKey", base64.RawURLEncoding.EncodeToString(devicePublicKey[:]))
		// fmt.Println("privk", base64.RawURLEncoding.EncodeToString(privk[:]))
		// fmt.Println("nonce", nonceStr)

		// sign it
		uri := "http://knotfree.com:8085/api1/nameService?"
		uri += "&sealed=" + base64.RawURLEncoding.EncodeToString(sealed)
		uri += "&cmd=" + strings.ReplaceAll(command, " ", "%20")
		uri += "&nonce=" + nonceStr
		uri += "&pubk=" + pubkStr // the owner pub key of the name
		uri += "&name=" + newName
		val := getVal(t, uri)

		fmt.Println("delete returned ", val)

		if val != "ok" {
			t.Error("expected ok, got", val)
		}
	}
	// make sure it's gone
	{ // this is overkill because "exists" doesn't require encryption
		nonceStr := tokens.GetRandomB36String()
		nonce := new([24]byte)
		copy(nonce[:], nonceStr[:])

		command := "exists"
		cmd := packets.Lookup{}
		cmd.Address.FromString(newName)
		cmd.SetOption("cmd", []byte(command))
		cmd.SetOption("pubk", []byte(pubkStr))
		cmd.SetOption("nonc", nonce[:]) // raw nonce

		timeStr := strconv.FormatInt(time.Now().Unix(), 10)
		// we need to sign this
		payload := command + "#" + timeStr

		buffer := make([]byte, 0, (len(payload) + box.Overhead))
		sealed := box.Seal(buffer, []byte(payload), nonce, devicePublicKey, &privk)
		cmd.SetOption("sealed", sealed)

		// send it
		reply, err := sc.GetPacketReplyLonger(&cmd, time.Duration(5555*time.Second))
		if err == nil {
			got := string(reply.(*packets.Send).Payload)
			want := `{"Exists":false,"Online":false}`
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

func TestNameApiSetOption(t *testing.T) {

	ce := makeClusterWithServiceContact()
	iot.InitMongEnv()
	iot.InitIotTables()

	sc := ce.PacketService
	_ = sc
	fmt.Println("devicePublicKey", base64.URLEncoding.EncodeToString(ce.PublicKeyTemp[:]))

	passphrase := "a-person-passphrase"
	pubk, privk := tokens.GetBoxKeyPairFromPassphrase(passphrase)
	pubkStr := base64.URLEncoding.EncodeToString(pubk[:])
	pubkStr = strings.TrimRight(pubkStr, "=")
	fmt.Println("pubkStr", pubkStr)
	fmt.Println("privkStr", base64.URLEncoding.EncodeToString(privk[:]))

	token := tokens.GetTest32xTokenwjwtid(pubkStr, "f9boplwyb2wxsjtkspnucrth")
	payload, err := tokens.ValidateToken(string(token))
	fmt.Println("payload", payload, err)

	aName := "a-person-channel_pod" // has complicated subkeys
	randStr := tokens.GetRandomB36String()
	{
		command := "set option txt dummy2 dummy2Val" + randStr // with subkey
		payload := command + "#" + strconv.FormatInt(time.Now().Unix(), 10)
		nonceStr := "07l70hfs8765r6z792dbf5j0" // tokens.GetRandomB36String()
		nonce := new([24]byte)
		copy(nonce[:], nonceStr[:])

		buffer := make([]byte, 0, (len(payload) + box.Overhead))
		sealed := box.Seal(buffer, []byte(payload), nonce, ce.PublicKeyTemp, &privk)

		fmt.Println("sealed", base64.RawURLEncoding.EncodeToString(sealed))
		fmt.Println("devicePublicKey", base64.RawURLEncoding.EncodeToString(ce.PublicKeyTemp[:]))
		fmt.Println("privk", base64.RawURLEncoding.EncodeToString(privk[:]))
		fmt.Println("nonce", nonceStr)

		// sign it
		uri := "http://knotfree.com:8085/api1/nameService?cmd=" + strings.ReplaceAll(command, " ", "%20")
		uri += "&sealed=" + base64.RawURLEncoding.EncodeToString(sealed)
		uri += "&nonce=" + nonceStr
		uri += "&pubk=" + pubkStr // the owner pub key of the name
		uri += "&name=" + aName

		{
			pubkBytes, err := base64.RawURLEncoding.DecodeString(string(pubkStr))
			_ = err
			var nonce2 [24]byte
			copy(nonce2[:], nonceStr)
			var pubk2 [32]byte
			copy(pubk2[:], pubkBytes)

			out := make([]byte, 0, len(sealed)) // it's actually smaller

			result2, ok2 := box.Open(out, sealed, &nonce2, &pubk2, ce.PrivateKeyTemp)
			fmt.Println("result2", ok2)
			_ = result2
		}
		val := getVal(t, uri)

		fmt.Println("get-names returned ", val)
		if val != "ok" {
			t.Error("expected", "ok", "got", val)
		}
	}
	{
		command := "get option txt dummy2" // with subkey
		payload := command + "#" + strconv.FormatInt(time.Now().Unix(), 10)
		nonceStr := "07l70hfs8765r6z792dbf5j0" // tokens.GetRandomB36String()
		nonce := new([24]byte)
		copy(nonce[:], nonceStr[:])

		buffer := make([]byte, 0, (len(payload) + box.Overhead))
		sealed := box.Seal(buffer, []byte(payload), nonce, ce.PublicKeyTemp, &privk)

		fmt.Println("sealed", base64.RawURLEncoding.EncodeToString(sealed))
		fmt.Println("devicePublicKey", base64.RawURLEncoding.EncodeToString(ce.PublicKeyTemp[:]))
		fmt.Println("privk", base64.RawURLEncoding.EncodeToString(privk[:]))
		fmt.Println("nonce", nonceStr)

		// sign it
		uri := "http://knotfree.com:8085/api1/nameService?cmd=" + strings.ReplaceAll(command, " ", "%20")
		uri += "&sealed=" + base64.RawURLEncoding.EncodeToString(sealed)
		uri += "&nonce=" + nonceStr
		uri += "&pubk=" + pubkStr // the owner pub key of the name
		uri += "&name=" + aName

		{
			pubkBytes, err := base64.RawURLEncoding.DecodeString(string(pubkStr))
			_ = err
			var nonce2 [24]byte
			copy(nonce2[:], nonceStr)
			var pubk2 [32]byte
			copy(pubk2[:], pubkBytes)

			out := make([]byte, 0, len(sealed)) // it's actually smaller

			result2, ok2 := box.Open(out, sealed, &nonce2, &pubk2, ce.PrivateKeyTemp)
			fmt.Println("result2", ok2)
			_ = result2
		}
		val := getVal(t, uri)

		want := "dummy2Val" + randStr
		fmt.Println("get-names returned ", val)
		if val != want {
			t.Error("expected", want, "got", val)
		}
	}
}

func TestNameApiGetOption(t *testing.T) {

	ce := makeClusterWithServiceContact()
	iot.InitMongEnv()
	iot.InitIotTables()

	sc := ce.PacketService
	_ = sc
	fmt.Println("devicePublicKey", base64.URLEncoding.EncodeToString(ce.PublicKeyTemp[:]))

	passphrase := "a-person-passphrase"
	pubk, privk := tokens.GetBoxKeyPairFromPassphrase(passphrase)
	pubkStr := base64.URLEncoding.EncodeToString(pubk[:])
	pubkStr = strings.TrimRight(pubkStr, "=")
	fmt.Println("pubkStr", pubkStr)
	fmt.Println("privkStr", base64.URLEncoding.EncodeToString(privk[:]))

	token := tokens.GetTest32xTokenwjwtid(pubkStr, "f9boplwyb2wxsjtkspnucrth")
	payload, err := tokens.ValidateToken(string(token))
	fmt.Println("payload", payload, err)

	aName := "a-person-channel_pod" // has complicated subkeys
	{
		command := "get option A" // plain, no subkey
		payload := command + "#" + strconv.FormatInt(time.Now().Unix(), 10)
		nonceStr := "07l70hfs8765r6z792dbf5j0" // tokens.GetRandomB36String()
		nonce := new([24]byte)
		copy(nonce[:], nonceStr[:])

		buffer := make([]byte, 0, (len(payload) + box.Overhead))
		sealed := box.Seal(buffer, []byte(payload), nonce, ce.PublicKeyTemp, &privk)

		fmt.Println("sealed", base64.RawURLEncoding.EncodeToString(sealed))
		fmt.Println("devicePublicKey", base64.RawURLEncoding.EncodeToString(ce.PublicKeyTemp[:]))
		fmt.Println("privk", base64.RawURLEncoding.EncodeToString(privk[:]))
		fmt.Println("nonce", nonceStr)

		// sign it
		uri := "http://knotfree.com:8085/api1/nameService?cmd=" + strings.ReplaceAll(command, " ", "%20")
		uri += "&sealed=" + base64.RawURLEncoding.EncodeToString(sealed)
		uri += "&nonce=" + nonceStr
		uri += "&pubk=" + pubkStr // the owner pub key of the name
		uri += "&name=" + aName

		{
			pubkBytes, err := base64.RawURLEncoding.DecodeString(string(pubkStr))
			_ = err
			var nonce2 [24]byte
			copy(nonce2[:], nonceStr)
			var pubk2 [32]byte
			copy(pubk2[:], pubkBytes)

			out := make([]byte, 0, len(sealed)) // it's actually smaller

			result2, ok2 := box.Open(out, sealed, &nonce2, &pubk2, ce.PrivateKeyTemp)
			fmt.Println("result2", ok2)
			_ = result2
		}
		val := getVal(t, uri)

		fmt.Println("get-names returned ", val)
		if val != "216.128.128.195" {
			t.Error("expected", aName, "got", "216.128.128.195")
		}
	}
	{
		command := "get option txt dummy" // with subkey
		payload := command + "#" + strconv.FormatInt(time.Now().Unix(), 10)
		nonceStr := "07l70hfs8765r6z792dbf5j0" // tokens.GetRandomB36String()
		nonce := new([24]byte)
		copy(nonce[:], nonceStr[:])

		buffer := make([]byte, 0, (len(payload) + box.Overhead))
		sealed := box.Seal(buffer, []byte(payload), nonce, ce.PublicKeyTemp, &privk)

		fmt.Println("sealed", base64.RawURLEncoding.EncodeToString(sealed))
		fmt.Println("devicePublicKey", base64.RawURLEncoding.EncodeToString(ce.PublicKeyTemp[:]))
		fmt.Println("privk", base64.RawURLEncoding.EncodeToString(privk[:]))
		fmt.Println("nonce", nonceStr)

		// sign it
		uri := "http://knotfree.com:8085/api1/nameService?cmd=" + strings.ReplaceAll(command, " ", "%20")
		uri += "&sealed=" + base64.RawURLEncoding.EncodeToString(sealed)
		uri += "&nonce=" + nonceStr
		uri += "&pubk=" + pubkStr // the owner pub key of the name
		uri += "&name=" + aName

		{
			pubkBytes, err := base64.RawURLEncoding.DecodeString(string(pubkStr))
			_ = err
			var nonce2 [24]byte
			copy(nonce2[:], nonceStr)
			var pubk2 [32]byte
			copy(pubk2[:], pubkBytes)

			out := make([]byte, 0, len(sealed)) // it's actually smaller

			result2, ok2 := box.Open(out, sealed, &nonce2, &pubk2, ce.PrivateKeyTemp)
			fmt.Println("result2", ok2)
			_ = result2
		}
		val := getVal(t, uri)

		fmt.Println("get-names returned ", val)
		if val != "dummy" {
			t.Error("expected", "dummy", "got", val)
		}
	}
	{
		command := "get txt bitcoin" // with subkey
		payload := command + "#" + strconv.FormatInt(time.Now().Unix(), 10)
		nonceStr := "07l70hfs8765r6z792dbf5j0" // tokens.GetRandomB36String()
		nonce := new([24]byte)
		copy(nonce[:], nonceStr[:])

		buffer := make([]byte, 0, (len(payload) + box.Overhead))
		sealed := box.Seal(buffer, []byte(payload), nonce, ce.PublicKeyTemp, &privk)

		fmt.Println("sealed", base64.RawURLEncoding.EncodeToString(sealed))
		fmt.Println("devicePublicKey", base64.RawURLEncoding.EncodeToString(ce.PublicKeyTemp[:]))
		fmt.Println("privk", base64.RawURLEncoding.EncodeToString(privk[:]))
		fmt.Println("nonce", nonceStr)

		// sign it
		uri := "http://knotfree.com:8085/api1/nameService?cmd=" + strings.ReplaceAll(command, " ", "%20")
		uri += "&sealed=" + base64.RawURLEncoding.EncodeToString(sealed)
		uri += "&nonce=" + nonceStr
		uri += "&pubk=" + pubkStr // the owner pub key of the name
		uri += "&name=" + aName

		{
			pubkBytes, err := base64.RawURLEncoding.DecodeString(string(pubkStr))
			_ = err
			var nonce2 [24]byte
			copy(nonce2[:], nonceStr)
			var pubk2 [32]byte
			copy(pubk2[:], pubkBytes)

			out := make([]byte, 0, len(sealed)) // it's actually smaller

			result2, ok2 := box.Open(out, sealed, &nonce2, &pubk2, ce.PrivateKeyTemp)
			fmt.Println("result2", ok2)
			_ = result2
		}
		val := getVal(t, uri)

		fmt.Println("get-names returned ", val)
		if val != "3G2cGahYrRNXWbUQLtCaF8joHD3VJyS7hr" {
			t.Error("expected", aName, "got", "3G2cGahYrRNXWbUQLtCaF8joHD3VJyS7hr")
		}
	}
}

// TestNameApiDetails tests the name api with the 'details' command.
func TestNameApiDetails(t *testing.T) {

	ce := makeClusterWithServiceContact()
	iot.InitMongEnv()
	iot.InitIotTables()

	sc := ce.PacketService
	_ = sc
	fmt.Println("devicePublicKey", base64.URLEncoding.EncodeToString(ce.PublicKeyTemp[:]))

	passphrase := "a-person-passphrase"
	pubk, privk := tokens.GetBoxKeyPairFromPassphrase(passphrase)
	pubkStr := base64.URLEncoding.EncodeToString(pubk[:])
	pubkStr = strings.TrimRight(pubkStr, "=")
	fmt.Println("pubkStr", pubkStr)
	fmt.Println("privkStr", base64.URLEncoding.EncodeToString(privk[:]))

	token := tokens.GetTest32xTokenwjwtid(pubkStr, "f9boplwyb2wxsjtkspnucrth")
	payload, err := tokens.ValidateToken(string(token))
	fmt.Println("payload", payload, err)

	aName := "get-unix-time"
	//
	{

		command := "details"
		payload := command + "#" + strconv.FormatInt(time.Now().Unix(), 10)
		nonceStr := "07l70hfs8765r6z792dbf5j0" // tokens.GetRandomB36String()
		nonce := new([24]byte)
		copy(nonce[:], nonceStr[:])

		buffer := make([]byte, 0, (len(payload) + box.Overhead))
		sealed := box.Seal(buffer, []byte(payload), nonce, ce.PublicKeyTemp, &privk)

		fmt.Println("sealed", base64.RawURLEncoding.EncodeToString(sealed))
		fmt.Println("devicePublicKey", base64.RawURLEncoding.EncodeToString(ce.PublicKeyTemp[:]))
		fmt.Println("privk", base64.RawURLEncoding.EncodeToString(privk[:]))
		fmt.Println("nonce", nonceStr)

		// sign it
		uri := "http://knotfree.com:8085/api1/nameService?cmd=" + command
		uri += "&sealed=" + base64.RawURLEncoding.EncodeToString(sealed)
		uri += "&nonce=" + nonceStr
		uri += "&pubk=" + pubkStr // the owner pub key of the name
		uri += "&name=" + aName

		{
			pubkBytes, err := base64.RawURLEncoding.DecodeString(string(pubkStr))
			_ = err
			var nonce2 [24]byte
			copy(nonce2[:], nonceStr)
			var pubk2 [32]byte
			copy(pubk2[:], pubkBytes)

			out := make([]byte, 0, len(sealed)) // it's actually smaller

			result2, ok2 := box.Open(out, sealed, &nonce2, &pubk2, ce.PrivateKeyTemp)
			fmt.Println("result2", ok2)
			_ = result2
		}
		val := getVal(t, uri)

		fmt.Println("get-names returned ", val)
		// now, decode the response
		// it's a watchedTopic
		topic := iot.WatchedTopic{}
		json.Unmarshal([]byte(val), &topic)
		if topic.NameStr != aName {
			t.Error("expected", aName, "got", topic.Name)
		}
	}
}

func TestNameApiAddName(t *testing.T) {

	ce := makeClusterWithServiceContact()
	iot.InitMongEnv()
	iot.InitIotTables()

	sc := ce.PacketService
	devicePublicKey := ce.PublicKeyTemp
	_ = devicePublicKey
	_ = sc
	fmt.Println("devicePublicKey", base64.URLEncoding.EncodeToString(devicePublicKey[:]))

	passphrase := "a-person-passphrase"
	pubk, privk := tokens.GetBoxKeyPairFromPassphrase(passphrase)
	pubkStr := base64.URLEncoding.EncodeToString(pubk[:])
	pubkStr = strings.TrimRight(pubkStr, "=")
	fmt.Println("pubkStr", pubkStr)
	fmt.Println("privkStr", base64.URLEncoding.EncodeToString(privk[:]))

	token := tokens.GetTest32xTokenwjwtid(pubkStr, "f9boplwyb2wxsjtkspnucrth")
	payload, err := tokens.ValidateToken(string(token))
	fmt.Println("payload", payload, err)

	// add a new name with web-api
	{
		newName := "test-new-name"
		command := "reserve " + newName + " " + string(token)
		payload := command + "#" + strconv.FormatInt(time.Now().Unix(), 10)
		nonceStr := tokens.GetRandomB36String()
		nonce := new([24]byte)
		copy(nonce[:], nonceStr[:])

		buffer := make([]byte, 0, (len(payload) + box.Overhead))
		sealed := box.Seal(buffer, []byte(payload), nonce, devicePublicKey, &privk)

		fmt.Println("sealed", base64.RawURLEncoding.EncodeToString(sealed))
		fmt.Println("devicePublicKey", base64.RawURLEncoding.EncodeToString(devicePublicKey[:]))
		fmt.Println("privk", base64.RawURLEncoding.EncodeToString(privk[:]))
		fmt.Println("nonce", nonceStr)

		// sign it
		uri := "http://knotfree.com:8085/api1/nameService?"
		uri += "&sealed=" + base64.RawURLEncoding.EncodeToString(sealed)
		uri += "&cmd=" + strings.ReplaceAll(command, " ", "%20")
		uri += "&nonce=" + nonceStr
		uri += "&pubk=" + pubkStr // the owner pub key of the name
		uri += "&name=" + newName
		val := getVal(t, uri)

		fmt.Println("get-names returned ", val)

		if val != "ok" {
			t.Error("expected ok, got", val)
		}
	}
}

func TestNameApiList(t *testing.T) {

	ce := makeClusterWithServiceContact()
	iot.InitMongEnv()
	iot.InitIotTables()

	sc := ce.PacketService
	devicePublicKey := ce.PublicKeyTemp
	_ = devicePublicKey
	_ = sc
	fmt.Println("devicePublicKey", base64.URLEncoding.EncodeToString(devicePublicKey[:]))

	passphrase := "a-person-passphrase"
	pubk, privk := tokens.GetBoxKeyPairFromPassphrase(passphrase)
	pubkStr := base64.URLEncoding.EncodeToString(pubk[:])
	pubkStr = strings.TrimRight(pubkStr, "=")
	fmt.Println("pubkStr", pubkStr)
	fmt.Println("privkStr", base64.URLEncoding.EncodeToString(privk[:]))

	// get a list of names for a user pubk
	{ // a dns lookup with iot name like get-unix-time.iot

		payload := pubkStr + "#" + strconv.FormatInt(time.Now().Unix(), 10)
		nonceStr := tokens.GetRandomB36String()
		nonce := new([24]byte)
		copy(nonce[:], nonceStr[:])

		buffer := make([]byte, 0, (len(payload) + box.Overhead))
		sealed := box.Seal(buffer, []byte(payload), nonce, devicePublicKey, &privk)

		fmt.Println("sealed", base64.RawURLEncoding.EncodeToString(sealed))
		fmt.Println("devicePublicKey", base64.RawURLEncoding.EncodeToString(devicePublicKey[:]))
		fmt.Println("privk", base64.RawURLEncoding.EncodeToString(privk[:]))
		fmt.Println("nonce", nonceStr)

		// sign it
		uri := "http://knotfree.com:8085/api1/getNames?cmd=" + base64.RawURLEncoding.EncodeToString(sealed)
		uri += "&nonce=" + nonceStr
		uri += "&pubk=" + pubkStr
		val := getVal(t, uri)

		fmt.Println("get-names returned ", len(val))

		valBin, err := base64.RawURLEncoding.DecodeString(val)
		if err != nil {
			fmt.Println("failed to deocde", err)
			t.Error("failed to decode", err)
		}
		decrypted, ok := box.Open(nil, valBin, nonce, devicePublicKey, &privk)
		if !ok {
			fmt.Println("failed to open")
			t.Error("failed to decode", err)
		}

		fmt.Println("get-names returned ", len(string(decrypted)))
		// fmt.Println("get-names returned ", string(decrypted))
		// now, decode the response

		// now decode

	}

}

func getVal(t *testing.T, url string) string {
	resp, err := http.Get(url)
	assert.Nil(t, err)
	if err != nil {
		fmt.Println("http.Get err", err)
		return err.Error()
	}
	assert.Equal(t, 200, resp.StatusCode)
	defer resp.Body.Close()
	resBody, err := io.ReadAll(resp.Body)
	assert.Nil(t, err)
	if err != nil {
		return ""
	}
	return string(resBody)
}

func TestUrl(t *testing.T) {

	ce := makeClusterWithServiceContact()

	iot.InitMongEnv()
	iot.InitIotTables()

	// note: the .com and .test tlds are in /etc/hosts

	{ // a dns lookup with iot name like get-unix-time.iot
		val := getVal(t, "http://get.option.a.get-unix-time.test:8085") // note: the .com and .test tlds must be in /etc/hosts
		fmt.Println("get.option.a", val)
		assert.Equal(t, val, "216.128.128.195")
	}

	{ // a regular api call
		val := getVal(t, "http://knotlocal.com:8085/api1/getPublicKey")
		fmt.Println("getPublicKey", val)
		sss := base64.RawURLEncoding.EncodeToString(ce.PublicKeyTemp[:])
		assert.Equal(t, val, sss) //"-muxcABH_pTsuNqT3yaYfQj-3krwM6XmEu47vTZLSHM")
	}
	startAServer("get-unix-time", "")     // start a thing server
	startAServer("get-unix-time_iot", "") // start a thing server
	{                                     // a device call
		val := getVal(t, "http://get-unix-time.knotlocal.com:8085/get/pubk")
		fmt.Println("pubk", val)
		assert.Equal(t, val, "bht-Ka3j7GKuMFOablMlQnABnBvBeugvSf4CdFV3LXs")
	}
	{ // a device call
		val := getVal(t, "http://get-unix-time_iot.knotlocal.com:8085/get/pubk")
		fmt.Println("pubk", val)
		assert.Equal(t, val, "bht-Ka3j7GKuMFOablMlQnABnBvBeugvSf4CdFV3LXs")
	}

	{ // a device call with iot name like get-unix-time.iot
		val := getVal(t, "http://get-unix-time.test:8085/get/pubk") // note: the .com and .test tlds are in /etc/hosts
		fmt.Println("pubk", val)
		assert.Equal(t, val, "bht-Ka3j7GKuMFOablMlQnABnBvBeugvSf4CdFV3LXs")
	}

}

func TestStringToMap(t *testing.T) {
	str := "a b c d"
	m := iot.StringToMap(str)
	fmt.Println("m", m)
	assert.Equal(t, m["a"], "b")
	assert.Equal(t, m["c"], "d")

	str = "a"
	m = iot.StringToMap(str)
	fmt.Println("m", m)
	assert.Equal(t, m["@"], "a")
}

func TestMapToString(t *testing.T) {
	m := make(map[string]string)
	m["a"] = "b"
	m["c"] = "d"
	str := iot.MapToString(m)
	fmt.Println("str", str)
	assert.Equal(t, str, "a b c d")
}
