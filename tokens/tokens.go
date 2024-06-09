// Copyright 2019,2020,2021 Alan Tracey Wootton
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

// Package tokens comments. TODO: package comments for these crypto utils. ed25519 jwt.
package tokens

import (
	"crypto/ed25519"
	rand "crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	mathrand "math/rand"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/awootton/knotfreeiot/badjson"
	"github.com/gbrlsnchs/jwt/v3"
	"golang.org/x/crypto/curve25519"
)

var aTest32xToken = []byte("")

func GetTest32xToken() []byte {
	if len(aTest32xToken) != 0 {
		return aTest32xToken
	}
	LoadPrivateKeys("~/atw/privateKeys4.txt")

	payload := GetSampleTokenFromStats(uint32(time.Now().Unix()), "knotfree.dog:8085/mqtt", GetTokenStatsAndPrice(Medium).Stats) // is localhost in my /etc/hosts

	signingKey := GetPrivateKeyWhole(0)
	bytes, err := MakeToken(payload, []byte(signingKey))
	_ = err
	aTest32xToken = bytes
	return bytes
}

// KnotFreeTokenPayload is our JWT 'claims'.
type KnotFreeTokenPayload struct {
	//
	ExpirationTime uint32 `json:"exp,omitempty"` // unix seconds
	Issuer         string `json:"iss"`           // first 4 bytes (or more) of base64 public key of issuer
	JWTID          string `json:"jti,omitempty"` // a unique serial number for this Issuer

	KnotFreeContactStats // limits on what we're allowed to do.

	URL string `json:"url"` // address of the service eg. "knotfree.net" or knotfree0.com for localhost
}

// KnotFreeContactStats is the numeric part of the token claims
// it is floats to and fractions in json
type KnotFreeContactStats struct {
	//
	Input         float64 `json:"in"`  // bytes per sec float32
	Output        float64 `json:"out"` // bytes per sec
	Subscriptions float64 `json:"su"`  // seconds per sec
	Connections   float64 `json:"co"`  // seconds per sec
}

// TokenRequest is created in javascript and sent as json.
type TokenRequest struct {
	//
	Pkey    string                `json:"pkey"` // a curve25519 pub key of caller
	Payload *KnotFreeTokenPayload `json:"payload"`
	Comment string                `json:"comment"`
}

// TokenReply is created here and boxed and sent back to js
type TokenReply struct {
	Pkey    string `json:"pkey"` // a curve25519 pub key of server
	Payload string `json:"payload"`
	Nonce   string `json:"nonce"`
}

// MakeToken is
func MakeToken(data *KnotFreeTokenPayload, privateKey []byte) ([]byte, error) {

	algo := jwt.NewEd25519(jwt.Ed25519PrivateKey(privateKey))
	token, err := jwt.Sign(data, algo)
	if err != nil {
		return []byte(""), err
	}
	return token, nil
}

// VerifyToken is
func VerifyToken(ticket []byte, publicKey []byte) (*KnotFreeTokenPayload, bool) {

	payload := KnotFreeTokenPayload{}

	algo := jwt.NewEd25519(jwt.Ed25519PublicKey(publicKey))
	hd, err := jwt.Verify([]byte(ticket), algo, &payload)
	if err != nil {
		return &KnotFreeTokenPayload{}, false
	}
	_ = hd
	return &payload, true
}

// just parseout the payload. Only for testing.
func ParseTokenNoVerify(toke []byte) *KnotFreeTokenPayload {

	payload := KnotFreeTokenPayload{}

	parts := strings.Split(string(toke), ".")
	if len(parts) != 3 {
		return &KnotFreeTokenPayload{}
	}
	claimsPlain, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return &KnotFreeTokenPayload{}
	}
	err = json.Unmarshal(claimsPlain, &payload)
	if err != nil {
		return &KnotFreeTokenPayload{}
	}
	return &payload
}

// SubscriptionNameReservationPayload is our JWT 'claims'.

