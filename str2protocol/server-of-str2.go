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
	"bufio"
	"errors"
	"io"
	"github.com/awootton/knotfreeiot/iot"
	"github.com/awootton/knotfreeiot/iot/reporting"
	"strings"
)

/** Here is the protocol.

There is a type byte "P" or "S" or whatever.

Then there is an arg count unsigned byte,
	so, we're up to two bytes now.
Then a coded list of int of length arg_count
	unsigned bytes <= 127 are a length
	else the lower 7 bits are the msb and the next byte is lsb. etc.
Finally, all the bytes of the args

so, in chars, the command "P topic msg" becomes:
P 2 5 3 t o p i c m s g
where 2 is the number of args 5 is the len of "topic" and 3 is the len of "msg"
followed by the string "topicmsg"

*/

// CommandType is
type CommandType uint8

const (
	none    = 0
	connect = iota + 1
	subscribe
	unsubscribe
	publish
)

// Bstr is like a string but is any bytes
// it could be utf8. many times it's 16 bytes for 128 bits
type Bstr []byte

// Str2 is the internal representation.
type Str2 struct {
	cmd  CommandType
	args []Bstr
}

// Read an Str2 packet.
func Read(reader io.Reader) (*Str2, error) {

	str := Str2{}
	b1 := []uint8{0}
	n, err := reader.Read(b1) // read the command type
	if err != nil {
		return &str, err
	}
	str.cmd = CommandType(b1[0]) // read the lengths of the followint args
	n, err = reader.Read(b1)
	if err != nil {
		return &str, err
	}
	argsLen := uint8(b1[0])
	lengths := make([]int, argsLen)
	total := 0
	for i := uint8(0); i < argsLen; i++ { // read the lengths of the following strings
		aval, err := readVarLen(reader)
		if err != nil {
			return &str, err
		}
		lengths[i] = aval
		total += aval

	}
	if total > 1024*1024 {
		return &str, errors.New("packet too long for reality")
	}
	// now we can read the rest all at once

	bytes := make([]uint8, total) // alloc the base array
	n, err = reader.Read(bytes)   // timeout?
	if err != nil || n != total {
		return &str, err
	}
	// now we can slice the args
	position := 0
	str.args = make([]Bstr, len(lengths))
	for i := 0; i < len(lengths); i++ {
		str.args[i] = bytes[position : position+lengths[i]]
		position += lengths[i]
	}
	return &str, nil
}

// Write an Str2 packet.
func (str *Str2) Write(writer io.Writer) error {

	b1 := []uint8{0}
	b1[0] = uint8(str.cmd)
	n, err := writer.Write(b1)
	if err != nil {
		return err
	}
	b1[0] = uint8(len(str.args))
	n, err = writer.Write(b1)
	if err != nil {
		return err
	}
	// write the lengths
	for i := 0; i < len(str.args); i++ {
		err = writeVarLen(uint32(len(str.args[i])), uint32(0x00), writer)
		if err != nil {
			return err
		}
	}
	// write the bytes
	for i := 0; i < len(str.args); i++ {
		n, err = writer.Write(str.args[i])
		if err != nil {
			return err
		}
	}
	_ = n
	return nil
}

func writeVarLen(len uint32, mask uint32, writer io.Writer) error {
	if len > 127 {
		// write the lsb first
		err := writeVarLen(len>>7, 0x80, writer)
		if err != nil {
			return err
		}
	}
	{
		b1 := []uint8{0}
		b1[0] = uint8((len & 0x7F) | mask)
		_, err := writer.Write(b1)
		return err
	}
}

func readVarLen(reader io.Reader) (int, error) {
	b1 := []uint8{0}
	_, err := reader.Read(b1)
	if err != nil {
		return 0, err
	}
	aval := 0
	remaining := 4
	for remaining != 0 {
		aval <<= 7
		if b1[0] >= 128 {
			aval |= int(b1[0]) & 0x7F
			remaining--
			_, err := reader.Read(b1)
			if err != nil {
				return 0, err
			}
		} else { // the common case
			aval |= int(b1[0])
			remaining = 0
			break
		}
	}
	return aval, nil
}

// ServerOfStrings - implement string messages
func ServerOfStrings(subscribeMgr iot.PubsubIntf, addr string) *iot.SockStructConfig {

	config := iot.NewSockStructConfig(subscribeMgr)

	ServerOfStringsInit(config)

	iot.ServeFactory(config, addr)

	return config
}

