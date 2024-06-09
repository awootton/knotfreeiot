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

package iot_test

import (
	"math/rand"
	"strings"

	"encoding/base64"
	"fmt"
	"testing"
	"time"

	"github.com/awootton/knotfreeiot/iot"
	"github.com/awootton/knotfreeiot/packets"
	"github.com/awootton/knotfreeiot/tokens"
	"golang.org/x/crypto/nacl/box"
)

type MathRandReader struct{}

func (MathRandReader) Read(buf []byte) (int, error) {
	for i := range buf {
		buf[i] = byte(rand.Int())
	}
	return len(buf), nil
}

// FIXME: reserve name is broken
func not_TestReserveName(t *testing.T) {

	tokens.LoadPublicKeys()
	tokens.LoadPrivateKeys("~/atw/privateKeys4.txt")

	localtime := starttime
	getTime := func() uint32 {
		return localtime
	}

	rand.Seed(123456)

	var reader MathRandReader
	client_public, client_private, err := box.GenerateKey(reader)                             // NOT ed25519.GenerateKey(reader) which returns 64 byte secret
	fmt.Println("client_public k  ", base64.RawURLEncoding.EncodeToString(client_public[:]))  // PPWTXny1zMVx4RTQpTJ3qfZUwoefiuvl5Nk97dE7rjY
	fmt.Println("client_private k ", base64.RawURLEncoding.EncodeToString(client_private[:])) // 0n-f1uxXIEuSy8KLUC2lfjGW5iQfPSNUPkyo0ADKZMs

	reserve := tokens.SubscriptionNameReservationPayload{}
	reserve.ExpirationTime = starttime + 60*60*24*(365)*2 // 2 years from 2020 jan 1
	reserve.Issuer = "yRst"
	reserve.JWTID = base64.RawURLEncoding.EncodeToString(client_public[:])
	reserve.Name = "contact9_address"

	reserveBytes, err := tokens.MakeNameToken(&reserve, []byte(tokens.GetPrivateKeyMatching(reserve.Issuer)))

	fmt.Println("Original name jwt", string(reserveBytes))

	var nonce [24]byte
	reader.Read(nonce[:])

	clusters := StartClusterOfClusters(getTime)
	_ = clusters

	aide0 := getAnAide(clusters, 0)
	aide9 := getAnAide(clusters, 9999) // a completely different cluster, will have to go through super.

	pub := aide0.Config.GetCe().PublicKeyTemp[:]
	pri := aide0.Config.GetCe().PrivateKeyTemp[:]
	fmt.Println("knot_public k ", base64.RawURLEncoding.EncodeToString(pub))
	fmt.Println("knot_private k ", base64.RawURLEncoding.EncodeToString(pri))

	contact0 := makeTestContact(aide0.Config, "")
	contact9 := makeTestContact(aide9.Config, "")

	subs := packets.Subscribe{}
	subs.Address.FromString("contact0_address")
	iot.PushPacketUpFromBottom(contact0, &subs)

	connect := packets.Connect{}
	connect.SetOption("token", []byte(tokens.GetImpromptuGiantToken()))
	iot.PushPacketUpFromBottom(contact0, &connect)
	iot.PushPacketUpFromBottom(contact9, &connect)

	aMap := iot.GuruNameToConfigMap
	topGuruExec := aMap["guru0_1_0"]
	topGuruExecSubs, _ := topGuruExec.GetSubsCount() // "guru0_1_0" has 2 subscriptions

	// note: box.Precompute()
	// box.SealAfterPrecomputation()
	// let's just box it up. it has to be boxed by the client so we're sure it's the client using it.
	// only has to happen once per session
	reserveBoxed := make([]byte, len(reserveBytes)+box.Overhead+10)
	reserveBoxed = reserveBoxed[:0] // empty it
	ccc := cap(reserveBoxed)
	_ = ccc
	clusterPubk := aide0.Config.GetCe().PublicKeyTemp
	box_bytes := box.Seal(reserveBoxed, reserveBytes, &nonce, clusterPubk, client_private)

	subs = packets.Subscribe{}
	subs.Address.FromString("contact9_address")
	subs.SetOption("pubk", client_public[:]) // pub key of client required
	subs.SetOption("tokn", []byte(tokens.GetImpromptuGiantToken()))
	subs.SetOption("reserved", box_bytes)
	subs.SetOption("nonce", nonce[:])
	// let's check the unbox right here right now.
	{
		// we have subs
		box_bytes2, _ := subs.GetOption("reserved")
		pubk, _ := subs.GetOption("pubk")
		//tokn, _ := subs.GetOption("tokn")
		nonce, _ := subs.GetOption("nonce")
		var nonce2 [24]byte
		copy(nonce2[:], nonce)
		var pubk2 [32]byte
		copy(pubk2[:], pubk)

		clusterSecret := aide0.Config.GetCe().PrivateKeyTemp

		dest_buffer := make([]byte, len(box_bytes2)-box.Overhead)
		dest_buffer = dest_buffer[:0]
		open_bytes, err := box.Open(dest_buffer, box_bytes2, &nonce2, &pubk2, clusterSecret)
		_ = err
		// this should be our original jwt for a name res
		fmt.Println("recovered name jwt", string(open_bytes))

		publicKeyBytes := tokens.FindPublicKey("yRst")
		namePayload, ok := tokens.VerifyNameToken([]byte(open_bytes), []byte(publicKeyBytes))
		if !ok {
			t.Errorf("got %v, want %v", "false", "true")
		}
		fmt.Println("payload of name token ", namePayload)

		// and here's the trick
		// the public key in the namePayload must
		// match the pubk for the box

		if namePayload.JWTID != base64.RawURLEncoding.EncodeToString(pubk) {
			t.Errorf("got %v, want %v", base64.RawURLEncoding.EncodeToString(pubk), namePayload.JWTID)
		}

		// also the names must match
		if strings.Trim(subs.Address.String(), " ") != namePayload.Name {
			t.Errorf("got '%v', want '%v'", subs.Address.String(), namePayload.Name)
		}
	}

	time.Sleep(100 * time.Millisecond)
	time.Sleep(100 * time.Millisecond)
	time.Sleep(100 * time.Millisecond)
	time.Sleep(100 * time.Millisecond)

	//subs.SetOption("debg", []byte("12345678"))
	err = iot.PushPacketUpFromBottom(contact9, &subs)
	if err != nil {
		t.Error("got error ")
	}

	time.Sleep(100 * time.Millisecond)
	time.Sleep(100 * time.Millisecond)
	time.Sleep(100 * time.Millisecond)
	time.Sleep(100 * time.Millisecond)

	IterateAndWait(t, func() bool {
		cnt, _ := topGuruExec.GetSubsCount()
		return cnt > topGuruExecSubs
	}, "timed out waiting for sub to move up")
	topGuruExecSubs, _ = topGuruExec.GetSubsCount() // now it's 3

	// now unsubscribe
	unsub := packets.Unsubscribe{}
	unsub.Address.FromString("contact9_address")
	err = iot.PushPacketUpFromBottom(contact9, &unsub)

	time.Sleep(100 * time.Millisecond)
	time.Sleep(100 * time.Millisecond)
	time.Sleep(100 * time.Millisecond)
	time.Sleep(100 * time.Millisecond)

	// and then look at it
	look := packets.Lookup{}
	look.Address.FromString("contact9_address")
	look.Source.FromString("contact0_address")
	//look.SetOption("debg", []byte("12345678"))
	iot.PushPacketUpFromBottom(contact0, &look)

	got := ""
	count := 0
	IterateAndWait(t, func() bool {
		got, ok := contact0.(*testContact).popResultAsString()
		if ok {
			fmt.Println("packets.Lookup 4 reserve got ", got)
			count++
		}
		return count >= 4
	}, "timed out waiting for look message 2 arrive")
	fmt.Println("lookup was " + got)

}