type SubscriptionNameReservationPayload struct {
	//
	ExpirationTime uint32 `json:"exp,omitempty"` // unix seconds
	Issuer         string `json:"iss"`           // first 4 bytes (or more) of base64 public key of issuer
	JWTID          string `json:"jti,omitempty"` // a unique serial number for this Issuer. must be public key of user
	Name           string `json:"name"`          // the subscription name
}

// MakeNameToken is
func MakeNameToken(data *SubscriptionNameReservationPayload, privateKey []byte) ([]byte, error) {

	algo := jwt.NewEd25519(jwt.Ed25519PrivateKey(privateKey))
	token, err := jwt.Sign(data, algo)
	if err != nil {
		return []byte(""), err
	}
	return token, nil
}

// VerifyToken is
func VerifyNameToken(ticket []byte, publicKey []byte) (*SubscriptionNameReservationPayload, bool) {

	payload := SubscriptionNameReservationPayload{}

	algo := jwt.NewEd25519(jwt.Ed25519PublicKey(publicKey))
	hd, err := jwt.Verify([]byte(ticket), algo, &payload)
	if err != nil {
		return &SubscriptionNameReservationPayload{}, false
	}
	_ = hd
	// TODO: compare all the fields with limits.

	return &payload, true
}

// GetKnotFreePayload returns the trimmed token
// and the issuer. We allow all kinds of not b64 junk around our JWT's
// it is tolerant of junk before and after the token.
// Only return the issuer. Let Verify get the claims.
// yes, we end up unmarshaling KnotFreeTokenPayload twice.
func GetKnotFreePayload(token string) (string, string, error) {

	issuer := ""
	tokenStartIndex := 0
	tokenEndIndex := 0

	// part 1 eg eyJhbGciOiJFZERTQSIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2OTczNDU4OTMsImlzcyI6Il85c2giLCJqdGkiOiJCMEQxT1JoWGVjYnA0anVPSXZKcDQ5ajYiLCJpbiI6MTAwMDAwMCwib3V0IjoxMDAwMDAwLCJzdSI6MjAwMDAwLCJjbyI6MjAwMDAwLCJ1cmwiOiJrbm90ZnJlZS5uZXQifQ.SH47mr46105AL8wxfZkNB0iZMAc-MzpZ1hqzNz3lPa65R8XmR4TXNrzPz3aTVJd5PYXhgXmt0EubSvJB7mqADA
	// or     eyJhbGciOiJFZDI1NTE5IiwidHlwIjoiSldUIn0.
	// I don't like this: should just check the front. Too fragile.
	{
		firstPart := "eyJhbGciOiJFZERTQSIsInR5cCI6IkpXVCJ9."     // {"alg":"EdDSA","typ":"JWT"}
		firstPart2 := "eyJhbGciOiJFZDI1NTE5IiwidHlwIjoiSldUIn0." //{"alg":"Ed25519","typ":"JWT"}
		index := strings.Index(token, firstPart)
		flen := len(firstPart)
		if index < 0 {
			index = strings.Index(token, firstPart2)
			flen = len(firstPart2)
		}
		tokenStartIndex = index
		tokenEndIndex = index + flen
		if index < 0 {
			s := "expected eyJhbGciOiJFZERTQSIsInR5cCI6IkpXVCJ9. OR eyJhbGciOiJFZDI1NTE5IiwidHlwIjoiSldUIn0. got " + token
			return token, issuer, errors.New(s)
		}
	}
	// part 2
	{
		t := token[tokenEndIndex:]
		index := strings.Index(t, ".")
		if index < 0 {
			s := "expected . got " + token
			return token, issuer, errors.New(s)
		}
		part2 := token[tokenEndIndex : tokenEndIndex+index]
		claimsPlain, err := base64.RawURLEncoding.DecodeString(part2)
		if err != nil {
			return token, issuer, err
		}
		payload := KnotFreeTokenPayload{}
		err = json.Unmarshal(claimsPlain, &payload)
		if err != nil {
			return token, issuer, err
		}
		issuer = payload.Issuer
		tokenEndIndex += index + 1
	}
	// part 3
	// scan as b64
	// is it not always the same length? Why are we scanning?
	// TODO: just get indexof .
	for {
		if tokenEndIndex >= len(token) {
			break
		}
		//r, runeLength := utf8.DecodeRuneInString(token[tokenEndIndex:])
		r := token[tokenEndIndex]
		runeLength := 1
		if runeLength != 1 {
			break
		}
		if badjson.B64DecodeMap[r] == byte(0xFF) {
			break
		}
		tokenEndIndex += runeLength
	}

	trimmedToken := token[tokenStartIndex:tokenEndIndex]
	return trimmedToken, issuer, nil
}

