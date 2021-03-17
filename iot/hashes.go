// Copyright 2019,2020 Alan Tracey Wootton
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

package iot

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"math/rand"
	"strconv"
)

// HashType will be the key
// I'm increasing it to 20 2/2020 atw or 24 2/2021
type notHashType struct {
	bytes [24]byte
}

// HashTypeLen now it's 24 bytes long
const HashTypeLen = 24

// HashType is for the hash table that Lookup uses.
type HashType [3]uint64

// GetHalfHash is for cases when we can do with 'just' 64 bits.
func (h *HashType) GetHalfHash() HalfHash {
	return HalfHash(h[0])
}

// GetUint64 is for cases when we can do with 'just' 64 bits.
func (h *HashType) GetUint64() uint64 {
	return h[0]
}

// HashString will hash the string and init the HashType
func (h *HashType) HashString(s string) {
	h.HashBytes([]byte(s))
}

// HashNameToAliasXXX checks for the name being utf8 or hex of a hash or base64 of a hash or binary.
// in the case of binary do nothing.
func HashNameToAliasXXX(name []byte) ([]byte, error) {
	if name[0] == '`' && len(name) == (HashTypeLen+1) {
		return name, nil
	}
	switch name[0] {
	case '$':
		if len(name) != HashTypeLen*2+1 {
			return name, errors.New("requires 48 bytes of hex")
		}

		return name, nil
	case '=':
		return name, nil
	default:
		// is utf8. Hash it.
		sh := sha256.New()
		sh.Write(name)
		shabytes := sh.Sum(nil)
		// move everything up by one
		for i := 0; i < HashTypeLen; i++ {
			shabytes[HashTypeLen-i+1] = shabytes[HashTypeLen-i]
		}
		shabytes[0] = '`'
		return shabytes[0 : HashTypeLen+1], nil
	}
}

// HashBytes will initialize an existing hash from a string.
// The string will get hashed to provide the bits so we'll wish this was faster.
// It doesn't have to be crypto safe but it does need to be evenly distributed.
// allocates. wanted to use highwayhash.New128 but was scared of 128 bits.
func (h *HashType) HashBytes(s []byte) {

	sh := sha256.New()
	sh.Write(s)
	shabytes := sh.Sum(nil)
	h.InitFromBytes(shabytes[0:24])
}

// InitFromBytes because I need to convert from [] to HashType
// should return error?
// rename
func (h *HashType) InitFromBytes(addressBytes []byte) {
	if len(addressBytes) != HashTypeLen {
		panic("InitFromBytes bad input")
	}

	h[0] = binary.BigEndian.Uint64(addressBytes[0:8])
	h[1] = binary.BigEndian.Uint64(addressBytes[8:16])
	h[2] = binary.BigEndian.Uint64(addressBytes[16:HashTypeLen])
}

// GetBytes will fill b byte array with value from h.
func (h *HashType) GetBytes(b []byte) {
	if len(b) < HashTypeLen {
		return // err ?
	}
	binary.BigEndian.PutUint64(b[0:8], h[0])
	binary.BigEndian.PutUint64(b[8:16], h[1])
	binary.BigEndian.PutUint64(b[16:HashTypeLen], h[2])
}

// HalfHash represents
//
type HalfHash uint64

// GetFractionalBits returns a slice of n bits. Values of n greater than 64 are not implemented.
func (h *HashType) GetFractionalBits(n int) int {
	if n < 64 {
		a := h.GetHalfHash()
		return int(a >> (64 - n))
	}
	fmt.Println("FIXME: implement GetFractionalBits for > 64")
	fmt.Println("FIXME: better idEa")
	fmt.Println("FIXME: please")
	return 0
}

var hashstartkey *[]byte

// FromHashType init an existing hash from another - basically a copy
func (h *HashType) FromHashType(src *HashType) {
	(*h) = *src
}

// Random HashType initializes with random bits.
// We don't need to hash these more do we?
func (h *HashType) Random() {

	var bytes [HashTypeLen]byte
	rand.Read(bytes[:])
	h.InitFromBytes(bytes[:])
}

func (h *HashType) String() string {
	//return hex.EncodeToString(h.bytes[0:16])
	var bytes [HashTypeLen]byte
	h.GetBytes(bytes[:])
	return base64.RawStdEncoding.EncodeToString(bytes[:])
}

func (a *HalfHash) String() string {
	return strconv.FormatUint(uint64(*a), 16)
}
