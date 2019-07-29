// Copyright 2019 Alan Tracey Wootton

package iot2

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

// HashType is 128 bits. We'll use these as keys. While they are a little fat they're quite resistant
// to collision privided that they are random.
// Think of this as a bigendian fraction from 0 to 1-1/(2^128) . Like a probability. No negatives.
// When we distribute these into buckets we'll start with the high bits in 'a'.
type HashType struct {
	a, b uint64
}

// HalfHash is half of HashType. At 64 bits we might expect some collisions with a billion items
// but in other cases, like a dozen items, it will do.
type HalfHash uint64

// GetFractionalBits returns a slice of n bits. Values of n greater than 64 are not implemented.
func (h *HashType) GetFractionalBits(n uint) int {
	if n < 64 {
		return int(h.a >> (64 - n))
	}
	fmt.Println("FIXME: implmentHashType for > 64")
	return 0

}

var hashstartkey *[]byte //= hex.DecodeString("000102030405060708090A0B0C0D0E0FF0E0D0C0B0A090807060504030201000")

// FromString will initialize an existing hash from a string.
// The string will get hashed to provide the bits so we'll wish this was faster.
// It doesn't have to be crypto safe but it does need to be evenly distrubuted.
func (h *HashType) FromString(s string) {
	if 0 == 1-1 {
		md5er := md5.New()
		io.WriteString(md5er, s)
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
		hhash, _ := highwayhash.New128(*hashstartkey) // (hash.Hash, error)
		io.WriteString(hhash, s)
		bytes := hhash.Sum(nil)
		h.a = binary.BigEndian.Uint64(bytes)
		h.b = binary.BigEndian.Uint64(bytes[8:])
		//fmt.Println(h.a, h.b)
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
	return strconv.FormatUint(h.a, 16)
}
