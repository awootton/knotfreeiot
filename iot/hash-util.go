// Copyright 2019 Alan Tracey Wootton
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
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"math/rand"
	"strconv"

	"github.com/minio/highwayhash"
)

// HashType represents 128 bits of randomness. We'll use these as keys and ID's. While they are a little fat they're quite resistant
// to collision provided that they are random.
// Think of this as a fraction from 0 to 1-1/(2^128) . Like a probability. Unsigned.
// When we distribute these into buckets we'll start with the high bits in 'a'.
type HashType struct {
	a, b uint64
}

// GetA just for debug
func (h *HashType) GetA() uint64 {
	return h.a
}

// HalfHash represents 64 bits of randomness. HalfHash is half of HashType. At 64 bits we might expect some collisions with a billion items
// but in other cases, like a dozen items, it will do.
type HalfHash uint64

// GetFractionalBits returns a slice of n bits. Values of n greater than 64 are not implemented.
func (h *HashType) GetFractionalBits(n uint) int {
	if n < 64 {
		return int(h.a >> (64 - n))
	}
	fmt.Println("FIXME: implement GetFractionalBits for > 64")
	return 0

}

var hashstartkey *[]byte

// FromString will hash the string and init the HashType
func (h *HashType) FromString(s string) {
	h.FromBytes([]byte(s))
}

// FromBytes will initialize an existing hash from a string .
// The string will get hashed to provide the bits so we'll wish this was faster.
// It doesn't have to be crypto safe but it does need to be evenly distributed.
func (h *HashType) FromBytes(s []byte) {
	if 0 == 2 {
		md5er := md5.New()
		io.WriteString(md5er, string(s))
		bytes := md5er.Sum(nil)
		h.a = binary.BigEndian.Uint64(bytes)
		h.b = binary.BigEndian.Uint64(bytes[8:])
		//fmt.Println(h.a, h.b)
	} else {
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
		h.a = binary.BigEndian.Uint64(bytes)
		h.b = binary.BigEndian.Uint64(bytes[8:])
		//fmt.Println("HashType", h.a, h.b)
	}
}

// FromHashType init an existing hash from another - basically a copy
func (h *HashType) FromHashType(src *HashType) {
	h.a = src.a
	h.b = src.b
}

// Random HashType initializes with random bits.
// We don't need to hash these more do we?
func (h *HashType) Random() {

	h.a = rand.Uint64()
	h.b = rand.Uint64()
	// randomStr := strconv.FormatInt(rand.Int63(), 16) + strconv.FormatInt(rand.Int63(), 16)
	// md5er := md5.New()
	// io.WriteString(md5er, randomStr)
	// bytes := md5er.Sum(nil)
	// h.a = binary.BigEndian.Uint64(bytes)
	// h.b = binary.BigEndian.Uint64(bytes[8:])
}

func (h *HashType) String() string {
	return strconv.FormatUint(h.a, 16) + strconv.FormatUint(h.b, 16)
}

func (a *HalfHash) String() string {
	return strconv.FormatUint(uint64(*a), 16)
}