// TestLookup still doesn't actually make a reservatiom
func TestLookupSubs(t *testing.T) {

	localtime := starttime
	getTime := func() uint32 {
		return localtime
	}

	rand.Seed(123456)

	clusters := StartClusterOfClusters(getTime)
	_ = clusters

	aide0 := getAnAide(clusters, 0)
	aide9 := getAnAide(clusters, 9999) // a completely different cluster, will have to go through super.

	contact0 := makeTestContact(aide0.Config, "")
	contact9 := makeTestContact(aide9.Config, "")

	connect := packets.Connect{}
	connect.SetOption("token", tokens.GetTest32xToken())
	iot.PushPacketUpFromBottom(contact0, &connect)
	iot.PushPacketUpFromBottom(contact9, &connect)

	aMap := iot.GuruNameToConfigMap
	topGuruExec := aMap["guru0_1_0"]
	topGuruExecSubs, _ := topGuruExec.GetSubsCount() // "guru0_1_0" has 2 subscriptions

	subs := packets.Subscribe{}
	subs.Address.FromString("contact0_address")
	//subs.SetOption("debg", []byte("12345678"))
	err := iot.PushPacketUpFromBottom(contact0, &subs)
	if err != nil {
		t.Error("got error ")
	}

	IterateAndWait(t, func() bool {
		cnt, _ := topGuruExec.GetSubsCount()
		return cnt > topGuruExecSubs
	}, "timed out waiting for sub to move up")
	topGuruExecSubs, _ = topGuruExec.GetSubsCount()

	subs = packets.Subscribe{}
	subs.Address.FromString("contact9_address")
	//subs.SetOption("debg", []byte("12345678"))
	err = iot.PushPacketUpFromBottom(contact9, &subs)
	if err != nil {
		t.Error("got error ")
	}

	IterateAndWait(t, func() bool {
		cnt, _ := topGuruExec.GetSubsCount()
		return cnt > topGuruExecSubs
	}, "timed out waiting for sub to move up")
	topGuruExecSubs, _ = topGuruExec.GetSubsCount() // now it's 4

	{
		sendmessage := packets.Send{}
		sendmessage.Address.FromString("contact9_address")
		sendmessage.Source.FromString("contact0_address")
		sendmessage.Payload = []byte("can you hear me now?")
		//sendmessage.SetOption("debg", []byte("12345678"))
		err = iot.PushPacketUpFromBottom(contact0, &sendmessage)
		if err != nil {
			t.Error("got error ")
		}
		got := ""
		ok := false
		IterateAndWait(t, func() bool {
			got, ok = contact9.(*testContact).popResultAsString()
			if ok {
				lll := contact9.(*testContact).getResultsCount()
				if lll != 0 {
					t.Errorf("got %v, want %v", lll, 0)
				}
			}
			return ok
		}, "timed out waiting for can you hear me now")

		fmt.Println("reply was " + got)
		want := `[P,=6X2eixvv3rz9Irvi85t2S5gdA0tRfB0B,contact0_address,"can you hear me now?"]`
		if got != want {
			t.Errorf("got %v, want %v", got, want)
		}
	}
	{
		// fmt.Println("pause here for a sec ")
		// time.Sleep(time.Second)
		// fmt.Println("moving along after a sec ")

		// send again
		sendmessage := packets.Send{}
		sendmessage.Address.FromString("contact9_address")
		sendmessage.Source.FromString("contact0_address")
		sendmessage.Payload = []byte("can you 2 hear me now2 ?")
		//sendmessage.SetOption("debg", []byte("12345678"))
		err = iot.PushPacketUpFromBottom(contact0, &sendmessage)
		if err != nil {
			t.Error("got error ")
		}

		got := ""
		ok := false
		IterateAndWait(t, func() bool {
			got, ok = contact9.(*testContact).popResultAsString()
			if ok {
				time.Sleep(100 * time.Millisecond)
				lll := contact9.(*testContact).getResultsCount()
				if lll != 0 {
					t.Errorf("got %v, want %v", lll, 0)
				}
			}
			return ok
		}, "timed out waiting2 for can you hear me now2")

		fmt.Println("reply2 was " + got)
		want := `[P,=6X2eixvv3rz9Irvi85t2S5gdA0tRfB0B,contact0_address,"can you 2 hear me now2 ?"]`
		if got != want {
			t.Errorf("got %v, want %v", got, want)
		}
	}

	{ // send from 0 to 9
		sendmessage := packets.Send{}
		sendmessage.Address.FromString("contact0_address")
		sendmessage.Source.FromString("contact9_address")
		sendmessage.Payload = []byte("message from 0 to 9. message from 0 to 9. ")
		//sendmessage.SetOption("debg", []byte("12345678"))
		err = iot.PushPacketUpFromBottom(contact9, &sendmessage)
		if err != nil {
			t.Error("got error ")
		}
		got := ""
		ok := false
		IterateAndWait(t, func() bool {
			got, ok = contact0.(*testContact).popResultAsString()
			if ok {
				time.Sleep(100 * time.Millisecond)
				lll := contact0.(*testContact).getResultsCount()
				if lll != 0 {
					t.Errorf("got %v, want %v", lll, 0)
				}
			}
			return ok
		}, "timed out waiting for message from 0 to 9. message from 0 to 9. ")
		fmt.Println("reply3 was " + got)
		want := `[P,=OKU2ncwOXF7_pEK8QM-duSHlTzE7jvDe,contact9_address,"message from 0 to 9. message from 0 to 9. "]`
		if got != want {
			t.Errorf("got %v, want %v", got, want)
		}
	}

	fmt.Println("------------------------------------")

	look := packets.Lookup{}
	look.Address.FromString("contact9_address")
	look.Source.FromString("contact0_address")
	look.SetOption("debg", []byte("12345678"))
	iot.PushPacketUpFromBottom(contact0, &look)

	got := ""
	count := 0
	IterateAndWait(t, func() bool {
		got, ok := contact0.(*testContact).popResultAsString()
		if ok {
			fmt.Println("packets.Lookup got ", got)
			count++
		}
		return count >= 4
	}, "timed out waiting for look message to arrive")
	fmt.Println("lookup was " + got)
}