// ServerOfStringsInit is to set default callbacks.
func ServerOfStringsInit(config *iot.SockStructConfig) {

	config.SetCallback(strServeCallback)

	servererr := func(ss *iot.SockStruct, err error) {
		sosLogThing.Collect("server closing")
	}
	config.SetClosecb(servererr)

	//  the writer
	handleTopicPayload := func(ss *iot.SockStruct, topic []byte, payload []byte, returnAddress []byte) error {

		cmd := `add "` + string(returnAddress) + `" `
		// TODO: warning. this will BLOCK and jam up the whole machine.
		n, err := ss.GetConn().Write([]byte(cmd + "\n"))
		if err != nil || n != (len(cmd)+1) {
			return err
		}

		// make a 'command' (called 'pub'), serialize it and write it to the sock
		str := string(payload)
		cmd = `pub "` + string(topic) + `" ` + str
		// TODO: warning. this will BLOCK and jam up the whole machine.
		n, err = ss.GetConn().Write([]byte(cmd + "\n"))
		if err != nil || n != (len(cmd)+1) {
			sosLogThing.Collect("error in str writer") //, n, err, cmd)
			return err
		}

		return nil
	}

	config.SetWriter(handleTopicPayload)
}

// ServerOfStringsWrite translates the object into bytes and sends it.
// This is pretty easy since our objects are all strings.
// Clients will need this and it's NOT the same as config.writer for ServerOfStrings
func ServerOfStringsWrite(ss *iot.SockStruct, str string) error {
	bytes := []byte(str + "\n")
	conn := ss.GetConn()
	if conn != nil {
		n, err := conn.Write(bytes)
		if err != nil {
			ss.Close(err)
			return err
		}
		_ = n // fixme
	}
	return nil
}

// strServeCallback is the default callback which implements an api
// to the pub sub manager.
// This protcol echos back commands.
func strServeCallback(ss *iot.SockStruct) {

	reader := bufio.NewReader(ss.GetConn())

	for {
		text, err := reader.ReadString('\n')
		if err != nil {
			err = ServerOfStringsWrite(ss, "error "+err.Error())
			ss.Close(err)
			return
		}
		if len(text) <= 1 {
			err = ServerOfStringsWrite(ss, "error Empty sent")
			if err != nil {
				ss.Close(err)
			}
			continue
		}
		text = text[0 : len(text)-1] // remove the \n
		first, remaining := GetFirstWord(text)
		switch first {
		case "exit":
			ServerOfStringsWrite(ss, "exit")
			err := errors.New("exit")
			ss.Close(err)

		case "sub":
			topic := strings.Trim(remaining, " ")
			if len(topic) <= 0 {
				ServerOfStringsWrite(ss, "error say 'sub mytopic' and not "+text)
			} else {
				ss.SendSubscriptionMessage([]byte(topic))
				ServerOfStringsWrite(ss, "ok sub "+topic)
			}

		case "add":
			returnAddr := strings.Trim(remaining, " ")
			if len(returnAddr) <= 0 {
				ServerOfStringsWrite(ss, "error say 'add returnAddr' and not "+text)
			} else {
				ss.SetSelfAddress([]byte(returnAddr))
				ss.SendSubscriptionMessage([]byte(returnAddr))
				ServerOfStringsWrite(ss, "ok add "+returnAddr)
			}

		case "unsub":
			topic := strings.Trim(remaining, " ")
			if len(topic) <= 0 {
				ServerOfStringsWrite(ss, "error say 'unsub mytopic' and not "+text)
			} else {
				ss.SendUnsubscribeMessage([]byte(topic))
				ServerOfStringsWrite(ss, "ok unsub "+topic)
			}

		case "pub":
			topic, payload := GetFirstWord(remaining)
			if len(topic) <= 0 || len(payload) < 0 {
				ServerOfStringsWrite(ss, "error say 'pub mytopic mymessage' and not "+text)
			} else {
				topicHash := iot.HashType{}
				topicHash.FromString(topic)
				bytes := []byte(payload)
				ss.SendPublishMessage([]byte(topic), []byte(bytes), []byte("unknown")) // FIXME:
				ServerOfStringsWrite(ss, "ok pub "+topic+" "+payload)
			}

		default:
			ServerOfStringsWrite(ss, "error unknown command "+text)
		}
	}
}

// GetFirstWord will return the first word of str where the words are delimited by spaces.
// So this string: "pub aaa bbb" would get split into "pub" and "aaa bbb".
// It will also return the remaining words in string. If there is a '"' at the beginning of str then we'll match quotes to
// get the first word. There will be no escaping of quotes.
// So this string: '"aaa bbb" ccc ddd' would get split into "aaa bbb" and "ccc ddd".
func GetFirstWord(str string) (string, string) {

	str = strings.Trim(str, " ")
	if len(str) <= 1 {
		return str, ""
	}
	if str[0:1] == "\"" {
		str := str[1:]
		pos := strings.IndexByte(str, '"')
		if pos <= 0 { // no 2nd quote
			return strings.Trim(str, " "), ""
		}
		first := str[0:pos]
		second := str[pos+1:]
		return strings.Trim(first, " "), strings.Trim(second, " ")
	}
	// else look for a space
	pos := strings.IndexByte(str, ' ')
	if pos <= 0 { // no 2nd space
		return str, ""
	}
	first := str[0:pos]
	second := str[pos:]
	return strings.Trim(first, " "), strings.Trim(second, " ")
}

var sosLogThing = reporting.NewStringEventAccumulator(16)
