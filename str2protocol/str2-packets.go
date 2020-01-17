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

package str2protocol

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net"
	"unicode/utf8"
)

/** There is a struct (Str2) that can represent any packet.
There are individual types for each of the packets.
We read the Str2 and then construct the Packet from that and visa versa when writing.
It may seem like we're duplicating data and making a mess but the structs
are full of slices backed by the same data. Readability counts.
*/

// PacketIntf is virtual functions for all packets.
// Basically versions of marshal and unmarshal.
type PacketIntf interface {
	Write(conn net.Conn) error // write to Str2 and then to conn
	Fill(*Str2) error          // init myself from a Str2 that was just read
	ToJSON() ([]byte, error)   // for debugging and String() etc.
	String() string
}

// PacketCommon is stuff the packets all have, like options.
type PacketCommon struct {

	// for internal use only:
	backingstr2 *Str2             // might be nil, a write will fill it.
	options     map[string][]byte // OptionsMap // optional
}

// StandardAlias is really a HashType in bytes or [16]byte
type StandardAlias []byte

// MessageCommon is
type MessageCommon struct {
	PacketCommon
	// the fields:
	// address can be empty if sourceAlias is not. None should be null.
	address      []byte
	addressAlias StandardAlias
}

// Send aka 'publish' sends data to destination possibly expecting a reply to source.
//
type Send struct {
	MessageCommon // address

	// a return address
	source      []byte
	sourceAlias StandardAlias

	payload []byte
}

