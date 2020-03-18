package tokens

import (
	"crypto/ed25519"
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
	Subscriptions float32 `json:"su"`  // hours per sec
	Connections   float32 `json:"co"`  // hours per sec
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

// GetPrivateKey is
func GetPrivateKey(first4 string) string {
	return knownPrivateKeys[first4]
}

// LoadPrivateKeys is
func LoadPrivateKeys(fname string) error {
	home, _ := os.UserHomeDir()
	fname = strings.Replace(fname, "~", home, 1)
	data, err := ioutil.ReadFile(fname)
	if err != nil {
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
			fmt.Println("fail 64")
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

// SampleSmallToken is a small token signed by "/9sh" (below)
// p.Input = 20
// p.Output = 20
// p.Subscriptions = 2
// p.Connections = 2
var SampleSmallToken = `["My token expires: 2020-12-30",{"iss":"/9sh","in":32,"out":32,"su":4,"co":2,"url":"knotfree.net"},"eyJhbGciOiJFZDI1NTE5IiwidHlwIjoiSldUIn0.eyJleHAiOjE2MDkzNzI4MDAsImlzcyI6Ii85c2giLCJqdGkiOiJyMWxkWnRsU3ljSVJlcFpRbWtPYVFIdGsiLCJpbiI6MzIsIm91dCI6MzIsInN1Ijo0LCJjbyI6MiwidXJsIjoia25vdGZyZWUubmV0In0.xkFa05XXUXphdBXwVTaZKLQlpsXzZtuVIET0dStobB1JhTcqEikw7snxUbR4YxLg7DlT_LpKeS1G2arYm3pgDw"]`

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

var publicKeys string = `
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
`
