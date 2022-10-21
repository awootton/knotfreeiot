// See copyright below
package packets

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"unicode/utf8"

	"io"

	"github.com/awootton/knotfreeiot/badjson"

	"github.com/emirpasic/gods/trees/redblacktree"
)

/**
All packets are in the form of an array of strings where the first string is length of 1.
The format is the first string followed by a count followed by the length of each string followed by the
strings themselves. This format is read and written into and from the Universal struct below.
After that the various classes of packets are derived from the Universal. There is also a Typescript version available.

Addresses can be plain text, in which case they will be hashed (SHA-256) before use
or they can be in hex or base64 or 32 binary bytes.

Lengths are variale length integers.
*/

// Interface is virtual functions for all packets.
// Basically versions of marshal and unmarshal.
type Interface interface {
	//
	Write(writer io.Writer) error // write to Universal and then to writer
	Fill(*Universal) error        // init myself from a Universal that was just read
	//
	ToJSON() ([]byte, error) // for debugging and String() etc.
	String() string

	GetOption(key string) ([]byte, bool)

	GetOptionKeys() ([]string, [][]byte)
}

// AddressType is a byte
type AddressType byte

const (
	// BinaryAddress is whan an AddressUnion is 24 bytes of bits
	BinaryAddress = AddressType(0)
	// HexAddress is whan an AddressUnion is 48 bytes of hex bytes
	HexAddress = AddressType('$')
	// Base64Address is whan an AddressUnion is 32 bytes of base64 bytes
	Base64Address = AddressType('=')
	// Utf8Address is whan an AddressUnion is a utf-8 bytes. The default
	Utf8Address = AddressType(' ')
)

// AddressUnion is one byte followed by more bytes.
// is either utf-8 of an address, or it is a coded version of HashTypeLen bytes
// coding:
// space followed by utf8 glyphs
// $ followed by exactly 48 bytes of hex
// = followed by exactly 32 bytes of base64
// \0 followed by exactly 24 bytes of binary
type AddressUnion struct {
	Type  AddressType // AddressType
	Bytes []byte      // ' ' or '$' or '=' or 0
}

// NewAddressUnion is for constructing utf-8 AddressUnion
func NewAddressUnion(str string) AddressUnion {
	a := &AddressUnion{}
	a.Type = Utf8Address
	a.Bytes = []byte(str)
	return *a
}

// FromString will construct an AddressUnion from a string
// we check the first byte for encoding type else assume it's a string
func (address *AddressUnion) FromString(bytes string) {
	address.FromBytes([]byte(bytes))
}

// FromBytes will construct an AddressUnion from a string
// we check the first byte for encoding type else assume it's a string
func (address *AddressUnion) FromBytes(bytes []byte) {
	a := address
	if len(bytes) == 0 {
		a.Type = Utf8Address
		a.Bytes = bytes
		return
	}
	first := AddressType(bytes[0])
	more := bytes[1:]
	if first == BinaryAddress && len(more) == 24 {
		a.Type = BinaryAddress
		a.Bytes = more
		return
	}
	if first == HexAddress && len(more) == 48 {
		a.Type = HexAddress
		a.Bytes = more
		return
	}
	if first == Base64Address && len(more) == 32 {
		a.Type = Base64Address
		a.Bytes = more
		return
	}
	if first == Utf8Address {
		a.Type = Utf8Address
		a.Bytes = more
		return
	}
	a.Type = Utf8Address
	a.Bytes = bytes // and not more
	return
}

// ToString is for display purposes. It will need to convert the binary type to base64
func (address *AddressUnion) String() string {
	if (address.Type == BinaryAddress) && len(address.Bytes) == 24 {
		//dest := make([]byte, 24)
		str := base64.RawURLEncoding.EncodeToString(address.Bytes)
		return "=" + str
	}
	var b strings.Builder
	b.Grow(len(address.Bytes) + 1)
	b.WriteByte(byte(address.Type))
	b.Write(address.Bytes)
	return b.String()
}

