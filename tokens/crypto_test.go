package tokens_test

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"strings"
	"testing"
	"time"

	"github.com/awootton/knotfreeiot/tokens"
	"github.com/gbrlsnchs/jwt/v3"
	"golang.org/x/crypto/nacl/box"
)

const starttime = uint32(1577840400) // Wednesday, January 1, 2020 1:00:00 AM

// TODO: add more keys to this test.
func TestFind(t *testing.T) {

	got := "ok"
	want := "ok"

	got = tokens.FindPublicKey("abc")
	want = "" // because it's empty.
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	// tokens.SavePublicKey("1iVt", string(GetSamplePublic()))
	// got = tokens.FindPublicKey("1iVt")
	// want = string(GetSamplePublic())
	// if got != want {
	// 	t.Errorf("got %v, want %v", got, want)
	// }

	dat, err := ioutil.ReadFile("./publicKeys.txt")
	if err != nil {
		fmt.Println("fail 1")
	}
	datparts := strings.Split(string(dat), "\n")
	if len(datparts) < 64 {
		t.Errorf("got %v, want %v", len(datparts), 64)
	}
	for i, part := range datparts {
		if i >= 64 {
			break
		}
		prefix := part[0:4]
		bytes, err := base64.RawStdEncoding.DecodeString(part)
		if err != nil {
			fmt.Println("fail 2")
		}
		tokens.SavePublicKey(prefix, string(bytes))
	}

	for i, part := range datparts {
		if i >= 64 {
			break
		}
		prefix := part[0:4]
		bytes, err := base64.RawStdEncoding.DecodeString(part)
		want = string(bytes)
		if err != nil {
			fmt.Println("fail 3")
		}
		got = tokens.FindPublicKey(prefix)
		if got != want {
			t.Errorf("got %v, want %v", got, want)
		}
	}

}

