package iot

import (
	"bytes"
	"fmt"
	"reflect"

	"github.com/awootton/knotfreeiot/packets"
)

// check a send packet so we know it will transmit fully and read back

// This allocates a mb buffer every time so turn it off when you're done. ok?

// TODO: (atw) make the tests in stressLoadNames_test.go pass with this turned off. FIXME: (atw) wtaf.
// I don't know why. I think it's because the sessionKey is a slice of the backing array and the backing array is being reused. So we copy it to a new slice and set it back in the packet. This is superstitious, but it works.

const turnOffTheseCheckers = false // before prod or something.

func CheckSendPacket(send *packets.Send) bool {

	if turnOffTheseCheckers {
		return true
	}

	// normalize the address first.
	send.Address.EnsureAddressIsBinary()

	// // is this send creepy somehow?
	// // let's send it to a buffer and see if we can read it back.
	var buf bytes.Buffer
	buf.Grow(1024 * 1024) // 1MB
	err := send.Write(&buf)
	if err != nil {
		fmt.Println("processLookup sendReply ERROR marshaling send:", err)
		return false
	}
	data := buf.Bytes()
	bufReader := bytes.NewReader(data)

	gotpacket, err := packets.ReadPacket(bufReader)
	if err != nil {
		fmt.Println("processLookup sendReply ERROR reading packet:", err)
		return false
	}
	var recvSend, ok = gotpacket.(*packets.Send)
	if !ok {
		fmt.Println("processLookup sendReply ERROR gotpacket is not of type *packets.Send")
		return false
	}
	address := recvSend.Address.ToBytes()
	source := recvSend.Source.ToBytes()
	payload := recvSend.Payload
	if !bytes.Equal(address, []byte(send.Address.ToBytes())) {
		fmt.Println("processLookup sendReply ERROR address mismatch. got:", string(address), "want:", string(send.Address.ToBytes()))
		return false
	}
	if !bytes.Equal(source, []byte(send.Source.ToBytes())) {
		fmt.Println("processLookup sendReply ERROR source mismatch. got:", string(source), "want:", string(send.Source.ToBytes()))
		return false
	}
	if !bytes.Equal(payload, []byte(send.Payload)) {
		fmt.Println("processLookup sendReply ERROR payload mismatch. got:", string(payload), "want:", string(send.Payload))
		return false
	}

	keys, values := recvSend.GetOptionKeys()
	sendkeys, sendvalues := send.GetOptionKeys()
	if !reflect.DeepEqual(keys, sendkeys) {
		fmt.Println("processLookup sendReply ERROR option keys mismatch. got:", keys, "want:", sendkeys)
		return false
	}
	if !reflect.DeepEqual(values, sendvalues) {
		fmt.Println("processLookup sendReply ERROR option values mismatch. got:", values, "want:", sendvalues)
		return false
	}
	const printkeys = false
	if printkeys {
		fmt.Println("processLookup original option keys and values:")
		for i := 0; i < len(sendkeys); i++ {
			fmt.Println("    key is ", string(sendkeys[i]), "val is ", string(sendvalues[i]))
		}
		fmt.Println("processLookup recvSend option keys and values:")
		for i := 0; i < len(keys); i++ {
			fmt.Println("    key is ", string(keys[i]), "val is ", string(values[i]))
		}
	}

	return true
}

