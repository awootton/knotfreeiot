// Copyright 2019,2020,2021 Alan Tracey Wootton
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

package iot

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/awootton/knotfreeiot/packets"
)

// FIXME: needs test. This is not right.
// the reply doesn't go down - it becomes a publish to a
// return address.

func processLookup(me *LookupTableStruct, bucket *subscribeBucket, lookmsg *lookupMessage) {

	lookReplyObject := LookReply{}

	watcheditem, ok := getWatcher(bucket, &lookmsg.topicHash)
	count := uint32(0) // people watching
	if ok == false {
		// nobody watching
		lookReplyObject.Null = true
	} else {
		count = uint32(watcheditem.getSize())
		// todo: add more info
		lookReplyObject.Null = false
		lookReplyObject.Count = count
	}
	// set count, in decimal
	str := strconv.FormatUint(uint64(count), 10)
	lookmsg.p.SetOption("count", []byte(str))
	level := int64(0)
	levelBytes, ok := lookmsg.p.GetOption("level")
	if ok {
		level, _ = strconv.ParseInt(string(levelBytes), 10, 32)
	}
	level += 1
	lookmsg.p.SetOption("level", []byte(strconv.FormatUint(uint64(level), 10)))

	err := bucket.looker.PushUp(lookmsg.p, lookmsg.topicHash)
	if err != nil {
		// we should be ashamed
		fmt.Println("FIXME x-sw")
	}

	// now, reply to the retrun address. With what type of message?
	// Has to be a send unless we want to add another type
	send := packets.Send{}
	send.Address = lookmsg.p.Source
	send.Source = lookmsg.p.Address
	send.SetOption("isLookup", []byte("true"))
	send.CopyOptions(&lookmsg.p.PacketCommon)
	// we have level
	// we have the count at this level
	nodeName := me.ex.Name

	lookReplyObject.Level = uint32(level)
	//lookReplyObject.Count = int(count)
	lookReplyObject.Node = nodeName
	repl, err := json.Marshal(lookReplyObject)

	send.Payload = repl

	val, ok := lookmsg.p.GetOption("debg")
	if ok {
		send.SetOption("debg", val)
	}

	me.ex.channelToAnyAide <- &send

	SpecialPrint(&lookmsg.p.PacketCommon, func() {
		json, _ := send.ToJSON()
		fmt.Println("Lookup channelToAnyAide because ", string(json), " in ", me.ex.Name, "on")
	})

	_ = nodeName
}

type LookReply struct {
	Level       uint32
	Count       uint32
	Null        bool
	Node        string // node name
	IsPermanent bool

	// What else?
}

// TODO: chop out the dead wood in subscribe etc.
// there is not one of these. Lookup replies to the return address
//func processLookupDown(me *LookupTableStruct, bucket *subscribeBucket, lookmsg *lookupMessageDown) {

//	fmt.Println("FIXME processLookupDown FIXME processLookupDown FIXME processLookupDown FIXME processLookupDown FIXME processLookupDown ")

// FIXME: needs test. This is not right. there is no processLookupDown
// the reply doesn't go down - it becomes a publish to a
// return address.

// watcheditem, ok := getWatcher(bucket, &lookmsg.h)
// count := uint32(0) // people watching
// if ok == false {
// 	// nobody watching
// } else {
// 	count = uint32(watcheditem.getSize())
// 	// todo: add more info
// }
// // set count, in decimal
// str := strconv.FormatUint(uint64(count), 10)
// lookmsg.p.SetOption("count", []byte(str))
// lookmsg.ss.WriteDownstream(lookmsg.p)

//}
