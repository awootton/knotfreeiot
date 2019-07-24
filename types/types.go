// Copyright 2019 Alan Tracey Wootton

package types

import (
	"crypto/md5"
	"crypto/rsa"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"strconv"
	"time"

	"github.com/minio/highwayhash"
)

// ProtocolHandlerIntf does read and write of the various messages involved
// with an over the wire iot pub/sub protocol.
// The 'aa' protocol is an example. See timing.TestProtocolHandler as another example.
// Push and HandleWrite are the same. SAME. Server uses HandleWrite and Client uses Push
// Pop and Serve are the same. They block. Server uses Serve and Client uses Pop. COnfused?
// FIXME: redesign atw
// Why do we need this at all-
// Implementations have two channels of interface{} so can't we just expose the wires?
type ProtocolHandlerIntf interface {
	Serve() error

	// HandleWrite needs renaming FIXME.
	HandleWrite(*IncomingMessage) error

	// Push will Q the command and should return immediately. Used by clients
	Push(cmd interface{}) error
	// Pop will block for a timeout that could be as long as 30 minutes.
	// used by clients
	Pop(timeout time.Duration) (interface{}, error)
}

// ConnectionIntf stuff that deals with managing net connections
type ConnectionIntf interface {
	Close()

	GetTCPConn() *net.TCPConn
	SetTCPConn(t *net.TCPConn)

	GetKey() *HashType

	SetRealTopicName(*HashType, string)
	GetRealTopicName(*HashType) (string, bool)

	SetProtocolHandler(protocolHandler ProtocolHandlerIntf)
	GetProtocolHandler() ProtocolHandlerIntf
}

// SubscriptionsIntf stuff that deals with pub/sub
type SubscriptionsIntf interface {
	SendSubscriptionMessage(Topic *HashType, realName string, c ConnectionIntf)
	SendUnsubscribeMessage(Topic *HashType, c ConnectionIntf)
	SendPublishMessage(Topic *HashType, c ConnectionIntf, payload *[]byte)
	GetAllSubsCount() uint64
}

// HashType is 128 bits. We'll use these as keys everywhere
// should we use two longs?
// it's supposed to be immutable.
type HashType struct { // [16]byte
	a, b uint64 // think of this as a bigendian fraction from 0 to 1-1/2^128/. Like a probability. No negatives.
}

// func (k Key) hashCode() string { how?
// 	return fmt.Sprintf("%s/%s", k.Path, k.City) //omit Country in your hash code here
//  }

// IncomingMessage - for ConnectionMgr
// todo: rename or something
type IncomingMessage struct {
	Topic   *HashType
	Message *[]byte
}

// GetFractionalBits returns the required amount of bits. We could really take
// them from anywhere in a hash but we'll take n bits from the top.
func (h *HashType) GetFractionalBits(n uint) int {
	if n < 64 {
		return int(h.a >> (64 - n))
	}
	fmt.Println("FIXME: implmentHashType for > 64")
	return 0

}

var hashstartkey *[]byte //= hex.DecodeString("000102030405060708090A0B0C0D0E0FF0E0D0C0B0A090807060504030201000")

// FromString init an existing hash from a string
// todo: faster, copy less.
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

// FromHashType init an existing hash from another
func (h *HashType) FromHashType(src *HashType) {
	h.a = src.a
	h.a = src.b
}

func (h *HashType) String() string {
	return strconv.FormatUint(h.a, 16)
}

type tinyfloat float32 // actually 12 bits
type twentyFourBits uint32
type four uint32

// Contract instead of password.
// the ProducerKey references a public key
// that will decode
type Contract struct {
	ProducerKey uint32

	SerialNumber         uint32
	ExpirationDate       uint32
	SubscriptionMax      float32
	SendBPS              float32
	ReceiveBPS           float32
	SendBytesPer10min    float32
	ReceiveBytesPer10min float32
	hash                 uint64
}

// EncodeContract returns a big string with a bunch of base64 in it
func EncodeContract(con *Contract, priv *rsa.PrivateKey) string {
	return "ss"
}

// DecodeContract unpacks the string and checks that everything matches.
func DecodeContract(str string, priv *rsa.PublicKey) (Contract, error) {
	c := Contract{}
	return c, nil
}