// for some reason, adding makes stressLoadNames_test.go work.
// always? REally !?!
func CheckSendPacketAlways(send *packets.Send) bool {

	if turnOffTheseCheckers { // and, ...., um, .... , no.
		return true
	}

	// // is this send creepy somehow?
	// // let's send it to a buffer and see if we can read it back.
	var buf bytes.Buffer
	buf.Grow(1024 * 1024) // 1MB
	err := send.Write(&buf)
	if err != nil {
		fmt.Println("processLookup sendReply ERROR marshaling send:", err)
		return false
	}
	data := buf.Bytes()
	bufReader := bytes.NewReader(data)

	gotpacket, err := packets.ReadPacket(bufReader)
	if err != nil {
		fmt.Println("processLookup sendReply ERROR reading packet:", err)
		return false
	}
	var recvSend, ok = gotpacket.(*packets.Send)
	if !ok {
		fmt.Println("processLookup sendReply ERROR gotpacket is not of type *packets.Send")
		return false
	}
	address := recvSend.Address.ToBytes()
	source := recvSend.Source.ToBytes()
	payload := recvSend.Payload
	if !bytes.Equal(address, []byte(send.Address.ToBytes())) {
		fmt.Println("processLookup sendReply ERROR address mismatch. got:", string(address), "want:", string(send.Address.ToBytes()))
		return false
	}
	if !bytes.Equal(source, []byte(send.Source.ToBytes())) {
		fmt.Println("processLookup sendReply ERROR source mismatch. got:", string(source), "want:", string(send.Source.ToBytes()))
		return false
	}
	if !bytes.Equal(payload, []byte(send.Payload)) {
		fmt.Println("processLookup sendReply ERROR payload mismatch. got:", string(payload), "want:", string(send.Payload))
		return false
	}

	keys, values := recvSend.GetOptionKeys()
	sendkeys, sendvalues := send.GetOptionKeys()
	if !reflect.DeepEqual(keys, sendkeys) {
		fmt.Println("processLookup sendReply ERROR option keys mismatch. got:", keys, "want:", sendkeys)
		return false
	}
	if !reflect.DeepEqual(values, sendvalues) {
		fmt.Println("processLookup sendReply ERROR option values mismatch. got:", values, "want:", sendvalues)
		return false
	}
	// fmt.Println("processLookup sendReply option keys and values:")
	// for i := 0; i < len(keys); i++ {
	// 	fmt.Println("    key is ", string(keys[i]), "val is ", string(values[i]))
	// }
	return true
}

func CheckLookupPacket(send *packets.Lookup) bool {

	// is this send creepy somehow?
	// let's send it to a buffer and see if we can read it back.
	if turnOffTheseCheckers {
		return true
	}

	var buf bytes.Buffer
	buf.Grow(1024 * 1024) // 1MB
	err := send.Write(&buf)
	if err != nil {
		fmt.Println("processLookup sendReply ERROR marshaling send:", err)
		return false
	}
	data := buf.Bytes()
	bufReader := bytes.NewReader(data)

	gotpacket, err := packets.ReadPacket(bufReader)
	if err != nil {
		fmt.Println("processLookup sendReply ERROR reading packet:", err)
		return false
	}
	var recvSend, ok = gotpacket.(*packets.Lookup)
	if !ok {
		fmt.Println("processLookup sendReply ERROR gotpacket is not of type *packets.Lookup")
		return false
	}
	address := recvSend.Address.ToBytes()
	source := recvSend.Source.ToBytes()
	// payload := recvSend.Payload
	if !bytes.Equal(address, []byte(send.Address.ToBytes())) {
		fmt.Println("processLookup sendReply ERROR address mismatch. got:", string(address), "want:", string(send.Address.ToBytes()))
		return false
	}
	if !bytes.Equal(source, []byte(send.Source.ToBytes())) {
		fmt.Println("processLookup sendReply ERROR source mismatch. got:", string(source), "want:", string(send.Source.ToBytes()))
		return false
	}
	// if !bytes.Equal(payload, []byte(send.Payload)) {
	// 	fmt.Println("processLookup sendReply ERROR payload mismatch. got:", string(payload), "want:", string(send.Payload))
	// 	return false
	// }

	keys, values := recvSend.GetOptionKeys()
	sendkeys, sendvalues := send.GetOptionKeys()
	if !reflect.DeepEqual(keys, sendkeys) {
		fmt.Println("processLookup sendReply ERROR option keys mismatch. got:", keys, "want:", sendkeys)
		return false
	}
	if !reflect.DeepEqual(values, sendvalues) {
		fmt.Println("processLookup sendReply ERROR option values mismatch. got:", values, "want:", sendvalues)
		return false
	}
	// fmt.Println("processLookup sendReply option keys and values:")
	// for i := 0; i < len(keys); i++ {
	// 	fmt.Println("    key is ", string(keys[i]), "val is ", string(values[i]))
	// }

	return true
}

