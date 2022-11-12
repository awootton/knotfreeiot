package tokens_test

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math/rand"
	"testing"

	"github.com/awootton/knotfreeiot/tokens"
	"golang.org/x/crypto/curve25519"
	"golang.org/x/crypto/nacl/box"
)

//const starttime = uint32(1577840400) // Wednesday, January 1, 2020 1:00:00 AM

func TestMakingKeys(t *testing.T) {

	{
		masterPassPhrase := "any string at all" // say (2^14 words)^6 = 2^84. What's the target? 2^64? 128?
		hash := sha256.Sum256([]byte(masterPassPhrase))

		publicKey := new([32]byte)
		privateKey := new([32]byte)

		copy(privateKey[:], hash[:])

		curve25519.ScalarBaseMult(publicKey, privateKey)

		fmt.Println(" passphrase ", masterPassPhrase, " private ", "="+base64.RawURLEncoding.EncodeToString(privateKey[:]), " \npublic ", "="+base64.RawURLEncoding.EncodeToString(publicKey[:]))
	}

	{
		humanName := "KdsbMweME9Dw29FKooIDAd0qtZIH_ethPBmk3r0UGzk_playground_light" // say (2^14 words)^6 = 2^84. What's the target? 2^64? 128?

		// for the client we cannot use a passphrase
		randomName := make([]byte, 32)
		rand.Read(randomName)
		hash := sha256.Sum256([]byte(randomName))

		publicKey := new([32]byte)
		privateKey := new([32]byte)

		copy(privateKey[:], hash[:])

		curve25519.ScalarBaseMult(publicKey, privateKey)

		fmt.Println(" passphrase ", humanName, " private ", "="+base64.RawURLEncoding.EncodeToString(privateKey[:]), " \npublic ", "="+base64.RawURLEncoding.EncodeToString(publicKey[:]))
	}

}

func TestBox1(t *testing.T) {

	masterPassPhrase := "socks rocks foxy blocks over join" // say (2^14 words)^6 = 2^84. What's the target? 2^64? 128?
	hash := sha256.Sum256([]byte(masterPassPhrase))

	publicKey := new([32]byte)
	privateKey := new([32]byte)

	copy(privateKey[:], hash[:])

	curve25519.ScalarBaseMult(publicKey, privateKey)

	fmt.Println("private", hex.EncodeToString(privateKey[:]), "public", hex.EncodeToString(publicKey[:]))
	// a6d8051b972129f473b2f0865282d0b505a23a8c3c8c678bdd4b0cea33d2df0d
	// b8bbe796e2839c6ed69ca757847c9be0319f061d755d5bd23064a34b86075958

	devicePassPhrase := hex.EncodeToString(privateKey[:]) + "tramp light"
	devicePrivateKey := sha256.Sum256([]byte(devicePassPhrase))
	devicePublicKey := new([32]byte)

	curve25519.ScalarBaseMult(devicePublicKey, &devicePrivateKey)

	fmt.Println("dev private", hex.EncodeToString(devicePrivateKey[:]), "dev public", hex.EncodeToString(devicePublicKey[:]))
	// e214a049cf64bb02004557e7343c906f0c590403e20b4074585cf533dbda6108
	// d1eaa91d2a681dc02d357132b6a83e6140230a951ca72f55a0ce757527fae431

	message := "light on for 20 minutes"
	tmp := sha256.Sum256([]byte("irl this would be random"))
	nonce := new([24]byte)
	copy(nonce[:], tmp[:])

	buffer := make([]byte, 0, (len(message) + box.Overhead))
	sealed := box.Seal(buffer, []byte(message), nonce, devicePublicKey, privateKey)

	//fmt.Println("encrypted message ", hex.EncodeToString(out))
	fmt.Println("encrypted message", hex.EncodeToString(sealed))

	// send out, and nonce to device
	openbuffer := make([]byte, 0, (len(sealed))) // - box.Overhead))

	opened, ok := box.Open(openbuffer, sealed, nonce, publicKey, &devicePrivateKey)

	fmt.Println("decrypted message hex", hex.EncodeToString(opened), ok)
	fmt.Println("decrypted message ", string(opened))
	//fmt.Println("decrypted message ", hex.EncodeToString(result))
}

func TestBoxMatchesTypeScript(t *testing.T) {

	senderPass := "testString123" //
	hash := sha256.Sum256([]byte(senderPass))

	fmt.Println(senderPass, "hashes to ", base64.RawURLEncoding.EncodeToString(hash[:]))

	receiverPass := "myFamousOldeSaying" //
	hash = sha256.Sum256([]byte(receiverPass))

	fmt.Println(receiverPass, "hashes to ", base64.RawURLEncoding.EncodeToString(hash[:]))

	spublic, sprivate := tokens.GetBoxKeyPairFromPassphrase(senderPass)

	fmt.Println(senderPass, "makes sender public key ", base64.RawURLEncoding.EncodeToString(spublic[:]))
	fmt.Println(senderPass, "makes sender private key ", base64.RawURLEncoding.EncodeToString(sprivate[:]))
	//testString123 makes sender public key   bht-Ka3j7GKuMFOablMlQnABnBvBeugvSf4CdFV3LXs
	//testString123 makes sender secret key   VY5e4pCAwDlr-HdfioX6TCiv41Xx_SsTtUcupKndFpQ

	rpublic, rprivate := tokens.GetBoxKeyPairFromPassphrase(receiverPass)

	fmt.Println(receiverPass, "makes receiver public key ", base64.RawURLEncoding.EncodeToString(rpublic[:]))
	fmt.Println(receiverPass, "makes receiver private key ", base64.RawURLEncoding.EncodeToString(rprivate[:]))

	message := "this is my test message"

	nonceSlice := []byte("EhBJOkFN3CjwqBGzkSurniXj")
	var nonce [24]byte
	copy(nonce[:], nonceSlice[:])

	buffer := make([]byte, 0, (len(message) + box.Overhead))
	sealed := box.Seal(buffer, []byte(message), &nonce, &rpublic, &sprivate)

	fmt.Println("boxed", base64.RawURLEncoding.EncodeToString(sealed))

	// send out, and nonce to device
	openbuffer := make([]byte, 0, (len(sealed))) // - box.Overhead))

	opened, ok := box.Open(openbuffer, sealed, &nonce, &spublic, &rprivate)
	_ = ok
	fmt.Println("unboxed ", string(opened))

}
