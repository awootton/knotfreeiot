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
	"errors"
	"io"
	"net"
)

/** There is a struct (Str2) that can represent any packet.
There are individual types for each of the packets.
We read the Str2 and then construct the Packet from that and visa versa when writing.
It may seem like we're duplicating data and making a mess but the structs
are full of slices backed by the same data.
*/

// PacketCommon is stuff the packets all have
type PacketCommon struct {

	// for internal use only:
	src     *Str2           // might be nil
	options map[string]Bstr //OptionsMap // optional
}

// MessageCommon is
type MessageCommon struct {
	PacketCommon

	// the fields:
	source      []byte
	destination []byte
}

// OptionsMap is often nil
type OptionsMap map[string]Bstr

// Send aka 'publish' sends data to destination possibly expecting a reply to source.
//
type Send struct {
	MessageCommon
	data []byte
}

// Online returns information on the dest to source.
// can be used to verify existance of an endpoint prior to subscribe.
type Online struct {
	MessageCommon
}

// Subscribe is to declare that the Thing has an address.
// Presumably one would Subscribe before a Send.
type Subscribe struct {
	PacketCommon
	destination []byte
}

// Unsubscribe might prevent future reception at the indicated destination address.
type Unsubscribe struct {
	PacketCommon
	destination []byte
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
type Bstr []byte

// Str2 is the internal representation.
type Str2 struct {
	cmd  CommandType
	args []Bstr
}

// Packet is virtual functions for all packets.
// Basically versions of marshal and unmarshal.
type Packet interface {
	Write(conn net.Conn) error // write to Str2 and then to conn
	Fill(*Str2) error          // copy data from Str2 that was just read
}

// GetIPV6Option is an example of using options
func GetIPV6Option(p *PacketCommon) []byte {
	return p.options["ip"]
}

// ReadPacket attempts to obtain a valid Packet from the stream
func ReadPacket(reader io.Reader) (Packet, error) {

	str, err := ReadStr2(reader)
	if err != nil {
		return nil, err
	}
	var p Packet
	switch str.cmd {
	case 'P': // Send aka Publish
		p = &Send{}
		err := p.Fill(str)
		if err != nil {
			return nil, err
		}
	}

	return p, nil
}

func unpackOptions(args []Bstr, options *map[string]Bstr) {

}

// Fill implements the 2nd part of an unmarshal.
// See ReadPacket
func (p *Subscribe) Fill(str *Str2) error {

	if len(str.args) < 1 {
		return errors.New("too few args for Subscribe")
	}
	p.destination = str.args[0]
	unpackOptions(str.args[1:], &p.options)
	return nil
}

// Fill implements the 2nd part of an unmarshal.
func (p *Unsubscribe) Fill(str *Str2) error {

	if len(str.args) < 2 {
		return errors.New("too few args for Send")
	}
	p.destination = str.args[0]
	unpackOptions(str.args[1:], &p.options)
	return nil
}

// Fill implements the 2nd part of an unmarshal
func (p *Send) Fill(str *Str2) error {

	if len(str.args) < 3 {
		return errors.New("too few args for Send")
	}

	p.source = str.args[0]
	p.destination = str.args[1]
	p.data = str.args[2]
	unpackOptions(str.args[3:], &p.options)
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

func packOptions(args []Bstr, options *map[string]Bstr) []Bstr {
	for k, v := range *options {
		args = append(args, Bstr(k))
		args = append(args, v)
	}
	return args
}

// Write implements a marshal operation.
func (p *Subscribe) Write(conn net.Conn) error {
	if p.src == nil {
		str := new(Str2)
		p.src = str
		str.cmd = 'S' //
		str.args = make([]Bstr, 1+len(p.options)*2)
		str.args = append(str.args, p.destination)
		str.args = packOptions(str.args, &p.options)
	}
	err := p.src.Write(conn)
	return err
}

func (p *Unsubscribe) Write(conn net.Conn) error {
	if p.src == nil {
		str := new(Str2)
		p.src = str
		str.cmd = 'U' //
		str.args = make([]Bstr, 1+len(p.options)*2)
		str.args = append(str.args, p.destination)
		str.args = packOptions(str.args, &p.options)
	}
	err := p.src.Write(conn)
	return err
}

func (p *Send) Write(conn net.Conn) error {
	if p.src == nil {
		str := new(Str2)
		p.src = str
		str.cmd = 'P' // Publish
		str.args = make([]Bstr, 3+len(p.options)*2)
		str.args = append(str.args, p.source)
		str.args = append(str.args, p.destination)
		str.args = append(str.args, p.data)
		str.args = packOptions(str.args, &p.options)
	}
	err := p.src.Write(conn)
	return err
}

func (p *Connect) Write(conn net.Conn) error {
	if p.src == nil {
		str := new(Str2)
		p.src = str
		str.cmd = 'C'
		str.args = make([]Bstr, 0+len(p.options)*2)
		str.args = packOptions(str.args, &p.options)
	}
	err := p.src.Write(conn)
	return err
}

func (p *Disconnect) Write(conn net.Conn) error {
	if p.src == nil {
		str := new(Str2)
		p.src = str
		str.cmd = 'D'
		str.args = make([]Bstr, 0+len(p.options)*2)
		str.args = packOptions(str.args, &p.options)
	}
	err := p.src.Write(conn)
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

	// n, err = reader.Read(oneByte)
	// if err != nil {
	// 	return &str, err
	// }
	// argsLen := uint8(oneByte[0])
	// lengths := make([]int, argsLen)
	// total := 0
	// for i := uint8(0); i < argsLen; i++ { // read the lengths of the following strings
	// 	aval, err := readVarLen(reader)
	// 	if err != nil {
	// 		return &str, err
	// 	}
	// 	lengths[i] = aval
	// 	total += aval

	// }
	// if total > 1024*1024 {
	// 	return &str, errors.New("packet too long for reality")
	// }
	// // now we can read the rest all at once

	// bytes := make([]uint8, total) // alloc the base array
	// n, err = reader.Read(bytes)   // timeout?
	// if err != nil || n != total {
	// 	return &str, err
	// }
	// // now we can slice the args
	// position := 0
	// str.args = make([]Bstr, len(lengths))
	// for i := 0; i < len(lengths); i++ {
	// 	str.args[i] = bytes[position : position+lengths[i]]
	// 	position += lengths[i]
	// }
	_ = n
	return &str, err
}

// ReadArrayOfByteArray to read an array of byte arrays
func ReadArrayOfByteArray(reader io.Reader) ([]Bstr, error) {

	oneByte := []uint8{0}
	// read the lengths of the following args
	n, err := reader.Read(oneByte)
	if err != nil {
		return nil, err
	}
	argsLen := uint8(oneByte[0])
	lengths := [266]int{} // on the stack
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
		return nil, errors.New("packet too long for this reality")
	}
	// now we can read the rest all at once

	bytes := make([]uint8, total) // alloc the base array not locally
	n, err = reader.Read(bytes)   // timeout?
	if err != nil || n != total {
		return nil, err
	}
	// now we can slice the args
	position := 0
	args := make([]Bstr, argsLen) // array of slices
	for i := 0; i < int(argsLen); i++ {
		args[i] = bytes[position : position+lengths[i]]
		position += lengths[i]
	}
	return args, nil
}

// Write an Str2 packet.
func (str *Str2) Write(writer io.Writer) error {

	oneByte := []uint8{0}
	oneByte[0] = uint8(str.cmd)
	n, err := writer.Write(oneByte)
	if err != nil {
		return err
	}

	err = WriteArrayOfByteArray(str.args, writer)
	// oneByte[0] = uint8(len(str.args))
	// n, err = writer.Write(oneByte)
	// if err != nil {
	// 	return err
	// }
	// // write the lengths
	// for i := 0; i < len(str.args); i++ {
	// 	err = writeVarLen(uint32(len(str.args[i])), uint32(0x00), writer)
	// 	if err != nil {
	// 		return err
	// 	}
	// }
	// // write the bytes
	// for i := 0; i < len(str.args); i++ {
	// 	n, err = writer.Write(str.args[i])
	// 	if err != nil {
	// 		return err
	// 	}
	// }
	_ = n
	return nil
}

// WriteArrayOfByteArray write count then lengths and then bytes
func WriteArrayOfByteArray(args []Bstr, writer io.Writer) error {
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
func WriteVarLenInt(len uint32, mask uint8, writer io.Writer) error {
	if len > 127 {
		// write the lsb first
		err := WriteVarLenInt(len>>7, 0x80, writer)
		if err != nil {
			return err
		}
	}
	{
		oneByte := []uint8{0}
		oneByte[0] = uint8((len & 0x7F) | uint32(mask))
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