func TestFind2(t *testing.T) {

	got := "ok"
	want := "ok"

	got = tokens.FindPublicKey("abc")
	want = "" // because it's empty.
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestKnotVerify(t *testing.T) {
	got := "ok"
	want := "ok"

	token := []byte("eyJhbGciOiJFZDI1NTE5IiwidHlwIjoiSldUIn0.eyJleHAiOjE2MDk0NjI4MDAsImlzcyI6IjFpVnQiLCJqdGkiOiIxMjM0NTYiLCJpbiI6NzAwMDAsIm91dCI6NzAwMDAsInN1IjoyLCJjbyI6MiwidXJsIjoia25vdGZyZWUubmV0In0.T7SrbbXq7V7otfX0eo9eFabWguxwuPsG4Zn9XArGwMc2Q4ifMBm9aSOgvBIBn1Q0Or7pvIsA8u_UL9FnOW-aDg")

	p, ok := tokens.VerifyToken(token, GetSamplePublic())
	if !ok {
		t.Errorf("got %v, want %v", "false", "true")
	}
	bytes, err := json.Marshal(p)
	_ = err
	got = string(bytes)
	want = `{"exp":1609462800,"iss":"1iVt","jti":"123456","in":70000,"out":70000,"su":2,"co":2,"url":"knotfree.net"}`
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestKnotEncode(t *testing.T) {

	got := "ok"
	want := "ok"

	p := GetSampleTokenPayload(starttime)

	// or:
	// p := &tickets.KnotFreePayload{}
	// p.Issuer = "1iVt" // first 4 from public
	// p.ExpirationTime = starttime + 60*60*24*(365+1))
	// p.JWTID = "123456"
	// p.Input = 7e4
	// p.Output = 7e4
	// p.Subscriptions = 2
	// p.Connections = 2

	if os.Getenv("KUBE_EDITOR") == "atom --wait" {
		p.Issuer = "/9sh"
		bytes, err := tokens.MakeToken(p, []byte(getRemotePublic("/9sh")))
		if err != nil {
			got = err.Error()
		}
		got = string(bytes)
		//the old one `eyJhbGciOiJFZDI1NTE5IiwidHlwIjoiSldUIn0.eyJleHAiOjE2MDk0NjI4MDAsImlzcyI6IjFpVnQiLCJqdGkiOiIxMjM0NTYiLCJpbiI6NzAwMDAsIm91dCI6NzAwMDAsInN1IjoyLCJjbyI6MiwidXJsIjoia25vdGZyZWUubmV0In0.T7SrbbXq7V7otfX0eo9eFabWguxwuPsG4Zn9XArGwMc2Q4ifMBm9aSOgvBIBn1Q0Or7pvIsA8u_UL9FnOW-aDg`
		// this is the sample token used in the tests. It's a /9sh small token.
		want = `eyJhbGciOiJFZDI1NTE5IiwidHlwIjoiSldUIn0.eyJleHAiOjE2MDk0NjI4MDAsImlzcyI6Ii85c2giLCJqdGkiOiIxMjM0NTYiLCJpbiI6MjAsIm91dCI6MjAsInN1IjoyLCJjbyI6MiwidXJsIjoia25vdGZyZWUubmV0In0.YmKO8U_jKYyZsJo4m4lj0wjP8NJhciY4y3QXt_xlxvnHYznfWI455JJnnPh4HZluGaUcvrNdKAENGh4CfG4tBg`
		if got != want {
			t.Errorf("got %v, want %v", got, want)
		}
	}
}

func getRemotePublic(key string) string {

	usr, _ := user.Current()
	dir := usr.HomeDir

	dat, err := ioutil.ReadFile(dir + "/atw/privateKeys.txt")
	if err != nil {
		fmt.Println("fail 4")
	}
	datparts := strings.Split(string(dat), "\n")
	if len(datparts) < 64 {
		fmt.Printf("got %v, want %v", len(datparts), 64)
	}
	for _, part := range datparts {

		bytes, err := base64.RawStdEncoding.DecodeString(part)
		if err != nil {
			fmt.Println("fail 5")
		}

		privateKey := ed25519.PrivateKey(bytes)
		publicKey := privateKey.Public()
		epublic := bytes[32:] // publicKey.([]byte) or get bytes or something
		public64 := base64.RawStdEncoding.EncodeToString([]byte(epublic))
		//fmt.Println(public64)
		first4 := public64[0:4]
		if first4 == key {
			return string(bytes)
		}

		_ = publicKey
		_ = epublic
	}

	return ""
}

type CustomPayload struct {
	jwt.Payload
	//	Foo string `json:"foo,omitempty"`
	Bar int `json:"bar,omitempty"`
}

func TestVerify(t *testing.T) {

	got := "ok"
	want := "ok"

	ticket := "eyJhbGciOiJFZDI1NTE5IiwidHlwIjoiSldUIn0.eyJpc3MiOiJhdHciLCJzdWIiOiJrbm90ZnJlZSIsImF1ZCI6WyJodHRwczovL2dvbGFuZy5vcmciLCJodHRwczovL2p3dC5pbyJdLCJleHAiOjE2MTI5MzA4MDEsIm5iZiI6MTU4MTgyODYwMSwiaWF0IjoxNTgxODI2ODAxLCJqdGkiOiJmb29iYXIiLCJmb28iOiJmb28iLCJiYXIiOjEzMzd9.AvRapYOS1WHzds8zFscDdwWngj0t4OYYPLoyfEPnWNknwJbaHandfzMGenn9sNh6IHYpSoUXZe-1i5lek2F9AQ"
	algo := jwt.NewEd25519(jwt.Ed25519PublicKey(GetSamplePublic()))

	var plout CustomPayload
	hd, err := jwt.Verify([]byte(ticket), algo, &plout)
	if err != nil {
		// ...
		fmt.Println("verify err=", err)
		got = err.Error()
	}
	fmt.Println(hd)

	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

}

func TestMakeTicket(t *testing.T) {

	algo := jwt.NewEd25519(jwt.Ed25519PrivateKey(GetSamplePrivate()))

	fmt.Println("algo=", algo)

	now := time.Now()
	pl := CustomPayload{
		Payload: jwt.Payload{
			Issuer: "atw",
			//	Subject:        "knotfree",
			//	Audience:       jwt.Audience{"https://golang.org", "https://jwt.io"},
			ExpirationTime: jwt.NumericDate(now.Add(24 * 30 * 12 * time.Hour)),
			//	NotBefore:      jwt.NumericDate(now.Add(30 * time.Minute)),
			//	IssuedAt:       jwt.NumericDate(now),
			JWTID: "foobar",
		},
		//Foo: "foo",
		Bar: 1337,
	}

	token, err := jwt.Sign(pl, algo)
	if err != nil {
		// ...
	}

	fmt.Println("token=", string(token))

	algoPublic := jwt.NewEd25519(jwt.Ed25519PublicKey(GetSamplePublic()))

	var plout CustomPayload
	hd, err := jwt.Verify(token, algoPublic, &plout)
	if err != nil {
		// ...
		fmt.Println("verify err=", err)
	}
	fmt.Println(hd)

}

func TestMakeTicket000(t *testing.T) {

	var hs = jwt.NewHS256([]byte("secret"))

	sh := sha256.New()
	sh.Write([]byte("secret"))
	shabytes := sh.Sum(nil)

	fmt.Println(hex.EncodeToString(shabytes))

	now := time.Now()
	pl := CustomPayload{
		Payload: jwt.Payload{
			Issuer:         "gbrlsnchs",
			Subject:        "someone",
			Audience:       jwt.Audience{"https://golang.org", "https://jwt.io"},
			ExpirationTime: jwt.NumericDate(now.Add(24 * 30 * 12 * time.Hour)),
			NotBefore:      jwt.NumericDate(now.Add(30 * time.Minute)),
			IssuedAt:       jwt.NumericDate(now),
			JWTID:          "foobar",
		},
		//Foo: "foo",
		Bar: 1337,
	}

	token, err := jwt.Sign(pl, hs)
	if err != nil {
		// ...
	}

	fmt.Println("token=", string(token))

	var plout CustomPayload
	hd, err := jwt.Verify(token, hs, &plout)
	if err != nil {
		// ...
		fmt.Println("verify err=", err)
	}
	fmt.Println(hd)

}

type AtwEd int

func ExampleZeroReader() {

	var zero tokens.ZeroReader
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

	public, private, _ := ed25519.GenerateKey(rand.Reader)

	fmt.Println(base64.StdEncoding.WithPadding(base64.NoPadding).EncodeToString(public))
	fmt.Println(base64.StdEncoding.WithPadding(base64.NoPadding).EncodeToString(private))

	if os.Getenv("KNOT_KUNG_FOO") == "xxxxxatw" {
		_, err := os.Stat("./publicKeys.txt")
		if os.IsNotExist(err) {
			puf, err := os.Create("./publicKeys.txt")
			if err != nil {
				fmt.Println("fail 6")
			}
			defer puf.Close()
			prf, err := os.Create("privateKeys.txt")
			if err != nil {
				fmt.Println("fail 7")
			}
			defer prf.Close()

			for i := 0; i < 64; i++ {
				public, private, _ := ed25519.GenerateKey(rand.Reader)
				pu := base64.RawStdEncoding.EncodeToString(public)
				pr := base64.RawStdEncoding.EncodeToString(private)
				puf.WriteString(pu + "\n")
				prf.WriteString(pr + "\n")

			}
		}
	}

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

	public = tokens.ParseOpenSSHPublicKey(public)

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

	private = tokens.ParseOpenSSHPrivateKey(private)

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

// never use these
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
func GetSampleTokenPayload(startTime uint32) *tokens.KnotFreeTokenPayload {
	p := &tokens.KnotFreeTokenPayload{}
	p.Issuer = "1iVt" // first 4 from public
	p.ExpirationTime = startTime + 60*60*24*(365+1)
	p.JWTID = "123456"
	p.Input = 20
	p.Output = 20
	p.Subscriptions = 2
	p.Connections = 2
	p.URL = "knotfree.net"
	return p
}

// GetSampleBigToken is used for testing.
func GetSampleBigToken(startTime uint32) *tokens.KnotFreeTokenPayload {
	p := &tokens.KnotFreeTokenPayload{}
	p.Issuer = "1iVt" // first 4 from public
	p.ExpirationTime = startTime + 60*60*24*(365+1)
	p.JWTID = "123457"
	p.Input = 1e6
	p.Output = 1e6
	p.Subscriptions = 200
	p.Connections = 200
	p.URL = "knotfree.net"
	return p
}

// 123480 ns/op	    1248 B/op	      22 allocs/op  	~8000/sec
func BenchmarkCheckToken(b *testing.B) {
	ticket := []byte("eyJhbGciOiJFZDI1NTE5IiwidHlwIjoiSldUIn0.eyJleHAiOjE2MDk0NjI4MDAsImlzcyI6IjFpVnQiLCJqdGkiOiIxMjM0NTYiLCJpbiI6NzAwMDAsIm91dCI6NzAwMDAsInN1IjoyLCJjbyI6Mn0.N22xJiYz_FMQu_nG_cxlQk7gnvbeO9zOiuzbkZYWpxSzAPtQ_WyCVwWYBPZtA-0Oj-AggWakTNsmGoe8JIzaAg")
	publicKey := GetSamplePublic()
	// run the verify function b.N times
	for n := 0; n < b.N; n++ {

		p, ok := tokens.VerifyToken(ticket, publicKey)
		_ = p
		_ = ok

	}
}

// this is not especially quick
// 122662 ns/op	    1088 B/op	      19 allocs/op 	~8000/sec
func BenchmarkCheckToken2(b *testing.B) {
	ticket := []byte("eyJhbGciOiJFZDI1NTE5IiwidHlwIjoiSldUIn0.eyJleHAiOjE2MDk0NjI4MDAsImlzcyI6IjFpVnQiLCJqdGkiOiIxMjM0NTYiLCJpbiI6NzAwMDAsIm91dCI6NzAwMDAsInN1IjoyLCJjbyI6Mn0.N22xJiYz_FMQu_nG_cxlQk7gnvbeO9zOiuzbkZYWpxSzAPtQ_WyCVwWYBPZtA-0Oj-AggWakTNsmGoe8JIzaAg")
	publicKey := GetSamplePublic()
	payload := tokens.KnotFreeTokenPayload{}
	algo := jwt.NewEd25519(jwt.Ed25519PublicKey(publicKey))

	// run the verify function b.N times
	for n := 0; n < b.N; n++ {

		hd, err := jwt.Verify([]byte(ticket), algo, &payload)
		_ = hd
		_ = err
		if payload.Connections != 2 {
			fmt.Println("wrong")
		}
		payload.Connections = -1

	}
}

func TestMakeTok2(t *testing.T) {

	tokens.LoadPublicKeys()

	tokens.LoadPrivateKeys("~/atw/privateKeys4.txt")
	signingKey := tokens.GetPrivateKey("/9sh")

	payload := GetSampleTokenPayload(starttime)
	payload.Issuer = "/9sh"

	tok, err := tokens.MakeToken(payload, []byte(signingKey))
	fmt.Println("tok is ", tok, err)

	_, ok := tokens.VerifyToken(tok, []byte(tokens.FindPublicKey("/9sh")))

	fmt.Println("OK", ok)

}

func TestBox(t *testing.T) {

	counter := &tokens.CountReader{}

	// client
	clipub, clipriv, c := box.GenerateKey(counter)

	// server
	serpub, serpriv, g := box.GenerateKey(counter)
	_ = c
	_ = g

	tokens.LoadPublicKeys()

	tokens.LoadPrivateKeys("~/atw/privateKeys4.txt")
	signingKey := tokens.GetPrivateKey("/9sh")

	payload := GetSampleTokenPayload(starttime)
	payload.Issuer = "/9sh"
	payload.JWTID = getRandomB64String() // has len = 24

	tok, err := tokens.MakeToken(payload, []byte(signingKey))
	fmt.Println("tok is ", tok, err)

	// box it up
	boxout := make([]byte, len(tok)+box.Overhead+99)
	boxout = boxout[:0]
	// Seal appends an encrypted and authenticated copy of message to out, which
	// will be Overhead bytes longer than the original and must not overlap it. The
	// nonce must be unique for each distinct message for a given pair of keys.
	//func Seal(out, message []byte, nonce *[24]byte, peersPublicKey, privateKey *[32]byte) []byte {

	var jwtid [24]byte
	copy(jwtid[:], []byte(payload.JWTID))

	sealed := box.Seal(boxout, tok, &jwtid, clipub, serpriv)

	/////
	//  send the sealed and the nonce and the server pub key to the client
	/////

	// Open authenticates and decrypts a box produced by Seal and appends the
	// message to out, which must not overlap box. The output will be Overhead
	// bytes smaller than box.
	//func Open(out, box []byte, nonce *[24]byte, peersPublicKey, privateKey *[32]byte) ([]byte, bool) {
	openbuffer := make([]byte, len(tok)*2)
	opened, ok := box.Open(openbuffer, sealed, &jwtid, serpub, clipriv)
	if !ok {
		fmt.Println("OK 1 not ok ", ok)
	}

	_, ok = tokens.VerifyToken(opened, []byte(tokens.FindPublicKey("/9sh")))

	fmt.Println("OK", ok)

}

func getRandomB64String() string {
	var tmp [18]byte
	rand.Read(tmp[:])
	return base64.RawStdEncoding.EncodeToString(tmp[:])
}