// AllThePublicKeys is a globalservice with a list of public keys that
// can be used to verify tokens.
var allThePublicKeysInUniverse = make([]string, 0)
var allThePublicKeysInUniverseMux sync.Mutex

// SavePublicKey goes with FindPublicKey.
// We're using the first couple of bytes, in base54, of the 32byte public key as a name
// and looking them up in a gadget here.
// publicKey is actually an immutable array of bytes and not utf8. Is that going to be a problem?
func SavePublicKey(key string, publicKey string) {

	key = strings.ReplaceAll(key, "/", "_") // std to url encoding
	key = strings.ReplaceAll(key, "+", "-") // std to url encoding

	// publicKey is actually bytes and not a string

	if key == "1iVt" { // TODO: better black list
		return
	}
	if FindPublicKey(key) == publicKey {
		return
	}
	var prefixArr [43]byte
	n, err := decodeKey(key, prefixArr[:])
	if err != nil || n < 1 || n >= 32 {
		return
	}
	prefix := prefixArr[0:n]
	if !strings.HasPrefix(publicKey, string(prefix)) {
		return
	}
	allThePublicKeysInUniverseMux.Lock()
	defer allThePublicKeysInUniverseMux.Unlock()

	if len(publicKey) < 32 { // our keys are 32
		fmt.Println("fixme wtf key wrong len fatal ")
		return
	}
	allThePublicKeysInUniverse = append(allThePublicKeysInUniverse, publicKey)
	sort.Strings(allThePublicKeysInUniverse)
}

// FindPublicKey is
func FindPublicKey(thekey string) string {

	LoadPublicKeys()

	if thekey == "1iVt" { // TODO: make better blacklist
		return ""
	}

	var prefixArr [43]byte
	n, err := decodeKey(thekey, prefixArr[:])
	if err != nil || n < 1 || n >= 32 {
		return ""
	}
	prefix := prefixArr[0:n]
	allThePublicKeysInUniverseMux.Lock()
	defer allThePublicKeysInUniverseMux.Unlock()

	// for k, v := range allThePublicKeysInUniverse {
	// 	bytes := []byte(v)
	// 	fmt.Println(k)
	// 	fmt.Println(bytes)
	// }

	foundi := sort.Search(len(allThePublicKeysInUniverse), func(i int) bool {
		item := allThePublicKeysInUniverse[i][0:len(prefix)]
		return item >= string(prefix)
	})
	if foundi >= len(allThePublicKeysInUniverse) {
		return "" // not found
	}
	item := allThePublicKeysInUniverse[foundi]
	if strings.HasPrefix(item, string(prefix)) {
		return item
	}
	return ""
}

func decodeKey(key string, destination []byte) (int, error) {
	n, err := base64.RawURLEncoding.Decode(destination, []byte(key))
	return n, err
}

// from the short name of first 4 b64 from pub key to the 128 byte private key in hex
// ed25519 token signing private keys.
var knownPrivateKeys = make(map[string]string)
var kpkSync sync.Mutex
var knownPrivateKeyPrefixes []string

// GetPrivateKey will return the Nth private key from the list of known private keys.
func GetPrivateKeyWhole(n int) string {
	LoadPrivateKeys("~/atw/privateKeys4.txt")
	kpkSync.Lock()
	defer kpkSync.Unlock()
	prefix := knownPrivateKeyPrefixes[n][0:4]
	return knownPrivateKeys[prefix]
}

func GetPrivateKeyPrefix(n int) string {
	LoadPrivateKeys("~/atw/privateKeys4.txt")
	kpkSync.Lock()
	defer kpkSync.Unlock()
	prefix := knownPrivateKeyPrefixes[n][0:4]
	return prefix
}