// ToBytes simply concates the Type and the Bytes
// The utf-8 types will NOT end up with a space at the start
// is it better to just alloc the bytes?
func (address *AddressUnion) ToBytes() []byte {

	var b bytes.Buffer
	if address.Type == Utf8Address {
		b.Grow(len(address.Bytes))
		//b.WriteByte(byte(address.Type))
		b.Write(address.Bytes)
		return (b.Bytes())
	}
	b.Grow(len(address.Bytes) + 1)
	b.WriteByte(byte(address.Type))
	b.Write(address.Bytes)
	return (b.Bytes())
}

// EnsureAddressIsBinary looks at the type and then
// changes the address as necessary to be BinaryAddress
// if the $ and = types are malformed then it all gets hashed
// as if it were a string in the first place.
// parsers should screen for bad cases
func (address *AddressUnion) EnsureAddressIsBinary() {
	// HashTypeLen is 24
	if len(address.Bytes) == HashTypeLen && address.Type == BinaryAddress {
		return
	}
	switch address.Type {
	case '$':
		if len(address.Bytes) != HashTypeLen*2 {
			// fall through to utf case
			break // return address, errors.New("requires 48 bytes of hex")
		}
		tmp := make([]byte, HashTypeLen)
		n, _ := hex.Decode(tmp, address.Bytes)
		_ = n
		address.Type = BinaryAddress
		address.Bytes = tmp
		return //&result, err
	case '=':
		if len(address.Bytes) != HashTypeLen*8/6 {
			break // return address, errors.New("requires 32 bytes of base64")
		}
		tmp := make([]byte, HashTypeLen)
		base64.RawURLEncoding.Decode(tmp, address.Bytes)
		address.Type = BinaryAddress
		address.Bytes = tmp
		return // &result, err
	default:
	}
	// is utf8. Hash it.
	// FIXME this is the same as in HashType but we can't use hashtype in packets package.
	sh := sha256.New()
	sh.Write(address.Bytes)
	shabytes := sh.Sum(nil)
	address.Type = BinaryAddress
	address.Bytes = shabytes[0:24] // same as in HashType = just keep 192 bits
	return                         // &result, nil

}

// PacketCommon is stuff the packets all have, like options.
type PacketCommon struct {
	// for internal use only:
	backingUniversal  *Universal         // might be nil, a write will fill it.
	optionalKeyValues *redblacktree.Tree // might be nil if no options.
	// There is a Get and a Put and a Size() for options below.
}

// Connect is the first message
// Usually the options have a JWT with permissions.
type Connect struct {
	PacketCommon
}

// Disconnect is the last thing a client will hear.
// May contain a string in options["error"]
// A client can also send this to server.
type Disconnect struct {
	PacketCommon
}

// Ping is a utility. Aka Heartbeat.
type Ping struct {
	PacketCommon
}

// MessageCommon is
type MessageCommon struct {
	PacketCommon
	// aka destination address aka channel aka topic.
	Address AddressUnion
}

// Subscribe is to declare that the Thing has an address.
// Presumably one would Subscribe before a Send.
type Subscribe struct {
	MessageCommon
}

// Unsubscribe might prevent future reception at the indicated destination address.
type Unsubscribe struct {
	MessageCommon
}

// Lookup returns information on the dest to source.
// can be used to verify existance of an endpoint prior to subscribe.
// If the topic metadata has one subscriber and an ipv6 address then this is the same as a dns lookup.
type Lookup struct {
	MessageCommon
	// a return address
	Source AddressUnion
}

// Send aka 'publish' aka 'push' sends Payload (and the options) to destination aka Address.
type Send struct {
	MessageCommon // address

	// a return address. Required.
	Source AddressUnion

	Payload []byte
}

// HashTypeLen must be the same as iot.HashTypeLen
const HashTypeLen int = 24