// Lookup returns information on the dest to source.
// can be used to verify existance of an endpoint prior to subscribe.
// If the topic metadata has one subscriber and an ipv6 address then this is the same as a dns lookup.
type Lookup struct {
	MessageCommon
	// a return address
	source      []byte
	sourceAlias StandardAlias
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

/** Here is the protocol.

There is a type byte "P" or "S" or whatever.

Then there is an arg count: unsigned byte,
	so, we're up to two bytes now.
Then a compressed list of integers, which are lengths of the following  byte arrays.
	unsigned bytes <= 127 are a length
	else the lower 7 bits are the msb and the next byte is lsb. etc.
Finally, all the bytes of the args

So, in chars, the command "P topic msg" becomes:
P 2 5 3 t o p i c m s g
where 2 is the number of args 5 is the len of "topic" and 3 is the len of "msg"
followed by the bytes "topicmsg"

*/

// CommandType is ascii
type CommandType uint8

// Bstr is like a string but is bytes. I just don't like to declare [][]byte below.
// it could be utf8. many times it's 16 bytes for 128 bits
// type Bstr []byte

// Str2 is the internal representation.
type Str2 struct {
	cmd  CommandType
	args [][]byte
}

// GetIPV6Option is an example of using options
func (p *PacketCommon) GetIPV6Option() []byte {
	return p.options["ipv6"]
}

// ReadPacket attempts to obtain a valid Packet from the stream
func ReadPacket(reader io.Reader) (PacketIntf, error) {

	str, err := ReadStr2(reader)
	if err != nil {
		return nil, err
	}
	var p PacketIntf
	switch str.cmd {
	case 'P': // Send aka Publish
		p = &Send{}
		err := p.Fill(str)
		if err != nil {
			return nil, err
		}
	case 'S': //
		p = &Subscribe{}
		err := p.Fill(str)
		if err != nil {
			return nil, err
		}
	case 'U': //
		p = &Unsubscribe{}
		err := p.Fill(str)
		if err != nil {
			return nil, err
		}
	case 'L': //
		p = &Lookup{}
		err := p.Fill(str)
		if err != nil {
			return nil, err
		}
	case 'C': //
		p = &Connect{}
		err := p.Fill(str)
		if err != nil {
			return nil, err
		}
	case 'D': //
		p = &Disconnect{}
		err := p.Fill(str)
		if err != nil {
			return nil, err
		}
	}

	return p, nil
}

// The args slice is key then value in pairs
func unpackOptions(args [][]byte, optionsP *map[string][]byte) {
	if *optionsP == nil {
		*optionsP = make(map[string][]byte, len(args)/2)
	}
	key := "none"
	for i, arg := range args {
		if i&1 == 1 { // on odd numbers
			(*optionsP)[key] = arg
		}
		key = string(arg)
	}
}

// Fill implements the 2nd part of an unmarshal.
// See ReadPacket
func (p *Subscribe) Fill(str *Str2) error {

	if len(str.args) < 2 {
		return errors.New("too few args for Subscribe")
	}
	p.address = str.args[0]
	p.addressAlias = str.args[1]

	unpackOptions(str.args[2:], &p.options)
	return nil
}

// Fill implements the 2nd part of an unmarshal.
func (p *Unsubscribe) Fill(str *Str2) error {

	if len(str.args) < 2 {
		return errors.New("too few args for Unsubscribe")
	}
	p.address = str.args[0]
	p.addressAlias = str.args[1]

	unpackOptions(str.args[2:], &p.options)
	return nil
}

// Fill implements the 2nd part of an unmarshal
func (p *Send) Fill(str *Str2) error {

	if len(str.args) < 5 {
		return errors.New("too few args for Send")
	}

	p.address = str.args[0]
	p.addressAlias = str.args[1]

	p.source = str.args[2]
	p.sourceAlias = str.args[3]

	p.payload = str.args[4]

	unpackOptions(str.args[5:], &p.options)
	return nil
}

// Fill implements the 2nd part of an unmarshal.
func (p *Connect) Fill(str *Str2) error {

	unpackOptions(str.args[0:], &p.options)
	return nil
}

// Fill implements the 2nd part of an unmarshal.
func (p *Disconnect) Fill(str *Str2) error {

	unpackOptions(str.args[0:], &p.options)
	return nil
}

// Fill implements the 2nd part of an unmarshal.
func (p *Lookup) Fill(str *Str2) error {

	if len(str.args) < 4 {
		return errors.New("too few args for Lookup")
	}

	p.address = str.args[0]
	p.addressAlias = str.args[1]

	p.source = str.args[2]
	p.sourceAlias = str.args[3]

	unpackOptions(str.args[4:], &p.options)
	return nil
}

func packOptions(args [][]byte, options *map[string][]byte) [][]byte {
	for k, v := range *options {
		args = append(args, []byte(k))
		args = append(args, v)
	}
	return args
}

func str2ToJSON(str *Str2) ([]byte, error) {

	amap := make(map[string]interface{})

	amap["cmd"] = string(str.cmd)

	argArr := make([]map[string]interface{}, 0, len(str.args))
	for i, bstr := range str.args {
		isascii := true
		val := make(map[string]interface{})
		for _, b := range bstr {
			if b < 32 || b > 127 {
				isascii = false
				break
			}
		}
		if isascii {
			val["ascii"] = string(bstr) // ascii
		} else {
			if utf8.Valid(bstr) {
				val["utf8"] = string(bstr)
			} else {
				val["b64"] = base64.StdEncoding.WithPadding(-1).EncodeToString(bstr)
			}
		}
		argArr = append(argArr, val)
		_ = i
	}

	amap["args"] = argArr
	bytes, err := json.Marshal(amap)
	if err != nil {
		return []byte(""), err
	}
	return bytes, err
}

// these are all the same:

// ToJSON to output a json version
func (p *Send) ToJSON() ([]byte, error) {
	p.Write(nil) // force existance of backingstr2
	bytes, err := str2ToJSON(p.backingstr2)
	return bytes, err
}

// ToJSON is not that efficient
func (p *Subscribe) ToJSON() ([]byte, error) {
	p.Write(nil) // force existance of backingstr2
	bytes, err := str2ToJSON(p.backingstr2)
	return bytes, err
}

// ToJSON is something that wastes memory.
func (p *Unsubscribe) ToJSON() ([]byte, error) {
	p.Write(nil) // force existance of backingstr2
	bytes, err := str2ToJSON(p.backingstr2)
	return bytes, err
}

// ToJSON is
func (p *Connect) ToJSON() ([]byte, error) {
	p.Write(nil) // force existance of backingstr2
	bytes, err := str2ToJSON(p.backingstr2)
	return bytes, err
}

// ToJSON is all the same
func (p *Disconnect) ToJSON() ([]byte, error) {
	p.Write(nil) // force existance of backingstr2
	bytes, err := str2ToJSON(p.backingstr2)
	return bytes, err
}

// ToJSON is
func (p *Lookup) ToJSON() ([]byte, error) {
	p.Write(nil) // force existance of backingstr2
	bytes, err := str2ToJSON(p.backingstr2)
	return bytes, err
}

func (str *Str2) String() string {
	b, _ := str2ToJSON(str)
	return string(b)
}

// These are all the same:

func (p *Send) String() string {
	b, _ := p.ToJSON()
	return string(b)
}

func (p *Subscribe) String() string {
	b, _ := p.ToJSON()
	return string(b)
}

// ToJSON is something that churns memory.
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
func (p *Subscribe) Write(conn net.Conn) error {
	if p.backingstr2 == nil {
		str := new(Str2)
		p.backingstr2 = str
		str.cmd = 'S' //
		str.args = make([][]byte, 0, 2+len(p.options)*2)
		str.args = append(str.args, p.address)
		str.args = append(str.args, p.addressAlias)
		str.args = packOptions(str.args, &p.options)
	}
	err := p.backingstr2.Write(conn)
	return err
}

// Write marshals to binary
func (p *Unsubscribe) Write(conn net.Conn) error {
	if p.backingstr2 == nil {
		str := new(Str2)
		p.backingstr2 = str
		str.cmd = 'U' //
		str.args = make([][]byte, 0, 2+len(p.options)*2)
		str.args = append(str.args, p.address)
		str.args = append(str.args, p.addressAlias)
		str.args = packOptions(str.args, &p.options)
	}
	err := p.backingstr2.Write(conn)
	return err
}

// Write forces backingstr2
func (p *Send) Write(conn net.Conn) error {
	if p.backingstr2 == nil {
		str := new(Str2)
		p.backingstr2 = str
		str.cmd = 'P' // Publish
		str.args = make([][]byte, 0, 5+len(p.options)*2)
		str.args = append(str.args, p.address)
		str.args = append(str.args, p.addressAlias)
		str.args = append(str.args, p.source)
		str.args = append(str.args, p.sourceAlias)
		str.args = append(str.args, p.payload)
		str.args = packOptions(str.args, &p.options)
	}
	err := p.backingstr2.Write(conn)
	return err
}

func (p *Connect) Write(conn net.Conn) error {
	if p.backingstr2 == nil {
		str := new(Str2)
		p.backingstr2 = str
		str.cmd = 'C'
		str.args = make([][]byte, 0, 0+len(p.options)*2)
		str.args = packOptions(str.args, &p.options)
	}
	err := p.backingstr2.Write(conn)
	return err
}

func (p *Disconnect) Write(conn net.Conn) error {
	if p.backingstr2 == nil {
		str := new(Str2)
		p.backingstr2 = str
		str.cmd = 'D'
		str.args = make([][]byte, 0, 0+len(p.options)*2)
		str.args = packOptions(str.args, &p.options)
	}
	err := p.backingstr2.Write(conn)
	return err
}

func (p *Lookup) Write(conn net.Conn) error {
	if p.backingstr2 == nil {
		str := new(Str2)
		p.backingstr2 = str
		str.cmd = 'D'
		str.args = make([][]byte, 0, 4+len(p.options)*2)
		str.args = append(str.args, p.address)
		str.args = append(str.args, p.addressAlias)
		str.args = append(str.args, p.source)
		str.args = append(str.args, p.sourceAlias)
	}
	err := p.backingstr2.Write(conn)
	return err
}

// ReadStr2 an Str2 packet.
func ReadStr2(reader io.Reader) (*Str2, error) {

	str := Str2{}
	oneByte := []uint8{0}
	n, err := reader.Read(oneByte) // read the command type
	if err != nil {
		return &str, err
	}
	str.cmd = CommandType(oneByte[0])

	// read array of byte arrays
	str.args, err = ReadArrayOfByteArray(reader)
	_ = n
	return &str, err
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

	lengths := [128]int{} // on the stack
	total := 0
	for i := uint8(0); i < argsLen; i++ { // read the lengths of the following strings
		aval, err := ReadVarLenInt(reader)
		if err != nil {
			return nil, err
		}
		lengths[i] = aval
		total += aval

	}
	if total > 1024*1024 {
		return nil, errors.New("Packet too long for this reality")
	}
	// now we can read the rest all at once

	bytes := make([]uint8, total) // alloc the base array
	n, err = reader.Read(bytes)   // timeout?
	if err != nil || n != total {
		return nil, err
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

// Write an Str2 packet.
func (str *Str2) Write(writer io.Writer) error {

	if writer == nil {
		return nil
	}

	oneByte := []uint8{0}
	oneByte[0] = uint8(str.cmd)
	n, err := writer.Write(oneByte)
	if err != nil {
		return err
	}
	err = WriteArrayOfByteArray(str.args, writer)
	_ = n
	return nil
}

// WriteArrayOfByteArray write count then lengths and then bytes
func WriteArrayOfByteArray(args [][]byte, writer io.Writer) error {
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
// A version of this without recursion would be better.
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
