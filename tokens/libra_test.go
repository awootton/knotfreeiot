package tokens_test

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"testing"
	"time"

	"github.com/the729/go-libra/client"
)

//const starttime = uint32(1577840400) // Wednesday, January 1, 2020 1:00:00 AM

const (
	defaultServer = "libra.libra:53110"
	waypoint      = "insecure" //0:bf7e1eef81af68cc6b4801c3739da6029c778a72e67118a8adf0dd759f188908"
)

func xxTestLibra2(t *testing.T) {

	if os.Getenv("KUBE_EDITOR") == "atom --wait" {
	} else {
		return // don't run this on github
	}

	c, err := client.New(defaultServer, waypoint)
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	// from alice to bob

	aliceAddrStr := "1ad7d1096dd7f127fb25315b7e6f9c619e1e350c9b03d81f8edacd63b5c640de"
	_ = aliceAddrStr

	bobAddrStr := "18b553473df736e5e363e7214bd624735ca66ac22a7048e3295c9b9b9adfc26a"
	_ = bobAddrStr

	balance, err := GetBalance(c, aliceAddrStr)

	log.Println("alice balance is ", balance)

	balance, err = GetBalance(c, bobAddrStr)

	log.Println("bob balance is ", balance)

	// this is NOT the real master account.
	masterSource := "e1a11f42f719c241b715c34e9bc7a5fb3da6787eed0625cc288278ed7f9e528a"
	balance, err = GetBalance(c, masterSource)
	log.Println("e1a11 is ", balance)

	masterPub, masterPriv := ReadMasterKey("master_fee_source.txt")

	// fmt.Println(masterPub)
	// fmt.Println(masterPriv)

	bpub, _ := hex.DecodeString(masterPub)
	masterAccountAddr := client.PubkeyMustToAddress(bpub)
	masterAccountHex := hex.EncodeToString(masterAccountAddr[:])

	fmt.Println("master account number ", masterAccountHex) // b3f902528cfeb475b257dcc808e8cb091d7038d3c50f008100ff73e14f011bb1

	balance, err = GetBalance(c, masterAccountHex)

	log.Println("master source balance is ", balance)

	// genpublic, genprivate, _ := ed25519.GenerateKey(rand.Reader)
	// fmt.Println(hex.EncodeToString(genpublic))
	// fmt.Println(hex.EncodeToString(genprivate))

	// Transaction parameters
	senderAddr := client.MustToAddress(masterAccountHex)
	priKeyBytes, _ := hex.DecodeString(masterPriv)
	priKey := ed25519.PrivateKey(priKeyBytes)

	//senderAddr = client.PubkeyMustToAddress(genpublic)
	//priKey = genprivate

	recvAddr := client.MustToAddress(bobAddrStr)
	amountMicro := uint64(1000)
	maxGasAmount := uint64(500000)
	gasUnitPrice := uint64(0)
	expiration := time.Now().Add(1 * time.Minute)

	log.Printf("Get current account sequence of sender...")
	seq, err := c.QueryAccountSequenceNumber(context.TODO(), senderAddr)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("... is %d", seq)

	rawTxn, err := client.NewRawP2PTransaction(
		senderAddr, recvAddr, seq,
		amountMicro, maxGasAmount, gasUnitPrice, expiration,
	)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Submit transaction...")
	expectedSeq, err := c.SubmitRawTransaction(context.TODO(), rawTxn, priKey)
	if err != nil {
		log.Fatal("submit error", err)
	}

	log.Printf("Waiting until transaction is included in ledger...")
	err = c.PollSequenceUntil(context.TODO(), senderAddr, expectedSeq, expiration)
	if err != nil {
		log.Fatal(err)
	}

	// now what is bob ?

	_ = err
}

