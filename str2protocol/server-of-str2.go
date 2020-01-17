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
	"github.com/awootton/knotfreeiot/iot"
	"github.com/awootton/knotfreeiot/iot/reporting"
)

// ServerOfStr2 - implement Str2 messages
func ServerOfStr2(subscribeMgr iot.PubsubIntf, addr string) *iot.SockStructConfig {

	config := iot.NewSockStructConfig(subscribeMgr)

	ServerOfStr2Init(config)

	iot.ServeFactory(config, addr)

	return config
}

// ServerOfStr2Init is to set default callbacks.
func ServerOfStr2Init(config *iot.SockStructConfig) {

	config.SetCallback(str2ServeCallback)

	servererr := func(ss *iot.SockStruct, err error) {
		str2LogThing.Collect("server closing")
	}
	config.SetClosecb(servererr)

	//  the writer
	handleTopicPayload := func(ss *iot.SockStruct, topic []byte, topicAlias *iot.HashType, returnAddress []byte, returnAlias *iot.HashType, payload []byte) error {

		cmd := Send{}
		cmd.source = returnAddress
		cmd.address = topic
		cmd.payload = payload

		err := cmd.Write(ss.GetConn())
		if err != nil {
			str2LogThing.Collect("error in str2 writer") //, n, err, cmd)
			return err
		}
		return nil
	}

	config.SetWriter(handleTopicPayload)
}

// str2ServeCallback is the default callback which calls the api
// to the pub sub manager.
//
func str2ServeCallback(ss *iot.SockStruct) {

	for {
		packet, err := ReadPacket(ss.GetConn())
		if err != nil {
			dis := Disconnect{}
			dis.options["error"] = []byte(err.Error())
			err2 := dis.Write(ss.GetConn())
			_ = err2
			ss.Close(err)
			return
		}
		// As much fun as it would be to make the following code into virtual methods
		// of the types involved (and I tried it) it's more annoying and harder to read
		// than just doing it all here.
		switch packet.(type) {

		case *Subscribe:
			p := packet.(*Subscribe)
			ss.SendSubscriptionMessage(p.address)

		case *Unsubscribe:
			p := packet.(*Unsubscribe)
			ss.SendSubscriptionMessage(p.address)

		case *Send:
			p := packet.(*Send)
			ss.SendPublishMessage(p.address, p.payload, p.source)

		case *Connect:
			p := packet.(*Connect)
			// TODO copy out the JWT
			_ = p

		case *Disconnect:
			p := packet.(*Disconnect)
			err := errors.New("exit") // TODO copy over options into json?
			ss.Close(err)
			_ = p

		default:
			dis := Disconnect{}
			dis.options["error"] = []byte("error unknown command")
			err2 := dis.Write(ss.GetConn())
			_ = err2
		}
	}
}

var str2LogThing = reporting.NewStringEventAccumulator(16)
