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

package packets

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"sync"
	"unicode/utf8"

	"io"

	"github.com/awootton/knotfreeiot/badjson"

	"github.com/emirpasic/gods/trees/redblacktree"
)

/** There is a struct (Universal) that can represent *any* packet.
There are individual types for each of the packets.
We read the Universal and then construct the Packet from that and visa versa when writing.
It may seem like we're duplicating data and making a mess but the structs
are full of slices backed by the same data. Readability counts.
*/

// Interface is virtual functions for all packets.
// Basically odd versions of marshal and unmarshal.
type Interface interface {
	//
	Write(writer io.Writer) error // write to Universal and then to writer
	Fill(*Universal) error        // init myself from a Universal that was just read
	//
	ToJSON() ([]byte, error) // for debugging and String() etc.
	String() string

	GetOption(key string) ([]byte, bool)
}

// StandardAlias is really a HashType in bytes or [20]byte or [32]byte. enforced elsewhere.
type StandardAlias []byte

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

// MessageCommon is
type MessageCommon struct {
	PacketCommon
	// the fields:
	// address can be empty if sourceAlias is not. None should be null.
	// aka destination address aka channel aka topic.
	Address      []byte
	AddressAlias StandardAlias
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
	Source      []byte
	SourceAlias StandardAlias
}

// Send aka 'publish' aka 'push' sends Payload (and the options) to destination aka Address.
type Send struct {
	MessageCommon // address

	// a return address. Required.
	Source      []byte
	SourceAlias StandardAlias

	Payload []byte
}

