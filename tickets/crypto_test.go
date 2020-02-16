package tickets_test

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"
	"testing"
	"time"

	"github.com/awootton/knotfreeiot/tickets"
	"github.com/gbrlsnchs/jwt/v3"
)

const starttime = uint32(1577840400) // Wednesday, January 1, 2020 1:00:00 AM

func TestKnotVerify(t *testing.T) {
	got := "ok"
	want := "ok"

	ticket := []byte("eyJhbGciOiJFZDI1NTE5IiwidHlwIjoiSldUIn0.eyJleHAiOjE2MDk0NjI4MDAsImlzcyI6IjFpVnQiLCJqdGkiOiIxMjM0NTYiLCJpbiI6NzAwMDAsIm91dCI6NzAwMDAsInN1IjoyLCJjbyI6Mn0.N22xJiYz_FMQu_nG_cxlQk7gnvbeO9zOiuzbkZYWpxSzAPtQ_WyCVwWYBPZtA-0Oj-AggWakTNsmGoe8JIzaAg")

	//p := &tickets.KnotFreePayload{}
	p, ok := tickets.VerifyTicket(ticket, tickets.GetSamplePublic())
	if !ok {
		t.Errorf("got %v, want %v", "false", "true")
	}
	bytes, err := json.Marshal(p)
	_ = err
	got = string(bytes)
	want = `{"exp":1609462800,"iss":"1iVt","jti":"123456","in":70000,"out":70000,"su":2,"co":2}`
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestKnotEncode(t *testing.T) {

	got := "ok"
	want := "ok"

	p := &tickets.KnotFreePayload{}
	p.Issuer = "1iVt" // first 4 from public
	p.ExpirationTime = starttime + 60*60*24*366
	p.JWTID = "123456"
	p.Input = 7e4
	p.Output = 7e4
	p.Subscriptions = 2
	p.Connections = 2

	bytes, err := tickets.MakeTicket(p, tickets.GetSamplePrivate())
	if err != nil {
		got = err.Error()
	}
	got = string(bytes)
	want = `eyJhbGciOiJFZDI1NTE5IiwidHlwIjoiSldUIn0.eyJleHAiOjE2MDk0NjI4MDAsImlzcyI6IjFpVnQiLCJqdGkiOiIxMjM0NTYiLCJpbiI6NzAwMDAsIm91dCI6NzAwMDAsInN1IjoyLCJjbyI6Mn0.N22xJiYz_FMQu_nG_cxlQk7gnvbeO9zOiuzbkZYWpxSzAPtQ_WyCVwWYBPZtA-0Oj-AggWakTNsmGoe8JIzaAg`
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
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
	algo := jwt.NewEd25519(jwt.Ed25519PublicKey(tickets.GetSamplePublic()))

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

	algo := jwt.NewEd25519(jwt.Ed25519PrivateKey(tickets.GetSamplePrivate()))

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

	algoPublic := jwt.NewEd25519(jwt.Ed25519PublicKey(tickets.GetSamplePublic()))

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

	var zero tickets.ZeroReader
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

	public = tickets.ParseOpenSSHPublicKey(public)

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

	private = tickets.ParseOpenSSHPrivateKey(private)

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
