package tokens_test

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/awootton/knotfreeiot/tokens"
)

const starttime = uint32(1577840400) // Wednesday, January 1, 2020 1:00:00 AM

func TestCalcTokenPrice(t *testing.T) {

	// := GetSampleTokenPayload(starttime) // is TinyX2 for 2 connections

	price := tokens.GetTokenStatsAndPrice(tokens.TinyX2)

	fmt.Println("TinyX2 cost is ", price.Price)

}

func TestCalcTokenPrice_TinyX4(t *testing.T) {

	// is TinyX4 for 4 connections which is our statdard free token

	price := tokens.GetTokenStatsAndPrice(tokens.TinyX4)

	fmt.Println("TinyX4 cost is ", price.Price)
	fmt.Println("TinyX4 for a year is ", price.Price*12) // 0.048 or about 50 cents per decade.
}

func fixme_TestMakeReservation(t *testing.T) {

	tokens.LoadPublicKeys()
	tokens.LoadPrivateKeys("~/atw/privateKeys4.txt")

	sampleToken := `[Free_token_expires:_2021-12-31,{exp:1641023999,iss:_9sh,jti:HpifIJkhgnTOGc3EDmOJaV0A,in:32,out:32,su:4,co:2,url:knotfree.net},eyJhbGciOiJFZDI1NTE5IiwidHlwIjoiSldUIn0.eyJleHAiOjE2NDEwMjM5OTksImlzcyI6Il85c2giLCJqdGkiOiJIcGlmSUpraGduVE9HYzNFRG1PSmFWMEEiLCJpbiI6MzIsIm91dCI6MzIsInN1Ijo0LCJjbyI6MiwidXJsIjoia25vdGZyZWUubmV0In0.YSo2Ur7lbkwTPZfQymyvy4N1mWQaUn_cziwK36kTKlASgqOReHQ4FAocVvgq7ogbPWB1hD4hNoJtCg2WWq-BCg]`
	trimmedToken, issuer, err := tokens.GetKnotFreePayload(string(sampleToken))
	if err != nil {
		t.Errorf("got %v, want %v", "nil", "something else")
	}
	publicKeyBytes := tokens.FindPublicKey(issuer)
	// find the public key that matches.
	if len(publicKeyBytes) != 32 {
		t.Errorf("got %v, want %v", "nil", "something else")
	}
	foundPayload, ok := tokens.VerifyToken([]byte(trimmedToken), []byte(publicKeyBytes))
	_ = ok
	fmt.Println("foundPayload is ", foundPayload)

	payload := &tokens.SubscriptionNameReservationPayload{}
	payload.ExpirationTime = starttime + 60*60*24*(365)
	payload.Issuer = "yRst"
	payload.JWTID = "thePublicKeyOfTheOwner"
	payload.Name = "dummy2"

	privKey := tokens.GetPrivateKey(payload.Issuer)
	tokBytes, err := tokens.MakeNameToken(payload, []byte(privKey))
	if err != nil {
		t.Errorf("got %v, want %v", "nil", "something else")
	}
	fmt.Println("tok bytes ", string(tokBytes))

	when := time.Unix(int64(payload.ExpirationTime), 0)
	year, month, day := when.Date()

	comments := make([]interface{}, 3)
	tmp := fmt.Sprintf(" expires: %v-%v-%v", year, int(month), day)
	comments[0] = "Reserve Name " + payload.Name + tmp
	comments[1] = payload
	comments[2] = string(tokBytes)
	returnval, err := json.Marshal(comments)
	returnval = []byte(strings.ReplaceAll(string(returnval), `"`, ``))
	returnval = []byte(strings.ReplaceAll(string(returnval), ` `, `_`))

	fmt.Println("final token ", string(returnval))

	// now, unpack it. and verify

	trimmedToken, issuer, err = tokens.GetKnotFreePayload(string(returnval))
	if err != nil {
		t.Errorf("got %v, want %v", "err", "something else")
	}
	// find the public key that matches.
	publicKeyBytes = tokens.FindPublicKey(issuer)
	if len(publicKeyBytes) != 32 {
		t.Errorf("got %v, want %v", "err", "something else")
	}
	namePayload, ok := tokens.VerifyNameToken([]byte(trimmedToken), []byte(publicKeyBytes))
	if !ok {
		t.Errorf("got %v, want %v", "false", "true")
	}
	fmt.Println("payload of name token ", namePayload)
	fmt.Println("payload of name token ", namePayload)
	fmt.Println("payload of name token ", namePayload)

	got := "ok"
	want := "ok"

	got = base64.RawURLEncoding.EncodeToString([]byte(tokens.FindPublicKey("yRst")))
	want = "yRst5ig1Zf1iYVvI0q0LltjU8gmT-9ZZBKWijosq2Vg"
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestFindsrr(t *testing.T) {

	tokens.LoadPublicKeys()

	got := "ok"
	want := "ok"

	got = base64.RawURLEncoding.EncodeToString([]byte(tokens.FindPublicKey("yRst")))
	want = "yRst5ig1Zf1iYVvI0q0LltjU8gmT-9ZZBKWijosq2Vg"
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}