func GetPrivateKeyMatching(prefix string) string {
	LoadPrivateKeys("~/atw/privateKeys4.txt")
	kpkSync.Lock()
	defer kpkSync.Unlock()
	return knownPrivateKeys[prefix]
}

// LoadPrivateKeys is
func LoadPrivateKeys(fname string) error {

	// what if fname is different?
	if len(knownPrivateKeys) > 0 {
		return nil
	}
	kpkSync.Lock()
	defer kpkSync.Unlock()
	home, _ := os.UserHomeDir()
	fname = strings.Replace(fname, "~", home, 1)
	data, err := os.ReadFile(fname)
	if err != nil {
		fmt.Println("pk read file err", fname, err)
		return err
	}
	data = []byte(strings.Trim(string(data), "\n"))
	datparts := strings.Split(string(data), "\n")
	for _, part := range datparts {

		part = strings.ReplaceAll(part, "/", "_") // std to url encoding
		part = strings.ReplaceAll(part, "+", "-") // std to url encoding

		bytes, err := base64.RawURLEncoding.DecodeString(part)
		if err != nil {
			fmt.Println("fail to decode part")
			continue
		}
		if len(bytes) != 64 {
			fmt.Println("fail 64 bytes expected", len(bytes))
			continue
		}

		privateKey := ed25519.PrivateKey(bytes)
		publicKey := privateKey.Public()
		epublic := bytes[32:] // publicKey.([]byte) or get bytes or something
		public64 := base64.RawURLEncoding.EncodeToString([]byte(epublic))
		fmt.Println("loaded public key ", public64)
		//fmt.Println(public64)
		first4 := public64[0:4]
		knownPrivateKeys[first4] = string(privateKey)
		knownPrivateKeyPrefixes = append(knownPrivateKeyPrefixes, first4)
		_ = publicKey
		_ = epublic
	}
	return nil
}

// ZeroReader is too public
type ZeroReader struct{}

func (ZeroReader) Read(buf []byte) (int, error) {
	for i := range buf {
		buf[i] = 0
	}
	return len(buf), nil
}

// CountReader is too public
type CountReader struct {
	count int
}

func (cr *CountReader) Read(buf []byte) (int, error) {
	for i := range buf {
		buf[i] = byte(cr.count)
		cr.count++
	}
	return len(buf), nil
}

// // xxunusedxxParseOpenSSHPublicKey an ed25519 result is 32 bytes
// // some routines to parse open ssl that we'll probably never use
// func xxunusedxxParseOpenSSHPublicKey(in []byte) []byte {

// 	out, rest, ok := parseString(in)
// 	_ = rest
// 	_ = ok // atw fixme

// 	out, rest, ok = parseString(rest)

// 	if len(out) != 32 {
// 		// atw fixme
// 	}

// 	return out
// }

// // xxunusedxxParseOpenSSHPrivateKey parses openssh private key in the file here.
// func xxunusedxxParseOpenSSHPrivateKey(in []byte) []byte {

// 	// see ssh.keys.go
// 	const magic = "openssh-key-v1\x00"
// 	rest := in[len(magic):]
// 	cipher, rest, ok := parseString(rest)     // is "none"
// 	kdfname, rest, ok := parseString(rest)    // is "none"
// 	kdfoptions, rest, ok := parseString(rest) // is ""
// 	keyCount := binary.BigEndian.Uint32(rest) // is 1
// 	rest = rest[4:]

// 	pubkey, rest, ok := parseString(rest)

// 	remaining := binary.BigEndian.Uint32(rest) // is 160
// 	rest = rest[4:]

// 	crap := string(rest[0:8])
// 	rest = rest[8:] // pass some crap

// 	keytype, rest, ok := parseString(rest) // is "ssh-ed25519"

// 	pubkeyAgain, rest, ok := parseString(rest)

// 	privKey, rest, ok := parseString(rest)

