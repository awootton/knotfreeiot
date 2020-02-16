package tickets

import (
	"encoding/base64"
	"encoding/binary"

	"github.com/gbrlsnchs/jwt/v3"
)

// KnotFreePayload is
type KnotFreePayload struct {
	//jwt.Payload
	ExpirationTime uint32 `json:"exp"`
	Issuer         string `json:"iss"`
	JWTID          string `json:"jti"`

	Input         float32 `json:"in"`  // bytes per hour
	Output        float32 `json:"out"` // bytes per hour
	Subscriptions float32 `json:"su"`  // per hour
	Connections   float32 `json:"co"`  // per hour

	URL string `json:"url"` // address of the service eg. "knotfreeiot.io"
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
