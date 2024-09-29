package iot_test

import (
	"encoding/base64"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/awootton/knotfreeiot/iot"
	"github.com/awootton/knotfreeiot/packets"
	"github.com/awootton/knotfreeiot/tokens"
	"golang.org/x/crypto/nacl/box"
)

// TestReserveNames is not really a test, it's a script to reserve names.
// Using the look api.
func XxxxTestReserveNames(t *testing.T) {

	iot.InitMongEnv()
	iot.InitIotTables()

	ce := makeClusterWithServiceContact()
	sc := ce.PacketService
	_ = sc

	devicePublicKey := ce.PublicKeyTemp
	devicePublicKeyStr := base64.URLEncoding.EncodeToString(devicePublicKey[:])
	devicePublicKeyStr = strings.TrimRight(devicePublicKeyStr, "=")
	_ = devicePublicKeyStr

	// get the user dicrectory

	homeDir, err := os.UserHomeDir()
	check(err)

	passPhrase, err := os.ReadFile(homeDir + "/atw_private/passphrase.txt")
	check(err)
	domainList, err := os.ReadFile(homeDir + "/atw_private/domainList.txt")
	check(err)

	// fmt.Println("passPhrase", passPhrase)
	// fmt.Println("domainList", domainList)

	pubk, privk := tokens.GetBoxKeyPairFromPassphrase(string(passPhrase))
	pubkStr := base64.URLEncoding.EncodeToString(pubk[:])
	pubkStr = strings.TrimRight(pubkStr, "=")
	_ = privk
	fmt.Println("pubkStr", pubkStr)

	// keep using the same jwtid as before
	token, payload := tokens.GetImpromptuGiantTokenLocal(pubkStr, "plfdfo4ezlgclcumjtqkiwre")
	_ = token
	_ = payload

	names := strings.Split(string(domainList), "\n")
	for _, name := range names {
		name = strings.TrimSpace(name)
		fmt.Println("Reserving", name)

		nonceStr := []byte(tokens.GetRandomB36String())
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
}