// RewriteSessionKeyToCheck pulls out the session value. Copies it. Turn this off later.
// When these are read from the network the value is a slice of the backing array.
// If the backing array is reused, the value will change (which is NEVER supposed to happen).
// So we copy it to a new slice and set it back in the packet. Superstitution, really.
func RewriteSessionKeyToCheck(p *packets.Send) {

	if turnOffTheseCheckers {
		return
	}

	if sessionValue, ok := p.GetOption(SessionKeyString); !ok {
		fmt.Println("ERROR processLookup lookmsg hasn't sessionKey") // doesn't happen. Does it?
	} else {

		// fmt.Println("processLookup sendReply sessionKey is ", string(sessionValue)) // doesn't happen. Does it?

		// Have it. I'm going to totally copy it. Just in case something is happening to the original backing array or something.
		// Hail Mary, full of grace,
		// the Lord is with thee. Blessed art thou among packets, and blessed is the fruit of thy payload,
		// Jesus. Holy Mary, Mother of God, pray for us sinners, now and at the hour of our death. Amen.
		keylen := len(sessionValue) // actually 28. It's always 28 and 'fetch' will never become a thing.
		if keylen != 28 {
			fmt.Println("processLookup sendReply ERROR sessionKey length is not 28. Is today 10/31? 4/1? 4/20? It's ", keylen)
		}
		newSessionKeyVal := make([]byte, 28) // actually 28, on the heap, not here in the stack frames.
		copied := copy(newSessionKeyVal[:], sessionValue)
		if copied != 28 {
			fmt.Println("processLookup sendReply ERROR sessionKey copy length is not 28. Is today 10/31? 4/1? 4/20? It's ", copied)
		}

		// they are the same. This is silly.
		// fmt.Println("processLookup sendReply newSessionKeyVal is ", string(newSessionKeyVal)) // doesn't happen. Does it?

		p.SetOption(SessionKeyString, newSessionKeyVal[:])
	}
}

// RewriteSessionKeyToCheck pulls out the session value. Copies it. Turn this off later.
// When these are read from the network the value is a slice of the backing array.
// If the backing array is reused, the value will change (which is NEVER supposed to happen).
// So we copy it to a new slice and set it back in the packet. Superstitution, really.

func RewriteSessionKeyToCheckLookup(p *packets.Lookup) {

	if turnOffTheseCheckers {
		return
	}

	if sessionValue, ok := p.GetOption(SessionKeyString); !ok {
		fmt.Println("ERROR processLookup lookmsg hasn't sessionKey") // doesn't happen. Does it?
	} else {

		// Have it. I'm going to totally copy it. Just in case something is happening to the original backing array or something.
		// Hail Mary, full of grace,
		// the Lord is with thee. Blessed art thou among packets, and blessed is the fruit of thy payload,
		// Jesus. Holy Mary, Mother of God, pray for us sinners, now and at the hour of our death. Amen.
		keylen := len(sessionValue) // actually 28. It's always 28 and 'fetch' will never become a thing.
		if keylen != 28 {
			fmt.Println("processLookup sendReply ERROR sessionKey length is not 28. Is today 10/31? 4/1? 4/20? It's ", keylen)
		}
		newSessionKeyVal := make([]byte, 28) // actually 28, on the heap, not here in the stack frames.
		copied := copy(newSessionKeyVal[:], sessionValue)
		if copied != 28 {
			fmt.Println("processLookup sendReply ERROR sessionKey copy length is not 28. Is today 10/31? 4/1? 4/20? It's ", copied)
		}
		p.SetOption(SessionKeyString, newSessionKeyVal[:])
	}
}

// Copyright 2026 Alan Tracey Wootton
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