/** Here is the protocol.

There is a type rune "P" or "S" etc. (see func FillPacket below)

Then there is an arg count: unsigned byte,
	so, we're up to two bytes now and we could have 256 args.
Then a compressed list of integers, which are lengths of the args.
	unsigned bytes <= 127 are a length
	else the lower 7 bits are the msb and the next byte is lsb. etc.
Finally, all the bytes of the args

So, in chars, the command "P topic msg" becomes:
P 2 5 3 t o p i c m s g
where 2 is the number of strings 5 is the len of "topic" and 3 is the len of "msg"
followed by the bytes "topicmsg"

The ints are all variable length unsigned with a max int of 128^4.
There is a max of 255 strings total per packet.

*/

// CommandType is usually ascii
type CommandType rune

// Universal is the wire representation.
// they can all be represented this way.
type Universal struct {
	Cmd  CommandType
	Args [][]byte
}

// GetIPV6Option is an example of using options
func (p *PacketCommon) GetIPV6Option() []byte {
	got, ok := p.GetOption("IPv6")
	if !ok {
		got = []byte("")
	}
	return got
}

// FillPacket construct a packet from supplied Universal
func FillPacket(uni *Universal) (Interface, error) {

	switch uni.Cmd {
	case 'P': // Send aka Publish
		p := new(Send)
		err := p.Fill(uni)
		if err != nil {
			return nil, err
		}
		p.backingUniversal = nil
		return p, nil
	case 'S': //
		p := &Subscribe{}
		err := p.Fill(uni)
		if err != nil {
			return nil, err
		}
		p.backingUniversal = nil
		return p, nil
	case 'U': //
		p := &Unsubscribe{}
		err := p.Fill(uni)
		if err != nil {
			return nil, err
		}
		p.backingUniversal = nil
		return p, nil
	case 'L': //
		p := &Lookup{}
		err := p.Fill(uni)
		if err != nil {
			return nil, err
		}
		p.backingUniversal = nil
		return p, nil
	case 'C': //
		p := &Connect{}
		err := p.Fill(uni)
		if err != nil {
			return nil, err
		}
		p.backingUniversal = nil
		return p, nil
	case 'D': //
		p := &Disconnect{}
		err := p.Fill(uni)
		if err != nil {
			return nil, err
		}
		p.backingUniversal = nil
		return p, nil
	case 'H': // Ping aka Heartbeat, used?
		p := &Ping{}
		err := p.Fill(uni)
		if err != nil {
			return nil, err
		}
		p.backingUniversal = nil
		return p, nil
	default:
		return nil, errors.New("unknown command " + string(uni.Cmd))
	}
}

// ReadPacket attempts to obtain a valid Packet from the stream
func ReadPacket(reader io.Reader) (Interface, error) {

	uni, err := ReadUniversal(reader)
	if err != nil {
		return nil, err
	}
	p, err := FillPacket(uni)
	return p, err
}

// The args slice is key then value in pairs
func (p *PacketCommon) unpackOptions(args [][]byte) {
	key := "none"
	for i, arg := range args {
		if i&1 == 1 { // on odd numbers
			if key != "" {
				p.SetOption(key, arg)
			}
		}
		key = string(arg)
	}
}

func (p *PacketCommon) packOptions(args [][]byte) [][]byte {
	if p.OptionSize() == 0 {
		return args
	}
	it := p.optionalKeyValues.Iterator()
	for it.Next() {
		args = append(args, []byte(it.Key().(string)))
		args = append(args, it.Value().([]byte))
	}
	return args
}

// Fill implements the 2nd part of an unmarshal.
// See ReadPacket
func (p *Subscribe) Fill(str *Universal) error {

	if len(str.Args) < 1 {
		return errors.New("too few args for Subscribe")
	}
	p.Address.FromBytes(str.Args[0])

	p.unpackOptions(str.Args[1:])
	return nil
}

// Fill implements the 2nd part of an unmarshal.
func (p *Unsubscribe) Fill(str *Universal) error {

	if len(str.Args) < 1 {
		return errors.New("too few args for Unsubscribe")
	}

	p.Address.FromBytes(str.Args[0])
	p.unpackOptions(str.Args[1:])
	return nil
}

