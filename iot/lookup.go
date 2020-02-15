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

package iot

import (
	"fmt"
	"strconv"
)

func processLookup(me *LookupTableStruct, bucket *subscribeBucket, lookmsg *lookupMessage) {

	watcheditem, ok := bucket.mySubscriptions[lookmsg.h]
	count := uint32(0) // people watching
	if ok == false {
		// nobody watching
	} else {
		count = uint32(watcheditem.watchers.Size())
		// todo: add more info
	}
	// set count, in decimal
	str := strconv.FormatUint(uint64(count), 10)
	lookmsg.p.SetOption("count", []byte(str))
	lookmsg.ss.WriteDownstream(lookmsg.p, lookmsg.timestamp)
	err := bucket.looker.PushUp(lookmsg.p, lookmsg.h, lookmsg.timestamp)
	if err != nil {
		// we should be ashamed
		fmt.Println("FIXME x-sw")
	}

}

func processLookupDown(me *LookupTableStruct, bucket *subscribeBucket, lookmsg *lookupMessage) {

	watcheditem, ok := bucket.mySubscriptions[lookmsg.h]
	count := uint32(0) // people watching
	if ok == false {
		// nobody watching
	} else {
		count = uint32(watcheditem.watchers.Size())
		// todo: add more info
	}
	// set count, in decimal
	str := strconv.FormatUint(uint64(count), 10)
	lookmsg.p.SetOption("count", []byte(str))
	lookmsg.ss.WriteDownstream(lookmsg.p, lookmsg.timestamp)

}
