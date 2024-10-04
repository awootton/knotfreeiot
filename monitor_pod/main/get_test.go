package main

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/awootton/knotfreeiot/monitor_pod"
	"github.com/awootton/knotfreeiot/tokens"
	"github.com/stretchr/testify/assert"
)

// TestGetIotResponseReno will only work if a certain device is alive.
// using production eg. http://demo-small-window-allow-should-engine.knotfree.io/help
func TestGetIotResponseReno(t *testing.T) {

	// read passphrase from ~/atw/renoIotpass.txt
	home, _ := os.UserHomeDir()
	tmp, err := os.ReadFile(home + "/atw/renoIotpass.txt")
	if err != nil {
		fmt.Println("TestGetIotResponseReno err", err)
		t.Fail()
	}

	c := monitor_pod.ThingContext{}

	c.Password = "demo-Device"

	pubk, privk := tokens.GetBoxKeyPairFromPassphrase(c.Password)
	c.PubStr = base64.RawURLEncoding.EncodeToString(pubk[:])
	c.PrivStr = base64.RawURLEncoding.EncodeToString(privk[:])
	_ = c.PrivStr
	c.PubStr = "iP8H8BJAvNsac3rI2SFXvGiHmDqZV3vxFFLEWE-8bnE"

	// c.PrivStr = base64.RawURLEncoding.EncodeToString(privk[:])

	adminPassPhrase := strings.TrimSpace(string(tmp))
	pubk, privk = tokens.GetBoxKeyPairFromPassphrase(adminPassPhrase)
	c.AdminPubStr = base64.RawURLEncoding.EncodeToString(pubk[:])
	c.AdminPrivStr = base64.RawURLEncoding.EncodeToString(privk[:])

	server := "knotfree.io"
	thing := "demo-small-window-allow-should-engine"
	cmd := "get f"

	r := monitor_pod.GetIotResponse(server, thing, cmd, c.PubStr, c.AdminPrivStr, c.AdminPubStr)

	fmt.Println("TestGetIotResponseReno r", r)

	// depends on the weather assert.Equal(t, "v0.1.5", r)
}

// and the localhost is set to knotfree.com and is running on port 8085
func TestGetIotResponseBoxed(t *testing.T) {

	c := monitor_pod.ThingContext{}
	monitor_pod.SetupKeys(&c)

	server := "knotfree.com:8085"
	thing := "get-unix-time_iot"
	cmd := "version"

	r := monitor_pod.GetIotResponse(server, thing, cmd, c.PubStr, c.AdminPrivStr, c.AdminPubStr)

	fmt.Println("TestGetIotResponseBoxed r", r)

	assert.Equal(t, "v0.1.5", r)

}

// and the localhost is set to knotfree.com and is running on port 8085
func TestGetIotResponse(t *testing.T) {

	server := "knotfree.com:8085"
	thing := "get-unix-time_iot"
	cmd := "get pubk"
	pubk := ""
	adminprivk := ""
	r := monitor_pod.GetIotResponse(server, thing, cmd, pubk, adminprivk, "")

	// fmt.Println("TestGetFromThing r", r)

	assert.Equal(t, "bht-Ka3j7GKuMFOablMlQnABnBvBeugvSf4CdFV3LXs", r)

}