// Fill implements the 2nd part of an unmarshal
func (p *Send) Fill(str *Universal) error {

	if len(str.Args) < 3 {
		return errors.New("too few args for Send")
	}
	p.Address.FromBytes(str.Args[0])
	p.Source.FromBytes(str.Args[1])
	p.Payload = str.Args[2]
	p.unpackOptions(str.Args[3:])
	return nil
}

// Fill implements the 2nd part of an unmarshal.
func (p *Connect) Fill(str *Universal) error {

	p.unpackOptions(str.Args[0:])
	return nil
}

// Fill implements the 2nd part of an unmarshal.
func (p *Disconnect) Fill(str *Universal) error {

	p.unpackOptions(str.Args[0:])
	return nil
}

// Fill implements the 2nd part of an unmarshal.
func (p *Ping) Fill(str *Universal) error {

	p.unpackOptions(str.Args[0:])
	return nil
}

// Fill implements the 2nd part of an unmarshal.
func (p *Lookup) Fill(str *Universal) error {

	if len(str.Args) < 2 {
		return errors.New("too few args for Lookup")
	}
	p.Address.FromBytes(str.Args[0])
	p.Source.FromBytes(str.Args[1])
	p.unpackOptions(str.Args[2:])
	return nil
}

// UniversalToJSON outputs an array of strings.
// in a bad json like syntax. It's just for debugging.
// It should be parseable by badjson.
// some of the strings start with \0 and must be converted
func UniversalToJSON(str *Universal) ([]byte, error) {

	var bb bytes.Buffer

	bb.WriteByte('[')
	bb.WriteRune(rune(str.Cmd))

	for i, bstr := range str.Args {
		_ = i
		if len(bstr) == 0 {
			bb.WriteByte(',')
		} else if bstr[0] == 0 {
			bstr = bstr[1:]
			bb.WriteByte(',')
			bb.WriteByte('=')
			tmp := base64.RawURLEncoding.EncodeToString(bstr)
			bb.WriteString(tmp)
		} else {
			isascii, hasdelimeters := badjson.IsASCII(bstr)
			if isascii {
				bb.WriteByte(',')
				if hasdelimeters {
					bb.WriteByte('"')
					bb.WriteString(badjson.MakeEscaped(string(bstr)))
					bb.WriteByte('"')
				} else {
					bb.Write(bstr)
				}
			} else if utf8.Valid(bstr) {
				bb.WriteByte(',')

				bb.WriteByte('"')

				bb.WriteString(badjson.MakeEscaped(string(bstr)))
				bb.WriteByte('"')

			} else {
				bb.WriteByte(',')
				bb.WriteByte('=')
				tmp := base64.RawURLEncoding.EncodeToString(bstr)
				bb.WriteString(tmp)
			}
		}
	}

	bb.WriteByte(']')

	return bb.Bytes(), nil
}

// these are all the same:

// ToJSON to output a bad json version
func (p *Send) ToJSON() ([]byte, error) {
	p.backingUniversal = nil
	p.Write(nil) // force existance of backingUniversal
	bytes, err := UniversalToJSON(p.backingUniversal)
	return bytes, err
}

// ToJSON is not that efficient
func (p *Subscribe) ToJSON() ([]byte, error) {
	p.backingUniversal = nil
	p.Write(nil) // force existance of backingUniversal
	bytes, err := UniversalToJSON(p.backingUniversal)
	return bytes, err
}

// ToJSON is something that wastes memory.
func (p *Unsubscribe) ToJSON() ([]byte, error) {
	p.backingUniversal = nil
	p.Write(nil) // force existance of backingUniversal
	bytes, err := UniversalToJSON(p.backingUniversal)
	return bytes, err
}

