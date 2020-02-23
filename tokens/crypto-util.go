package tokens

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/gbrlsnchs/jwt/v3"
)

// KnotFreePayload is our MQTT password.
type KnotFreePayload struct {
	ExpirationTime uint32 `json:"exp"` // unix seconds
	Issuer         string `json:"iss"` // first 4 bytes (or more) of base64 public key of issuer
	JWTID          string `json:"jti"` // a unique serial number for this Issuer

	Input         float32 `json:"in"`  // bytes per hour
	Output        float32 `json:"out"` // bytes per hour
	Subscriptions float32 `json:"su"`  // hours per hour
	Connections   float32 `json:"co"`  // hours per hour

	URL string `json:"url"` // address of the service eg. "knotfree.net"
}

// MakeTicket is
func MakeTicket(data *KnotFreePayload, privateKey []byte) ([]byte, error) {

	algo := jwt.NewEd25519(jwt.Ed25519PrivateKey(privateKey))
	token, err := jwt.Sign(data, algo)
	if err != nil {
		return []byte(""), err
	}
	return token, nil
}

// VerifyTicket is
func VerifyTicket(ticket []byte, publicKey []byte) (*KnotFreePayload, bool) {

	payload := KnotFreePayload{}

	algo := jwt.NewEd25519(jwt.Ed25519PublicKey(publicKey))
	hd, err := jwt.Verify([]byte(ticket), algo, &payload)
	if err != nil {
		return &KnotFreePayload{}, false
	}
	_ = hd
	// TODO: compare all the fields with limits.
	// FIXME:
	return &payload, true
}

// GetKnotFreePayload returns the payload THAT IS NOT VERIFIED YET.
func GetKnotFreePayload(token string) (*KnotFreePayload, error, string) {

	payload := new(KnotFreePayload)
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return payload, errors.New("expected 3 parts"), ""
	}
	middle, err := base64.RawStdEncoding.DecodeString(parts[1])
	if err != nil {
		return payload, err, ""
	}
	err = json.Unmarshal(middle, &payload)
	if err != nil {
		return payload, err, ""
	}
	return payload, nil, parts[2]
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

// SampleSmallToken is a small token signed by "1iVt" (below)
var SampleSmallToken = "eyJhbGciOiJFZDI1NTE5IiwidHlwIjoiSldUIn0.eyJleHAiOjE2MDk0NjI4MDAsImlzcyI6IjFpVnQiLCJqdGkiOiIxMjM0NTYiLCJpbiI6NzAwMDAsIm91dCI6NzAwMDAsInN1IjoyLCJjbyI6MiwidXJsIjoia25vdGZyZWUubmV0In0.T7SrbbXq7V7otfX0eo9eFabWguxwuPsG4Zn9XArGwMc2Q4ifMBm9aSOgvBIBn1Q0Or7pvIsA8u_UL9FnOW-aDg"

// ZeroReader is too public
type ZeroReader struct{}

func (ZeroReader) Read(buf []byte) (int, error) {
	for i := range buf {
		buf[i] = 0
	}
	return len(buf), nil
}

// or  crypto/rand:  rand.Read(b []byte) (n int, err error)

// some routines to parse open ssl that we'll probably never use

// ParseOpenSSHPublicKey an ed25519 result is 32 bytes
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

var samplePublic = "1iVt3d1E9TaxD/N0rC8c70pD5GryNlu49JC+iWD6UJc"
var samplePrivate = "u36xbHik/s/5uG6RCPT6MfAYHKJzk/nCZPHzZYZi2czWJW3d3UT1NrEP83SsLxzvSkPkavI2W7j0kL6JYPpQlw"

// GetSamplePublic is
func GetSamplePublic() []byte {
	bytes, _ := base64.RawStdEncoding.DecodeString(samplePublic)
	return bytes
}

// GetSamplePrivate is
func GetSamplePrivate() []byte {
	bytes, _ := base64.RawStdEncoding.DecodeString(samplePrivate)
	return bytes
}

// GetSampleTokenPayload is used for testing.
func GetSampleTokenPayload(startTime uint32) *KnotFreePayload {
	p := &KnotFreePayload{}
	p.Issuer = "1iVt" // first 4 from public
	p.ExpirationTime = startTime + 60*60*24*(365+1)
	p.JWTID = "123456"
	p.Input = 7e4
	p.Output = 7e4
	p.Subscriptions = 2
	p.Connections = 2
	p.URL = "knotfree.net"
	return p
}

// GetSampleBigToken is used for testing.
func GetSampleBigToken(startTime uint32) *KnotFreePayload {
	p := &KnotFreePayload{}
	p.Issuer = "1iVt" // first 4 from public
	p.ExpirationTime = startTime + 60*60*24*(365+1)
	p.JWTID = "123457"
	p.Input = 1e6
	p.Output = 1e6
	p.Subscriptions = 200
	p.Connections = 200
	return p
}