func xxTestLibra1(t *testing.T) {

	if os.Getenv("KUBE_EDITOR") == "atom --wait" {
	} else {
		return
	}

	// assume that libra on port 8000
	// eg.
	// kubectl -n libra port-forward libra-0 8000:8000
	// also, put 127.0.0.1 libra.libra in /etc/hosts

	// check it with mbp-atw-2:libra awootton$ cargo run --bin cli -- -a libra.libra -p 8000 -m "/Users/awootton/Documents/workspace/libra-statefulset/tmp/mint.key"

	c, err := client.New(defaultServer, waypoint)
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	aliceAddrStr := "1ad7d1096dd7f127fb25315b7e6f9c619e1e350c9b03d81f8edacd63b5c640de"
	_ = aliceAddrStr

	bobAddrStr := "18b553473df736e5e363e7214bd624735ca66ac22a7048e3295c9b9b9adfc26a"
	addr := client.MustToAddress(bobAddrStr)

	provenState, err := c.QueryAccountState(context.TODO(), addr)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("provenState address", provenState.GetAddress())
	fmt.Println("provenState GetLedgerInfo ", provenState.GetLedgerInfo())

	addr = client.MustToAddress(aliceAddrStr)
	provenState, err = c.QueryAccountState(context.TODO(), addr)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("provenState alice address", provenState.GetAddress())
	fmt.Println("provenState alice GetLedgerInfo ", provenState.GetLedgerInfo())

	// balance???

	resource, err := provenState.GetAccountBlob().GetLibraAccountResource()
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Balance (microLibra): %d", resource.GetBalance())
	log.Printf("Sequence Number: %d", resource.GetSequenceNumber())
	log.Printf("SentEventsCount: %d", resource.GetSentEvents().Count)
	log.Printf("    Key: %x", resource.GetSentEvents().Key)
	log.Printf("ReceivedEventsCount: %d", resource.GetReceivedEvents().Count)
	log.Printf("    Key: %x", resource.GetReceivedEvents().Key)
	log.Printf("DelegatedWithdrawalCapability: %v", resource.GetDelegatedWithdrawalCapability())
	log.Printf("Authentication key: %v", hex.EncodeToString(resource.GetAuthenticationKey()))
	log.Printf("Event generator: %v", resource.GetEventGenerator())

	balance, err := GetBalance(c, aliceAddrStr)

	log.Println("alice balance is ", balance)

	balance, err = GetBalance(c, bobAddrStr)

	log.Println("bob balance is ", balance)

}

// returns public key and private key in hex.
// the public key is first in the input
func ReadMasterKey(name string) (string, string) {

	home, _ := os.UserHomeDir()
	path := home + "/atw/" + name
	bin, err := ioutil.ReadFile(path)
	if err != nil {
		return "", ""
	}
	str := string(bin)
	pub := ""
	priv := ""
	// scan for a block of 64 hex and a block of 128 hex.
	hexrun := -999999
	for index, runeValue := range str {
		if index-hexrun == 64 && pub == "" {
			pub = str[hexrun:index]
		}
		if index-hexrun >= 128 && index-hexrun < 999999 {
			priv = str[hexrun:index]
		}
		if isHex(runeValue) == false {
			hexrun = -999999
		} else if hexrun < 0 {
			hexrun = index
		}
	}
	return pub, priv
}

func isHex(r rune) bool {
	if int(r) >= '0' && int(r) <= '9' {
		return true
	}
	if int(r) >= 'a' && int(r) <= 'f' {
		return true
	}
	if int(r) >= 'A' && int(r) <= 'F' {
		return true
	}
	return false
}

// GetBalance tries to get the account balance for the account.
func GetBalance(c *client.Client, accountHex string) (uint64, error) {
	if len(accountHex) != 64 {
		if len(accountHex) == 128 {
			// maybe it's a private key
			accountHex = accountHex[64:]
		} else {
			return -0, errors.New("expected 64 hex chars")
		}
	}

	addr := client.MustToAddress(accountHex)
	provenState, err := c.QueryAccountState(context.TODO(), addr)
	if err != nil {
		return 0, err
	}
	blob := provenState.GetAccountBlob()
	if blob == nil {
		return 0, errors.New("nil blob ")
	}
	resource, err := blob.GetLibraAccountResource()
	if err != nil {
		return 0, err
	}

	return resource.GetBalance(), nil

}