/** Here is the protocol.

There is a type rune "P" or "S" or whatever.

Then there is an arg count: unsigned byte,
	so, we're up to two bytes now and we could have 256 args.
Then a compressed list of integers, which are lengths of the args.
	unsigned bytes <= 127 are a length
	else the lower 7 bits are the msb and the next byte is lsb. etc.
Finally, all the bytes of the args

So, in chars, the command "P topic msg" becomes:
P 2 5 3 t o p i c m s g
where 2 is the number of args 5 is the len of "topic" and 3 is the len of "msg"
followed by the bytes "topicmsg"
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
	var p Interface
	switch uni.Cmd {
	case 'P': // Send aka Publish
		p = new(Send)
		err := p.Fill(uni)
		if err != nil {
			return nil, err
		}
	case 'S': //
		p = &Subscribe{}
		err := p.Fill(uni)
		if err != nil {
			return nil, err
		}
	case 'U': //
		p = &Unsubscribe{}
		err := p.Fill(uni)
		if err != nil {
			return nil, err
		}
	case 'L': //
		p = &Lookup{}
		err := p.Fill(uni)
		if err != nil {
			return nil, err
		}
	case 'C': //
		p = &Connect{}
		err := p.Fill(uni)
		if err != nil {
			return nil, err
		}
	case 'D': //
		p = &Disconnect{}
		err := p.Fill(uni)
		if err != nil {
			return nil, err
		}
	default:
		return nil, errors.New("unknown command " + string(uni.Cmd))
	}
	return p, nil
}

// ReadPacket attempts to obtain a valid Packet from the stream
func ReadPacket(reader io.Reader) (Interface, error) {

	uni, err := ReadUniversal(reader)
	if err != nil {
		return nil, err
	}
	return FillPacket(uni)
	// var p Interface
	// switch uni.Cmd {
	// case 'P': // Send aka Publish
	// 	p = &Send{}
	// 	err := p.Fill(uni)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// case 'S': //
	// 	p = &Subscribe{}
	// 	err := p.Fill(uni)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// case 'U': //
	// 	p = &Unsubscribe{}
	// 	err := p.Fill(uni)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// case 'L': //
	// 	p = &Lookup{}
	// 	err := p.Fill(uni)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// case 'C': //
	// 	p = &Connect{}
	// 	err := p.Fill(uni)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// case 'D': //
	// 	p = &Disconnect{}
	// 	err := p.Fill(uni)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// }

	// return p, nil
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

	if len(str.Args) < 2 {
		return errors.New("too few args for Subscribe")
	}
	p.Address = str.Args[0]
	p.AddressAlias = str.Args[1]

	p.unpackOptions(str.Args[2:])
	return nil
}

// Fill implements the 2nd part of an unmarshal.
func (p *Unsubscribe) Fill(str *Universal) error {

	if len(str.Args) < 2 {
		return errors.New("too few args for Unsubscribe")
	}
	p.Address = str.Args[0]
	p.AddressAlias = str.Args[1]

	p.unpackOptions(str.Args[2:])
	return nil
}

// Fill implements the 2nd part of an unmarshal
func (p *Send) Fill(str *Universal) error {

	if len(str.Args) < 5 {
		return errors.New("too few args for Send")
	}

	p.Address = str.Args[0]
	p.AddressAlias = str.Args[1]

	p.Source = str.Args[2]
	p.SourceAlias = str.Args[3]

	p.Payload = str.Args[4]

	p.unpackOptions(str.Args[5:])
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
func (p *Lookup) Fill(str *Universal) error {

	if len(str.Args) < 4 {
		return errors.New("too few args for Lookup")
	}

	p.Address = str.Args[0]
	p.AddressAlias = str.Args[1]

	p.Source = str.Args[2]
	p.SourceAlias = str.Args[3]

	p.unpackOptions(str.Args[4:])
	return nil
}

// UniversalToJSON outputs an array of strings.
// in a bad json like syntax. It's just for debugging.
// It should be parseable by badjson.
func UniversalToJSON(str *Universal) ([]byte, error) {

	var bb bytes.Buffer

	bb.WriteByte('[')
	bb.WriteRune(rune(str.Cmd))

	for i, bstr := range str.Args {
		_ = i

		isascii, hasdelimeters := badjson.IsASCII(bstr)
		if isascii {
			bb.WriteByte(',')
			if hasdelimeters {
				bb.WriteByte('"')
				bb.WriteString(badjson.MakeEscaped(string(bstr), 0))
				bb.WriteByte('"')
			} else {
				bb.Write(bstr)
			}
		} else if utf8.Valid(bstr) {
			bb.WriteByte(',')

			bb.WriteByte('"')

			bb.WriteString(badjson.MakeEscaped(string(bstr), 0))
			bb.WriteByte('"')

		} else {
			bb.WriteByte(',')
			bb.WriteByte('=')
			tmp := base64.RawStdEncoding.EncodeToString(bstr)
			bb.WriteString(tmp)
		}
	}

	bb.WriteByte(']')

	return bb.Bytes(), nil
}

// these are all the same:

// ToJSON to output a bad json version
func (p *Send) ToJSON() ([]byte, error) {
	p.Write(nil) // force existance of backingUniversal
	bytes, err := UniversalToJSON(p.backingUniversal)
	return bytes, err
}

// ToJSON is not that efficient
func (p *Subscribe) ToJSON() ([]byte, error) {
	p.Write(nil) // force existance of backingUniversal
	bytes, err := UniversalToJSON(p.backingUniversal)
	return bytes, err
}

// ToJSON is something that wastes memory.
func (p *Unsubscribe) ToJSON() ([]byte, error) {
	p.Write(nil) // force existance of backingUniversal
	bytes, err := UniversalToJSON(p.backingUniversal)
	return bytes, err
}

// ToJSON is
func (p *Connect) ToJSON() ([]byte, error) {
	p.Write(nil) // force existance of backingUniversal
	bytes, err := UniversalToJSON(p.backingUniversal)
	return bytes, err
}

// ToJSON is all the same
func (p *Disconnect) ToJSON() ([]byte, error) {
	p.Write(nil) // force existance of backingUniversal
	bytes, err := UniversalToJSON(p.backingUniversal)
	return bytes, err
}

// ToJSON is
func (p *Lookup) ToJSON() ([]byte, error) {
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
		str.Args = make([][]byte, 0, 2+(p.OptionSize()*2))
		str.Args = append(str.Args, p.Address)
		str.Args = append(str.Args, p.AddressAlias)
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
		str.Args = make([][]byte, 0, 2+(p.OptionSize()*2))
		str.Args = append(str.Args, p.Address)
		str.Args = append(str.Args, p.AddressAlias)
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
		str.Args = make([][]byte, 0, 5+(p.OptionSize()*2))
		str.Args = append(str.Args, p.Address)
		str.Args = append(str.Args, p.AddressAlias)
		str.Args = append(str.Args, p.Source)
		str.Args = append(str.Args, p.SourceAlias)
		str.Args = append(str.Args, p.Payload)
		str.Args = p.packOptions(str.Args)
	}
	err := p.backingUniversal.Write(writer)
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

func (p *Lookup) Write(writer io.Writer) error {
	if p.backingUniversal == nil {
		str := new(Universal)
		p.backingUniversal = str
		str.Cmd = 'L'
		str.Args = make([][]byte, 0, 4+(p.OptionSize()*2))
		str.Args = append(str.Args, p.Address)
		str.Args = append(str.Args, p.AddressAlias)
		str.Args = append(str.Args, p.Source)
		str.Args = append(str.Args, p.SourceAlias)
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
	if total > 1024*16 {
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
	return err
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

// WriteVarLenInt writes a variable length integer.
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
// Not meant for integers >= 2^28
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