// 	_ = ok
// 	_ = cipher
// 	_ = kdfname
// 	_ = kdfoptions
// 	_ = keyCount
// 	_ = pubkey
// 	_ = remaining
// 	_ = keytype
// 	_ = pubkeyAgain
// 	_ = crap

// 	if len(privKey) != 64 {
// 		// atw fixme
// 	}

// 	return privKey
// }

func parseString(in []byte) (out, rest []byte, ok bool) {
	if len(in) < 4 {
		return
	}
	length := binary.BigEndian.Uint32(in)
	in = in[4:]
	if uint32(len(in)) < length {
		return
	}
	out = in[:length]
	rest = in[length:]
	ok = true
	return
}

// GetSampleBigToken is used for testing. 256k connections is GiantX32
func GetSampleBigToken(startTime uint32, serviceUrl string) *KnotFreeTokenPayload {
	p := &KnotFreeTokenPayload{}
	p.Issuer = GetPrivateKeyPrefix(0)             // "_9sh"                             // first 4 from public
	p.ExpirationTime = startTime + 60*60*24*(365) // year
	p.JWTID = GetRandomB36String()

	// 256k connections is GiantX32
	p.KnotFreeContactStats = GetTokenStatsAndPrice(GiantX32).Stats
	// p.Input = 1e6
	// p.Output = 1e6
	// p.Subscriptions = 200000
	// p.Connections = 200000
	p.URL = serviceUrl
	return p
}

// GetSampleTokenFromStats is used for testing.
func GetSampleTokenFromStats(startTime uint32, serviceUrl string, stats KnotFreeContactStats) *KnotFreeTokenPayload {
	p := &KnotFreeTokenPayload{}
	p.Issuer = GetPrivateKeyPrefix(0)             //"_9sh"                             // first 4 from public
	p.ExpirationTime = startTime + 60*60*24*(365) // year
	p.JWTID = GetRandomB36String()

	p.KnotFreeContactStats = stats

	p.URL = serviceUrl
	return p
}

var giantToken = ""
var mediumToken = ""

// GetImpromptuGiantToken is GiantX32 256k connections is GiantX32
func GetImpromptuGiantToken() string {
	if len(giantToken) != 0 {
		return giantToken
	}

	LoadPrivateKeys("~/atw/privateKeys4.txt")

	payload := GetSampleBigToken(uint32(time.Now().Unix()), "knotfree.net/mqtt")
	signingKey := GetPrivateKeyWhole(0)
	bbb, err := MakeToken(payload, []byte(signingKey))
	if err != nil {
		fmt.Println("GetImpromptuGiantToken", err)
	}
	giantToken = string(bbb)
	return giantToken
}

func GetImpromptuGiantTokenLocal() string {

	if len(giantToken) != 0 {
		return giantToken
	}

	LoadPrivateKeys("~/atw/privateKeys4.txt")

	payload := GetSampleBigToken(uint32(time.Now().Unix()), "knotfree.dog:8085/mqtt") // is localhost in my /etc/hosts
	signingKey := GetPrivateKeyWhole(0)
	bbb, err := MakeToken(payload, []byte(signingKey))
	if err != nil {
		fmt.Println("GetImpromptuGiantToken", err)
	}
	giantToken = string(bbb)
	return giantToken
}

func Get32xTokenLocal() []byte {

	if len(aTest32xToken) != 0 {
		return aTest32xToken
	}

	LoadPrivateKeys("~/atw/privateKeys4.txt")

	payload := GetSampleTokenFromStats(uint32(time.Now().Unix()), "knotfree.dog:8085/mqtt", GetTokenStatsAndPrice(Medium).Stats) // is localhost in my /etc/hosts
	signingKey := GetPrivateKeyWhole(0)
	bbb, err := MakeToken(payload, []byte(signingKey))
	if err != nil {
		fmt.Println("Get32xTokenLocal", err)
	}
	aTest32xToken = bbb
	return aTest32xToken
}

// GetRandomB64String returns 18 bytes or 18 * 8 = 144 bits of randomness aka 24 bytes
func GetRandomB64String() string {
	var tmp [18]byte
	rand.Read(tmp[:])
	return base64.RawURLEncoding.EncodeToString(tmp[:])
}