// ToJSON is
func (p *Connect) ToJSON() ([]byte, error) {
	p.backingUniversal = nil
	p.Write(nil) // force existance of backingUniversal
	bytes, err := UniversalToJSON(p.backingUniversal)
	return bytes, err
}

// ToJSON is all the same
func (p *Disconnect) ToJSON() ([]byte, error) {
	p.backingUniversal = nil
	p.Write(nil) // force existance of backingUniversal
	bytes, err := UniversalToJSON(p.backingUniversal)
	return bytes, err
}

// ToJSON is all the same
func (p *Ping) ToJSON() ([]byte, error) {
	p.backingUniversal = nil
	p.Write(nil) // force existance of backingUniversal
	bytes, err := UniversalToJSON(p.backingUniversal)
	return bytes, err
}

// ToJSON is
func (p *Lookup) ToJSON() ([]byte, error) {
	p.backingUniversal = nil
	p.Write(nil) // force existance of backingUniversal
	bytes, err := UniversalToJSON(p.backingUniversal)
	return bytes, err
}

func (str *Universal) String() string {
	b, _ := UniversalToJSON(str)
	return string(b)
}

// These are all the same:
// for debugging
func (p *Send) String() string {
	b, _ := p.ToJSON()
	return string(b)
}

func (p *Subscribe) String() string {
	b, _ := p.ToJSON()
	return string(b)
}

func (p *Unsubscribe) String() string {
	b, _ := p.ToJSON()
	return string(b)
}

func (p *Connect) String() string {
	b, _ := p.ToJSON()
	return string(b)
}

func (p *Disconnect) String() string {
	b, _ := p.ToJSON()
	return string(b)
}

func (p *Ping) String() string {
	b, _ := p.ToJSON()
	return string(b)
}

func (p *Lookup) String() string {
	b, _ := p.ToJSON()
	return string(b)
}

// Write implements a marshal operation.
func (p *Subscribe) Write(writer io.Writer) error {
	if p.backingUniversal == nil {
		str := new(Universal)
		p.backingUniversal = str
		str.Cmd = 'S' //
		str.Args = make([][]byte, 0, 1+(p.OptionSize()*2))
		str.Args = append(str.Args, p.Address.ToBytes())
		str.Args = p.packOptions(str.Args)
	}
	err := p.backingUniversal.Write(writer)
	return err
}

// Write marshals to binary
func (p *Unsubscribe) Write(writer io.Writer) error {
	if p.backingUniversal == nil {
		str := new(Universal)
		p.backingUniversal = str
		str.Cmd = 'U' //
		str.Args = make([][]byte, 0, 1+(p.OptionSize()*2))
		str.Args = append(str.Args, p.Address.ToBytes())
		str.Args = p.packOptions(str.Args)
	}
	err := p.backingUniversal.Write(writer)
	return err
}

// Write forces backingUniversal
func (p *Send) Write(writer io.Writer) error {
	if p.backingUniversal == nil {
		str := new(Universal)
		p.backingUniversal = str
		str.Cmd = 'P' // Publish
		str.Args = make([][]byte, 0, 3+(p.OptionSize()*2))
		str.Args = append(str.Args, p.Address.ToBytes())
		str.Args = append(str.Args, p.Source.ToBytes())
		str.Args = append(str.Args, p.Payload)
		str.Args = p.packOptions(str.Args)
	}
	err := p.backingUniversal.Write(writer) // can have nulls in the addresses
	return err
}

func (p *Connect) Write(writer io.Writer) error {
	if p.backingUniversal == nil {
		str := new(Universal)
		p.backingUniversal = str
		str.Cmd = 'C'
		str.Args = make([][]byte, 0, 0+(p.OptionSize()*2))
		str.Args = p.packOptions(str.Args)
	}
	err := p.backingUniversal.Write(writer)
	return err
}

func (p *Disconnect) Write(writer io.Writer) error {
	if p.backingUniversal == nil {
		str := new(Universal)
		p.backingUniversal = str
		str.Cmd = 'D'
		str.Args = make([][]byte, 0, 0+(p.OptionSize()*2))
		str.Args = p.packOptions(str.Args)
	}
	err := p.backingUniversal.Write(writer)
	return err
}

