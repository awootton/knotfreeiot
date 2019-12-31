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

package strprotocol

import (
	"bufio"
	"errors"
	"knotfreeiot/iot"
	"knotfreeiot/iot/reporting"
	"strings"
)

/** Here is the protocol. Each line is a command.
There is command echo with the string "ok" or "error"

eg.

sub mytopic1

pub yourtopic2 hello to you

unsub mytopic1

returned strings are:

ok sub mytopic1

ok pub yourtopic2 hello to you

ok unsub mytopic1

pub mytopic1 hello from you

*/

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
