package tokens

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/awootton/knotfreeiot/badjson"
	"github.com/gbrlsnchs/jwt/v3"
)

// KnotFreeTokenPayload is our JWT 'claims'.
type KnotFreeTokenPayload struct {
	//
	ExpirationTime uint32 `json:"exp,omitempty"` // unix seconds
	Issuer         string `json:"iss"`           // first 4 bytes (or more) of base64 public key of issuer
	JWTID          string `json:"jti,omitempty"` // a unique serial number for this Issuer

	KnotFreeContactStats // limits on what we're allowed to do.

	URL string `json:"url"` // address of the service eg. "knotfree.net"
}

// KnotFreeContactStats is the numeric part of the token claims
// it is floats to compress numbers and allow fractions in json
// these don't count above 2^24 or else we need more bits.
type KnotFreeContactStats struct {
	//
	Input         float32 `json:"in"`  // bytes per sec
	Output        float32 `json:"out"` // bytes per sec
	Subscriptions float32 `json:"su"`  // seconds per sec
	Connections   float32 `json:"co"`  // seconds per sec
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
	// TODO: compare all the fields with limits.
	// FIXME:
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

	// part 1
	{
		firstPart := "eyJhbGciOiJFZDI1NTE5IiwidHlwIjoiSldUIn0."
		index := strings.Index(token, firstPart)
		tokenStartIndex = index
		tokenEndIndex = index + len(firstPart)
		if index < 0 {
			s := "expected eyJhbG..."
			return token, issuer, errors.New(s)
		}
	}
	// part 2
	{
		t := token[tokenEndIndex:]
		index := strings.Index(t, ".")
		if index < 0 {
			s := "expected ."
			return token, issuer, errors.New(s)
		}
		part2 := token[tokenEndIndex : tokenEndIndex+index]
		claimsPlain, err := base64.RawStdEncoding.DecodeString(part2)
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
		fmt.Println("fixme 43456")
		return
	}
	allThePublicKeysInUniverse = append(allThePublicKeysInUniverse, publicKey)
	sort.Strings(allThePublicKeysInUniverse)
}

// FindPublicKey is
func FindPublicKey(thekey string) string {

	if thekey == "1iVt" { // TODO: better black list
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
	n, err := base64.RawStdEncoding.Decode(destination, []byte(key))
	return n, err
}

// from the short name of first 4 b64 from pub key to the 128 byte private key in hex
// ed25519 token signing private keys.
var knownPrivateKeys = make(map[string]string)
var kpkSync sync.Mutex

// GetPrivateKey is
func GetPrivateKey(first4 string) string {
	kpkSync.Lock()
	defer kpkSync.Unlock()
	return knownPrivateKeys[first4]
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
	data, err := ioutil.ReadFile(fname)
	if err != nil {
		fmt.Println("pk read file err", fname, err)
		return err
	}
	datparts := strings.Split(string(data), "\n")
	for _, part := range datparts {

		bytes, err := base64.RawStdEncoding.DecodeString(part)
		if err != nil {
			fmt.Println("fail 5")
			continue
		}
		if len(bytes) != 64 {
			fmt.Println("fail 64", len(bytes))
			continue
		}

		privateKey := ed25519.PrivateKey(bytes)
		publicKey := privateKey.Public()
		epublic := bytes[32:] // publicKey.([]byte) or get bytes or something
		public64 := base64.RawStdEncoding.EncodeToString([]byte(epublic))
		//fmt.Println(public64)
		first4 := public64[0:4]
		knownPrivateKeys[first4] = string(privateKey)
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

// ParseOpenSSHPublicKey an ed25519 result is 32 bytes
// some routines to parse open ssl that we'll probably never use
func ParseOpenSSHPublicKey(in []byte) []byte {

	out, rest, ok := parseString(in)
	_ = rest
	_ = ok // atw fixme

	out, rest, ok = parseString(rest)

	if len(out) != 32 {
		// atw fixme
	}

	return out
}

// ParseOpenSSHPrivateKey parses openssh private key in the file here.
func ParseOpenSSHPrivateKey(in []byte) []byte {

	// see ssh.keys.go
	const magic = "openssh-key-v1\x00"
	rest := in[len(magic):]
	cipher, rest, ok := parseString(rest)     // is "none"
	kdfname, rest, ok := parseString(rest)    // is "none"
	kdfoptions, rest, ok := parseString(rest) // is ""
	keyCount := binary.BigEndian.Uint32(rest) // is 1
	rest = rest[4:]

	pubkey, rest, ok := parseString(rest)

	remaining := binary.BigEndian.Uint32(rest) // is 160
	rest = rest[4:]

	crap := string(rest[0:8])
	rest = rest[8:] // pass some crap

	keytype, rest, ok := parseString(rest) // is "ssh-ed25519"

	pubkeyAgain, rest, ok := parseString(rest)

	privKey, rest, ok := parseString(rest)

	_ = ok
	_ = cipher
	_ = kdfname
	_ = kdfoptions
	_ = keyCount
	_ = pubkey
	_ = remaining
	_ = keytype
	_ = pubkeyAgain
	_ = crap

	if len(privKey) != 64 {
		// atw fixme
	}

	return privKey
}

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

// GetSampleBigToken is used for testing.
func GetSampleBigToken(startTime uint32) *KnotFreeTokenPayload {
	p := &KnotFreeTokenPayload{}
	p.Issuer = "/9sh" // first 4 from public
	p.ExpirationTime = startTime + 60*60*24*(365+1)
	p.JWTID = getRandomB64String()
	p.Input = 1e6
	p.Output = 1e6
	p.Subscriptions = 200000
	p.Connections = 200000
	p.URL = "knotfree.net"
	return p
}

var giantToken string

// GetImpromptuGiantToken is
func GetImpromptuGiantToken() string {
	if len(giantToken) != 0 {
		return giantToken
	}

	LoadPrivateKeys("~/atw/privateKeys4.txt")

	payload := GetSampleBigToken(uint32(time.Now().Unix()))
	signingKey := GetPrivateKey("/9sh")
	bbb, err := MakeToken(payload, []byte(signingKey))
	if err != nil {
		fmt.Println("GetImpromptuGiantToken", err)
	}
	giantToken = string(bbb)
	return giantToken
}

func getRandomB64String() string {
	var tmp [18]byte
	rand.Read(tmp[:])
	return base64.RawStdEncoding.EncodeToString(tmp[:])
}

// LoadPublicKeys adds the public keys below
func LoadPublicKeys() {

	if len(allThePublicKeysInUniverse) > 0 {
		return // just do this once
	}

	tmp := strings.Trim(publicKeys, " \n")
	parts := strings.Split(tmp, "\n")

	for _, s := range parts {
		s = strings.Trim(s, " \n")
		if len(s) != 43 {
			fmt.Println("fatal", len(s))
		}
		front := s[0:4]
		bytes, _ := base64.RawStdEncoding.DecodeString(s)
		SavePublicKey(front, string(bytes))
	}
}
