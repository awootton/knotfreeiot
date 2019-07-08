package types

import (
	"crypto/md5"
	"io"
)

// XXHash128 comment
type XXHash128 struct {
	A int64 `json:"a,omitempty"`
	B int64 `json:"b,omitempty"`
}

// HashType is 128 bits. We'll use these as keys everywhere
type HashType [16]byte

// // Connection is a struct in ...
// type Connection interface {

// }

// ConnectionMgr s
type ConnectionMgr interface {
	//FindConnection(key *HashType) (*Connection, bool)
	Qmessage(channelID *HashType, message *IncomingMessage) bool
}

// IncomingMessage - for ConnectionMgr
type IncomingMessage struct {
	//Type    byte   `json:"@,omitempty"`
	Message []byte `json:"m,omitempty"`
}

// FromString init an existing hash from a string
// todo: faster, copy less.
func (h *HashType) FromString(s string) {
	md5er := md5.New()
	io.WriteString(md5er, s)
	bytes := md5er.Sum(nil)
	for i := 0; i < 16; i++ {
		h[i] = bytes[i]
	}
}

// FromHashType init an existing hash from another
// todo: faster
func (h *HashType) FromHashType(src *HashType) {
	for i := 0; i < 16; i++ {
		h[i] = src[i]
	}
}

// Connection comment
// type xxxConnection struct {
// 	Val *Hash128 `json:"val,omitempty"`
// }

// Contract comment
type Contract struct {
	ProducerKey          uint32 `protobuf:"varint,1,opt,name=producerKey,proto3" json:"producerKey,omitempty"`
	ExpirationDate       uint32 `protobuf:"varint,2,opt,name=expirationDate,proto3" json:"expirationDate,omitempty"`
	SubscriptionMax      uint64 `protobuf:"varint,3,opt,name=subscriptionMax,proto3" json:"subscriptionMax,omitempty"`
	SendBPS              uint32 `protobuf:"varint,4,opt,name=sendBPS,proto3" json:"sendBPS,omitempty"`
	ReceiveBPS           uint32 `protobuf:"varint,5,opt,name=receiveBPS,proto3" json:"receiveBPS,omitempty"`
	SendBytesPerSixth    uint32 `protobuf:"varint,6,opt,name=sendBytesPerSixth,proto3" json:"sendBytesPerSixth,omitempty"`
	ReceiveBytesPerSixth uint32 `protobuf:"varint,7,opt,name=receiveBytesPerSixth,proto3" json:"receiveBytesPerSixth,omitempty"`
	SerialNumber         uint32 `protobuf:"varint,8,opt,name=serialNumber,proto3" json:"serialNumber,omitempty"`
}

// Ack comment
type Ack struct {
	Ok           bool   `protobuf:"varint,1,opt,name=ok,proto3" json:"ok,omitempty"`
	Sequence     uint32 `protobuf:"varint,2,opt,name=sequence,proto3" json:"sequence,omitempty"`
	ErrorMessage string `protobuf:"bytes,3,opt,name=errorMessage,proto3" json:"errorMessage,omitempty"`
}

// PresentContractRequest comment
type PresentContractRequest struct {
	Contract []byte `protobuf:"bytes,1,opt,name=contract,proto3" json:"contract,omitempty"`
	Sequence uint32 `protobuf:"varint,2,opt,name=sequence,proto3" json:"sequence,omitempty"`
}

// xxSubscribeRequest comment. can we avoid pointers?
type xxSubscribeRequest struct {
	Channel      HashType `protobuf:"bytes,1,opt,name=channelHash,proto3" json:"channelHash,omitempty"`
	ChannelName  string   `protobuf:"bytes,2,opt,name=channelName,proto3" json:"channelName,omitempty"`
	ConnectionID HashType `protobuf:"bytes,3,opt,name=connection,proto3" json:"connection,omitempty"`
	//Sequence    uint32      `protobuf:"varint,4,opt,name=sequence,proto3" json:"sequence,omitempty"`
}

// Unsubscribe comment
type Unsubscribe struct {
	ChannelHash HashType `protobuf:"bytes,1,opt,name=channelHash,proto3" json:"channelHash,omitempty"`
	Connection  HashType `protobuf:"bytes,2,opt,name=connection,proto3" json:"connection,omitempty"`
	Sequence    uint32   `protobuf:"varint,3,opt,name=sequence,proto3" json:"sequence,omitempty"`
}

// PublishRequest comment
type PublishRequest struct {
	ChannelHash HashType `protobuf:"bytes,1,opt,name=channelHash,proto3" json:"channelHash,omitempty"`
	//	Sequence    uint32   `protobuf:"varint,2,opt,name=sequence,proto3" json:"sequence,omitempty"`
	Message []byte `protobuf:"bytes,3,opt,name=message,proto3" json:"message,omitempty"`
}

// ReceiveXXX comment
type ReceiveXXX struct {
	ChannelHash HashType `protobuf:"bytes,1,opt,name=channelHash,proto3" json:"channelHash,omitempty"`
	//	Sequence    uint32   `protobuf:"varint,2,opt,name=sequence,proto3" json:"sequence,omitempty"`
	Message []byte `protobuf:"bytes,3,opt,name=message,proto3" json:"message,omitempty"`
}
