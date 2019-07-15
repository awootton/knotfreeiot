package types

import (
	"crypto/md5"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"strconv"
)

// ProtocolHandler does read and write of the various messages involved
// with an over the wire iot pub/sub protocol.
// The 'aa' protocol is an example.
type ProtocolHandler interface {
	Serve() error
	// HandleWrite needs renaming FIXME.
	HandleWrite(*IncomingMessage) error

	// Push will Q the command and should return immediately. Used by clients
	Push(cmd interface{}) error
	// Pop will block for a timeout that could be as long as 30 minutes.
	// used by clients
	Pop() (interface{}, error)
}

// ConnectionIntf stuff that deals with managing net connections
type ConnectionIntf interface {
	GetTCPConn() *net.TCPConn
	Close()

	SetRealTopicName(*HashType, string)
	GetRealTopicName(*HashType) (string, bool)
	GetKey() *HashType
}

// SubscriptionsIntf stuff that deals with pub/sub
type SubscriptionsIntf interface {
	SendSubscriptionMessage(Topic *HashType, ConnectionID *HashType)
	SendUnsubscribeMessage(Topic *HashType, ConnectionID *HashType)
	SendPublishMessage(Topic *HashType, ConnectionID *HashType, payload *[]byte)
}

// HashType is 128 bits. We'll use these as keys everywhere
// should we use two longs?
type HashType struct { // [16]byte
	a, b uint64 // think of this as a bigendian fraction from 0 to 1-1/2^128/. Like a probability. No negatives.
}

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

// FromString init an existing hash from a string
// todo: faster, copy less.
func (h *HashType) FromString(s string) {
	md5er := md5.New()
	io.WriteString(md5er, s)
	bytes := md5er.Sum(nil)
	h.a = binary.BigEndian.Uint64(bytes)
	h.b = binary.BigEndian.Uint64(bytes[8:])
	//fmt.Println(h.a, h.b)
}

// FromHashType init an existing hash from another
func (h *HashType) FromHashType(src *HashType) {
	h.a = src.a
	h.a = src.b
}

func (h *HashType) String() string {
	return strconv.FormatUint(h.a, 16)
}

// Connection comment
// type xxxConnection struct {
// 	Val *Hash128 `json:"val,omitempty"`
// }

// xxxContract comment
type xxxContract struct {
	ProducerKey          uint32 `protobuf:"varint,1,opt,name=producerKey,proto3" json:"producerKey,omitempty"`
	ExpirationDate       uint32 `protobuf:"varint,2,opt,name=expirationDate,proto3" json:"expirationDate,omitempty"`
	SubscriptionMax      uint64 `protobuf:"varint,3,opt,name=subscriptionMax,proto3" json:"subscriptionMax,omitempty"`
	SendBPS              uint32 `protobuf:"varint,4,opt,name=sendBPS,proto3" json:"sendBPS,omitempty"`
	ReceiveBPS           uint32 `protobuf:"varint,5,opt,name=receiveBPS,proto3" json:"receiveBPS,omitempty"`
	SendBytesPerSixth    uint32 `protobuf:"varint,6,opt,name=sendBytesPerSixth,proto3" json:"sendBytesPerSixth,omitempty"`
	ReceiveBytesPerSixth uint32 `protobuf:"varint,7,opt,name=receiveBytesPerSixth,proto3" json:"receiveBytesPerSixth,omitempty"`
	SerialNumber         uint32 `protobuf:"varint,8,opt,name=serialNumber,proto3" json:"serialNumber,omitempty"`
}

// xxxAck comment
type xxxAck struct {
	Ok           bool   `protobuf:"varint,1,opt,name=ok,proto3" json:"ok,omitempty"`
	Sequence     uint32 `protobuf:"varint,2,opt,name=sequence,proto3" json:"sequence,omitempty"`
	ErrorMessage string `protobuf:"bytes,3,opt,name=errorMessage,proto3" json:"errorMessage,omitempty"`
}

// xxxPresentContractRequest comment
type xxxPresentContractRequest struct {
	Contract []byte `protobuf:"bytes,1,opt,name=contract,proto3" json:"contract,omitempty"`
	Sequence uint32 `protobuf:"varint,2,opt,name=sequence,proto3" json:"sequence,omitempty"`
}

// xxSubscribeRequest comment. can we avoid pointers?
type xxSubscribeRequest struct {
	Topic        HashType `protobuf:"bytes,1,opt,name=channelHash,proto3" json:"channelHash,omitempty"`
	TopicName    string   `protobuf:"bytes,2,opt,name=channelName,proto3" json:"channelName,omitempty"`
	ConnectionID HashType `protobuf:"bytes,3,opt,name=connection,proto3" json:"connection,omitempty"`
	//Sequence    uint32      `protobuf:"varint,4,opt,name=sequence,proto3" json:"sequence,omitempty"`
}

// xxUnsubscribe comment
type xxUnsubscribe struct {
	TopicHash  HashType `protobuf:"bytes,1,opt,name=channelHash,proto3" json:"channelHash,omitempty"`
	Connection HashType `protobuf:"bytes,2,opt,name=connection,proto3" json:"connection,omitempty"`
	Sequence   uint32   `protobuf:"varint,3,opt,name=sequence,proto3" json:"sequence,omitempty"`
}

// xxPublishRequest comment
type xxPublishRequest struct {
	TopicHash HashType `protobuf:"bytes,1,opt,name=channelHash,proto3" json:"channelHash,omitempty"`
	//	Sequence    uint32   `protobuf:"varint,2,opt,name=sequence,proto3" json:"sequence,omitempty"`
	Message []byte `protobuf:"bytes,3,opt,name=message,proto3" json:"message,omitempty"`
}

// ReceiveXXX comment
type ReceiveXXX struct {
	TopicHash HashType `protobuf:"bytes,1,opt,name=channelHash,proto3" json:"channelHash,omitempty"`
	//	Sequence    uint32   `protobuf:"varint,2,opt,name=sequence,proto3" json:"sequence,omitempty"`
	Message []byte `protobuf:"bytes,3,opt,name=message,proto3" json:"message,omitempty"`
}

// XXHash128 comment
type XXHash128 struct {
	A int64 `json:"a,omitempty"`
	B int64 `json:"b,omitempty"`
}