func (p *Ping) Write(writer io.Writer) error {
	if p.backingUniversal == nil {
		str := new(Universal)
		p.backingUniversal = str
		str.Cmd = 'H'
		str.Args = make([][]byte, 0, 0+(p.OptionSize()*2))
		str.Args = p.packOptions(str.Args)
	}
	err := p.backingUniversal.Write(writer)
	return err
}

func (p *Lookup) Write(writer io.Writer) error {
	if p.backingUniversal == nil {
		str := new(Universal)
		p.backingUniversal = str
		str.Cmd = 'L'
		str.Args = make([][]byte, 0, 2+(p.OptionSize()*2))
		str.Args = append(str.Args, p.Address.ToBytes())
		str.Args = append(str.Args, p.Source.ToBytes())
		str.Args = p.packOptions(str.Args)
	}
	err := p.backingUniversal.Write(writer)
	return err
}

// ReadUniversal reads a Universal packet.
func ReadUniversal(reader io.Reader) (*Universal, error) {

	str := Universal{}
	oneByte := []uint8{0}
	n, err := reader.Read(oneByte) // read the command type
	if err != nil {
		return &str, err
	}
	str.Cmd = CommandType(oneByte[0])

	// read array of byte arrays
	str.Args, err = ReadArrayOfByteArray(reader)
	_ = n
	return &str, err
}

// This is a pool of [128]int which is something the read uses to temporarily store
// the lengths of the strings. Awkward on an Arduino this will be.
var pool = sync.Pool{
	// New creates an object when the pool has nothing available to return.
	// New must return an interface{} to make it flexible. You have to cast
	// your type after getting it.
	New: func() interface{} {
		// Pools often contain things like *bytes.Buffer, which are
		// temporary and re-usable.
		return new([128]int)
	},
}

// ReadArrayOfByteArray to read an array of byte arrays
func ReadArrayOfByteArray(reader io.Reader) ([][]byte, error) {

	oneByte := []uint8{0}
	// read the lengths of the following args
	n, err := reader.Read(oneByte)
	if err != nil {
		return nil, err
	}
	argsLen := uint8(oneByte[0])
	if argsLen&0x80 != 0 {
		// in the future this would mean that another byte follows and the args
		// count is even bigger but for now ...
		return nil, errors.New("Too many strings")
	}

	lengths := pool.Get().(*[128]int)
	defer pool.Put(lengths)

	total := 0
	for i := uint8(0); i < argsLen; i++ { // read the lengths of the following strings
		aval, err := ReadVarLenInt(reader)
		if err != nil {
			return nil, err
		}
		lengths[i] = aval
		total += aval
	}
	// if total > (65536 - 256) { // was 1024*16 but now 65536 - 256 atw 3/2021
	// 	return nil, errors.New("Packet too long for this reality")
	// }

	if total >= 8000000 { // atw 4/2021
		return nil, errors.New("Packet too long for this reality")
	}

	// now we can read the rest all at once
	bytes := make([]uint8, total) // alloc the base array
	//n, err = reader.Read(bytes)   // read it. timeout?
	n, err = io.ReadFull(reader, bytes)
	if err != nil {
		return nil, err
	}
	if n != total {
		return nil, errors.New(fmt.Sprint("Too few bytes", n, total))
	}
	// now we can slice the args
	position := 0
	args := make([][]byte, argsLen) // array of slices
	for i := 0; i < int(argsLen); i++ {
		args[i] = bytes[position : position+lengths[i]]
		position += lengths[i]
	}
	return args, nil
}

// Write an Universal packet.
func (str *Universal) Write(writer io.Writer) error {

	if writer == nil {
		return nil
	}

	oneByte := []uint8{0}
	oneByte[0] = uint8(str.Cmd)
	n, err := writer.Write(oneByte)
	if err != nil {
		return err
	}
	err = WriteArrayOfByteArray(str.Args, writer)
	_ = n
	return err // can haz binary data preceded by \0
}