const b36 = "0123456789abcdefghijklmnopqrstuvwxyz"

// GetRandomB32String returns 24 bytes but only 24 * 5.1699 = 124 bits of random.
// but it's url compatible
func GetRandomB36String() string {
	var tmp [24]byte
	rand.Read(tmp[:])
	for i := 0; i < len(tmp); i++ {
		tmp[i] = b36[int(mathrand.Float32()*float32(len(b36)))]
	}
	return string(tmp[:])
}

// LoadPublicKeys adds the public keys below
func LoadPublicKeys() {

	if len(allThePublicKeysInUniverse) > 0 {
		return // just do this once
	}

	allThePublicKeysInUniverse = append(allThePublicKeysInUniverse, "dummy")

	tmp := strings.Trim(GetPublicKeys(), " \n")
	parts := strings.Split(tmp, "\n")

	for _, s := range parts {

		if len(s) < 4 {
			continue
		}

		s = strings.ReplaceAll(s, "/", "_") // std to url encoding
		s = strings.ReplaceAll(s, "+", "-") // std to url encoding

		s = strings.Trim(s, " \n")
		if len(s) != 43 {
			fmt.Println("fatal", len(s))
		}
		front := s[0:4]
		bytes, _ := base64.RawURLEncoding.DecodeString(s)
		SavePublicKey(front, string(bytes))
	}
	allThePublicKeysInUniverse = allThePublicKeysInUniverse[1:]
}

var StrangerSecretPhrase string = "dummy-dummy-dummy-dummy-dummy-dummy-dummy-dummy-dummy-dummy-dummy"

//var StrangerPrivateKey string = "cc0Obtu-3pBttENYZ2TqIMmbHH0Iv10U8SA8HXzEi0CNxD1gkawsQ-F4P4-eLl1TF_RzpZp2y64K42MigbKh0g"

// name alice_vociferous_mcgrath
var AliceSecretPhrase string = "join_red_this_string_plain_does_quart_simple_buy_line_fun_look_original_deal"

//var AlicePrivateKey string = "adnwz7Psriz6gjWog2zpkeqCrblavaDgWwQohp-av973sio8TWFb7mUinD3v_AbflX48eiqBbxq25PDUQvnDOA"

// building_bob_bottomline_boldness
var BobSecretPhrase string = "tail_wait_king_particular_track_third_arrive_agree_plural_charge_rise_grew_continent_fact"

//var BobPrivateKey string = "Lxy9zqiUPVYQ1HXJ1JFIRyTxFCeg-duJhN5ja4pQGlK8e--VbYj674lgchUPLBg1wgq4MkmezVIm3skQqDCyDw"

var CharlieSecretPhrase string = "sense_trouble_lost_final_crowd_child_fear_buy_card_apple_such_it_as_note"

//var CharliePrivateKey string = "J0KCPMT8QCW6l8Et3VX4Cn7BZ-VM3DbRQ1CC_nGxiD7e0YGd9XrRBv5o0C2PqB79slHHmD7Hn98Ebs3RX2ejSw"

var e_words []string

func MakeRandomPhrase(amount int) string {

	var tmp [8]byte
	rand.Read(tmp[:])
	rand64 := int64(0)
	for i := 0; i < len(tmp); i++ {
		rand64 = rand64<<8 + int64(tmp[i])
	}
	mathrand.Seed(rand64)

	if len(e_words) == 0 {
		str := English_words
		e_words = strings.Split(str, "\n")
	}
	result := ""
	for i := 0; i < amount; i++ {
		if i > 0 {
			result += "-"
		}
		max := len(e_words)
		index := mathrand.Intn(max)
		word := e_words[index]
		result += word
	}
	return result
}

func GetBoxKeyPairFromPassphrase(pass string) ([32]byte, [32]byte) {

	publicKey := new([32]byte)
	privateKey := new([32]byte)

	hash := sha256.Sum256([]byte(pass))
	copy(privateKey[:], hash[:])

	curve25519.ScalarBaseMult(publicKey, privateKey)

	return *publicKey, *privateKey
}
