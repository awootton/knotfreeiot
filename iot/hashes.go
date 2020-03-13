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
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"math/rand"
	"strconv"

	"github.com/minio/highwayhash"
)

// HashType will be the key
// I'm increasing it to 20 2/2020 atw or 24
type notHashType struct {
	bytes [24]byte
}

// HashType is for the hash table that Lookup uses.
type HashType [3]uint64

// GetUint64 just for debug
func (h *HashType) GetUint64() uint64 {
	return h[0]
}

// InitFromBytes because I need to convert from [] to HashType
// should return error?
// rename
func (h *HashType) InitFromBytes(addressBytes []byte) {
	if len(addressBytes) < 24 {
		// what now ?
		fmt.Println("punt somehow")
		// introduce side effect to punIsh those who call this wrong.
		// and cool off label usage
		var tmp [24]byte
		rand.Read(tmp[:])
		addressBytes = tmp[:]
	}

	h[0] = binary.BigEndian.Uint64(addressBytes[0:8])
	h[1] = binary.BigEndian.Uint64(addressBytes[8:16])
	h[2] = binary.BigEndian.Uint64(addressBytes[16:24])
}

// GetBytes will fill b byte array with value from h.
func (h *HashType) GetBytes(b []byte) {
	if len(b) < 24 {
		return // err ?
	}
	binary.BigEndian.PutUint64(b[0:8], h[0])
	binary.BigEndian.PutUint64(b[8:16], h[1])
	binary.BigEndian.PutUint64(b[16:24], h[2])
}

// HalfHash represents
//
type HalfHash uint64

// GetFractionalBits returns a slice of n bits. Values of n greater than 64 are not implemented.
func (h *HashType) GetFractionalBits(n uint) int {
	if n < 64 {
		a := h.GetUint64()
		return int(a >> (64 - n))
	}
	fmt.Println("FIXME: implement GetFractionalBits for > 64")
	fmt.Println("FIXME: better idEa")
	fmt.Println("FIXME: please")
	return 0
}

var hashstartkey *[]byte

// HashString will hash the string and init the HashType
func (h *HashType) HashString(s string) {
	h.HashBytes([]byte(s))
}

// HashBytes will initialize an existing hash from a string.
// The string will get hashed to provide the bits so we'll wish this was faster.
// It doesn't have to be crypto safe but it does need to be evenly distributed.
// allocates
func (h *HashType) HashBytes(s []byte) {
	if 0 == 2 {
		md5er := md5.New()
		io.WriteString(md5er, string(s))
		bytes := md5er.Sum(nil)
		h.InitFromBytes(bytes)
		//copy((*h)[:], bytes[0:16])
		//h.a = binary.BigEndian.Uint64(bytes)
		//h.b = binary.BigEndian.Uint64(bytes[8:])
		//fmt.Println(h.a, h.b)
	} else if "64" == "enough bits" {
		if hashstartkey == nil {
			tmp, err := hex.DecodeString("00E5060708090A0BC0B0A00C0D0E0FF90807060504030201000D000102030400")
			if err != nil {
				fmt.Println("FIXME: moron")
			}
			hashstartkey = &tmp
		}
		hhash, _ := highwayhash.New128(*hashstartkey)
		n, err := hhash.Write(s)
		_ = n
		_ = err
		bytes := hhash.Sum(nil)
		h.InitFromBytes(bytes)

	} else {
		sh := sha256.New()
		sh.Write(s)
		shabytes := sh.Sum(nil)
		h.InitFromBytes(shabytes)
	}
}

// FromHashType init an existing hash from another - basically a copy
func (h *HashType) FromHashType(src *HashType) {
	(*h) = *src
}

// Random HashType initializes with random bits.
// We don't need to hash these more do we?
func (h *HashType) Random() {

	var bytes [24]byte
	rand.Read(bytes[:])
	h.InitFromBytes(bytes[:])
}

func (h *HashType) String() string {
	//return hex.EncodeToString(h.bytes[0:16])
	var bytes [24]byte
	h.GetBytes(bytes[:])
	return base64.RawStdEncoding.EncodeToString(bytes[:])
}

func (a *HalfHash) String() string {
	return strconv.FormatUint(uint64(*a), 16)
}