// WriteArrayOfByteArray write count then lengths and then bytes
func WriteArrayOfByteArray(args [][]byte, writer io.Writer) error {

	if len(args) >= 128 {
		return errors.New("Too many args")
	}
	oneByte := []uint8{0}
	oneByte[0] = uint8(len(args))
	n, err := writer.Write(oneByte)
	if err != nil {
		return err
	}
	// write the lengths
	for i := 0; i < len(args); i++ {
		err = WriteVarLenInt(uint32(len(args[i])), uint8(0x00), writer)
		if err != nil {
			return err
		}
	}
	// write the bytes
	for i := 0; i < len(args); i++ {
		n, err = writer.Write(args[i])
		if err != nil {
			return err
		}
	}
	_ = n
	return nil
}

// WriteVarLenInt writes a variable length integer. bigendian
// I'm sure there's a name for this but I forget.
// Unsigned integers are written big end first 7 bits at at time.
// The last byte is >=0 and <=127. The other bytes have the high bit set.
// Small values use one byte.
// A version of this without recursion would be better. todo:
func WriteVarLenInt(uintvalue uint32, mask uint8, writer io.Writer) error {
	if uintvalue > 127 {
		// write the lsb first
		err := WriteVarLenInt(uintvalue>>7, 0x80, writer)
		if err != nil {
			return err
		}
	}
	{
		oneByte := []uint8{0}
		oneByte[0] = uint8((uintvalue & 0x7F) | uint32(mask))
		_, err := writer.Write(oneByte)
		return err
	}
}

// ReadVarLenInt see comments above
// Not meant for integers >= 2^28 big endian
func ReadVarLenInt(reader io.Reader) (int, error) {
	oneByte := []uint8{0}
	_, err := reader.Read(oneByte)
	if err != nil {
		return 0, err
	}
	aval := 0
	remaining := 4
	for remaining != 0 {
		aval <<= 7
		if oneByte[0] >= 128 {
			aval |= int(oneByte[0]) & 0x7F
			remaining--
			_, err := reader.Read(oneByte)
			if err != nil {
				return 0, err
			}
		} else { // the common case
			aval |= int(oneByte[0])
			remaining = 0
			break
		}
	}
	return aval, nil
}

// OptionSize returns key count which is same as value count
func (p *PacketCommon) OptionSize() int {
	if p.optionalKeyValues == nil {
		return 0
	}
	return p.optionalKeyValues.Size()
}

// GetOption returns the value,true to go with the key or nil,false
func (p *PacketCommon) GetOption(key string) ([]byte, bool) {
	if p.optionalKeyValues == nil {
		return nil, false
	}
	val, ok := p.optionalKeyValues.Get(key)
	if !ok {
		val = []byte("")
	}
	return val.([]byte), ok
}

// GetOptionKeys returns a slice of strings
func (p *PacketCommon) GetOptionKeys() ([]string, [][]byte) {
	keys := make([]string, 0)
	values := make([][]byte, 0)
	if p.optionalKeyValues == nil {
		return keys, values
	}
	it := p.optionalKeyValues.Iterator()
	for it.Next() {
		keys = append(keys, it.Key().(string))
		values = append(values, it.Value().([]byte))
	}
	return keys, values
}

// DeleteOption returns the value,true to go with the key or nil,false
func (p *PacketCommon) DeleteOption(key string) {
	if p.optionalKeyValues == nil {
		return
	}
	p.optionalKeyValues.Remove(key)

}

// SetOption adds the key,value
func (p *PacketCommon) SetOption(key string, val []byte) {
	if p.optionalKeyValues == nil {
		p.optionalKeyValues = redblacktree.NewWithStringComparator()
	}
	p.optionalKeyValues.Put(key, val)
}

// Copyright 2019,2020,2021,2022 Alan Tracey Wootton
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
