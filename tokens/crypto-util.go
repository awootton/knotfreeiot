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

// KnotFreeTokenPayload is our MQTT password.
type KnotFreeTokenPayload struct {
	ExpirationTime uint32 `json:"exp"` // unix seconds
	Issuer         string `json:"iss"` // first 4 bytes (or more) of base64 public key of issuer
	JWTID          string `json:"jti"` // a unique serial number for this Issuer

	Input         float32 `json:"in"`  // bytes per sec
	Output        float32 `json:"out"` // bytes per sec
	Subscriptions float32 `json:"su"`  // hours per sec
	Connections   float32 `json:"co"`  // hours per sec

	URL string `json:"url"` // address of the service eg. "knotfree.net"
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

// GetKnotFreePayload returns the payload THAT IS NOT VERIFIED YET.
func GetKnotFreePayload(token string) (*KnotFreeTokenPayload, string, error) {

	payload := new(KnotFreeTokenPayload)
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return payload, "", errors.New("expected 3 parts")
	}
	middle, err := base64.RawStdEncoding.DecodeString(parts[1])
	if err != nil {
		return payload, "", err
	}
	err = json.Unmarshal(middle, &payload)
	if err != nil {
		return payload, "", err
	}
	return payload, parts[2], nil
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

// SampleSmallToken is a small token signed by "/9sh" (below)
var SampleSmallToken = "eyJhbGciOiJFZDI1NTE5IiwidHlwIjoiSldUIn0.eyJleHAiOjE2MDk0NjI4MDAsImlzcyI6Ii85c2giLCJqdGkiOiIxMjM0NTYiLCJpbiI6MjAsIm91dCI6MjAsInN1IjoyLCJjbyI6MiwidXJsIjoia25vdGZyZWUubmV0In0.YmKO8U_jKYyZsJo4m4lj0wjP8NJhciY4y3QXt_xlxvnHYznfWI455JJnnPh4HZluGaUcvrNdKAENGh4CfG4tBg"

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
/9sh+kvk3Nd/oN7nq56ydRaFON0YxQ+qCoBL0H91fV4
8ZNPzzn2EEnlFCAH6Z//KNHoIyhnIWDGRcy0Ub6F/mc
yRst5ig1Zf1iYVvI0q0LltjU8gmT+9ZZBKWijosq2Vg
JvaLqA2oYU9mZHcYYtCWJ7occcW5BiNpbdR2gSVHCFY
JIbDPOv+0H2zT6bXlO8oMGWWh9NJf+Mz4d6UXETiPZo
aNhfKWPWWrCkP8R/BCWUmgwv2gZg2wz9e/FmXdKqNG0
RHLSR6DdlpwCeYOE7DF/QaUGE3AwMZU4F0/uuM1HYCY
B30LVkD9TY96cD6S54xrnSoa6j/W14RJ0NH55YPiaMw
cj2gEtBk0qXrxhjKbwUYlD1naOMMHhX0L3s7qGHMvmw
wfrQr0IqTuvXwTlNdg4yO0H5d2nmeEV93kwkplVV0Gc
sJNAsh0yH3sY8Qu56zo9J64kNSju+o662FT+OEaW3sc
p/ia+nTuOaEbKkp2S8uTyccacmdEKPaxj7AOzIyYPbU
VdGjvGBES2cBXsk8XvJVj55woUxTDegvR+NB1jfocbU
IN9yT9wMGTOoLQDgdHK7ue8IOzLHkrw5/0DM06jYYlI
dmTDblSn2A/gnF1dB6RuFDjMk29G8DziKBH0zOUjqUg
fr8KVrMqF669rKazI6Vs3OO3dYyGjW+gMgXx/XLiEX8
V5iE4tUGSeamu/r24XOWsrvzvdt0A8R+O2XArT1lvmQ
ql/nLaNeSDtl6i9fKofC2WT2H9VqHLj0VCLgWS8oEcM
cvVrcTKky67XukswYgYdttODLTuh5iwlpCBAKaysFGw
leePkNZx3ns8LOS5jxjxH0ybjn5E6r5gaEO4fwRXO8g
3ZKqO3ppTjjfaGFcgwYAcJ9OvXVF0hyeIu8KQgMVOQw
SxC+EHhmiVYCAtpvp3HWknAdkwzVhKaRnmj8Gnsic5Y
ebBVe8AMUIvz6raYozdfAWeRmcjF2a8lvY1dTnjDOOc
O+x0aSZ42c/AUH4hnb0GNRx2I70R1ncuBAeOSrLaG0k
rKxTWJhMAvaDtLEmxxB6kYSvpJR7ou80dMCEOOxzrLA
6sCrFd/c4Leh6F9WxpSCsuKeANpNN57OJxPcDK5KC68
3aTB768a1HYrBb2KA1rXv/A6AgBqZW1F7n3JTK8vpl8
bkBYvnQqxzCUBNpz8aPGBd8rM4gzdGO+JnNueicecr4
zGJSXO5/KdqqYMwBtHguHpT14jQdE+OOA6PQZjVphuQ
gW6CF4WH6dyg3Yx1LqLGpiH707NnQUP8nM9PY15SBKA
rkDnOkSj6XPNNyH/Vlkaaewwi3q8/ePcvXUOiBIu52o
ByKGuFQteDJLFizQfW1oPGbyh9rL1Yj7SNE0f/q5Xys
pwAkPWMNZwjuiTrPg4YR58KJFIjqn204BjVzdaCChtI
mp+9zBU/kSIAMHiZiZBxXe+DtIuddwsWWao/AtU7fmQ
tA8lPUJtgDP80ga+bj7XFwn2p6BOSZghk21v5X2jq5s
JAKlfGDiioDYYZEsq6NtEhZdIkCl83oHQRTe+SiA/bI
GJIebuse2Q/9T6wRFb7rlPd9uOcom0Wx28C/OCB4wHc
klitu+aunEGRjMaj2nYbBBS9hoohbDmIToQg+9Oc1pw
kNQSM4gz+1eXDoHnCOIK3oWvhczEgHuP6fD0ecqjGNo
f//Mser4Py/e/hIvxyDL9q2vjEdz6+ThZYrmXoVBxKc
vgpHdHc35hIj/DW+vwIiNbyUYwWsihApFo7Vfjd3z94
J8u8BnH6o6QOMJ16UwgIhL5Dn/ARB31xqnzvYMoHH6o
N7ok5KZ3YbgcxHkh8ZdV38yE+2Azq7BzyDDrC+JnMAY
FFdKDiX45E2RfauLWXVd7xBmFHO9Tu2zJSk6FTWHjbc
HgfPkJqVvifOEZQsJIdAJGGQVlpRO4JAhtcsp6Fz4lI
28Mwm1olWZ0D42IYd0hUlyGeHWN9jf4muiSQWen+WS4
arS0VuqGXNWssBgGc88n1ZfKA1KEcFYgn+Ox//LH5/w
8X80fLAo3Cfct/KqYRutuDLv4uCPZ2i3K7ayO7hYUlc
TJ6ZGaAfHZIU3T3EQ0L/jvB90L/R9yLjsECNFcFAXPY
gU51mGgvwB/OkQPY9YB3TSi+eNrBQh4vGLD6fTD4qrE
n25r4SFrtVsrfMGUw8kWUF4vTCkezgJ8raB4UpSKiTQ
IElPHV4ShGf0kN9pdKgvJrTT9JspWF2vMTtWBqTqUAY
sgIKxzYEWre7ZNYT4cfYldcGO3XUmXnIksJJh6+miP4
WVrL5zNpeO0BFlZr4hyBOdK7tLDyC37JrGbRvvEHhoc
JNMyq/aR4kQlHp0+x8D+E2caIBypaLUfBBzyXxYqsio
aoaJoZbc0AzXPCZTcfwVUnr3f5Owrojhh4w/wG9JH1M
UYsfYekd71ElufJcfJ9PMOyYkPoDgSXvlo3V6LKB5zU
xvWcpdZGW7GdrmZAIJsbGydcYXx495qSacoTSN1Xdsg
g+ezyaJgv/ZwBpEr80pLxGweXF1Hn6KIVJCg779+/FY
nm3TYMVGlIN+tXYoiAvOILjKUsmJ3OdbhGkh9puxguA
8cPnPSfE9wy7erZGriwde/R2u46mvDP0IGtfFDXaiJw
Ditv5v1hDgI5L0rD2dgJN6Iz+hzVqAiB08t7vSFnYxw
vdreVQjOIrv2o+wW/EJi0g+bQ8S71NHFB45ndKE1Des
7hwDiSi9ZOOn4IXVEIbMdTqpRE2ayScY6uogj5aBad0
`
