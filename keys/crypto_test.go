package crypto

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"strings"
	"testing"
	//"golang.org/x/crypto/ed25519"
)

type AtwEd int

func ExampleZeroReader() {

	var zero ZeroReader
	public, private, _ := ed25519.GenerateKey(zero)

	fmt.Println(base64.StdEncoding.WithPadding(base64.NoPadding).EncodeToString(public))
	fmt.Println(base64.StdEncoding.WithPadding(base64.NoPadding).EncodeToString(private))

	message := []byte("test message")
	sig := ed25519.Sign(private, message)
	if !ed25519.Verify(public, message, sig) {
		fmt.Println("valid signature rejected")
	} else {
		fmt.Println("good")
	}

	// Expected: O2onvM62pC1io6jQKm8Nc2UyFXcd4kOmOsBIoYtZ2ik
	// AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA7aie8zrakLWKjqNAqbw1zZTIVdx3iQ6Y6wEihi1naKQ
	// good

}

func Test1(t *testing.T) {
	ExampleZeroReader()
}

func Test2(t *testing.T) {
	fmt.Println("hello2")
	ExampleZeroReader()

	tmp, err := ioutil.ReadFile("ccced25519_2.pub")
	str := string(tmp)
	str = strings.Split(str, " ")[1]
	str = strings.ReplaceAll(str, " ", "")
	str = strings.ReplaceAll(str, "\n", "")

	public, err := base64.RawStdEncoding.DecodeString(str)

	public = ParsePublic(public)

	if err != nil || len(public) < 32 {
		t.Error()
		return
	}
	fmt.Println(base64.RawStdEncoding.EncodeToString(public))

	if len(public) != ed25519.PublicKeySize { // 32
		t.Error()
		return
	}

	tmp, err = ioutil.ReadFile("ccced25519_2")
	str = string(tmp)
	str = strings.Split(str, "-----")[2]
	str = strings.ReplaceAll(str, " ", "")
	str = strings.ReplaceAll(str, "\n", "")
	private, err := base64.RawStdEncoding.DecodeString(str)

	fmt.Println(base64.RawStdEncoding.EncodeToString(private))

	private = ParsePrivate(private)

	if err != nil || len(private) < 64 {
		t.Error()
		return
	}

	fmt.Println(base64.RawStdEncoding.EncodeToString(private))

	if len(private) != ed25519.PrivateKeySize { // 64
		t.Error()
		return
	}

	message := ([]byte("test message test message test message"))[0:32]
	sig := ed25519.Sign(private, message)
	if !ed25519.Verify(public, message, sig) {
		fmt.Println("valid signature rejected")
	} else {
		fmt.Println("good")
	}

	// the reverse ??
	// see https://blog.filippo.io/using-ed25519-keys-for-encryption/
	// message2 := "Top of the morning to you sir. a"
	// ed25519.

	_ = err

}

// ParsePublic an ed25519 result is 32 bytes
func ParsePublic(in []byte) []byte {

	//pubKey, err := ssh.ParsePublicKey(in) //(pubKey PublicKey, rest []byte, err error)
	//_ = err
	//_ = pubKey
	//  			the pubKey hides the 32 bytes so we do it the hard way:

	out, rest, ok := parseString(in)
	_ = rest
	_ = ok // atw fixme

	// fmt.Println(string(out)) // is "ssh-ed25519"

	out, rest, ok = parseString(rest)

	if len(out) != 32 {
		// atw fixme
	}

	return out
}

func ParsePrivate(in []byte) []byte {

	//signer, err := ssh.ParsePrivateKey(in) //(pubKey PublicKey, rest []byte, err error)
	//_ = err
	// it's opaque even though the key is right there !!!

	// see https://peterlyons.com/problog/2017/12/openssh-ed25519-private-key-file-format/

	// starts with:
	//# ASCII magic "openssh-key-v1" plus null byte
	// 6f70656e7373682d6b65792d7631 00 (15)bytes
	// then it's parseString over and over

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
