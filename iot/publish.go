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

import "fmt"

func processPublish(me *LookupTableStruct, bucket *subscribeBucket, pubmsg *publishMessage) {

	watcheditem, ok := getWatchers(bucket, &pubmsg.h) //[pubmsg.h]
	if ok == false {
		// no publish possible !
		// it's sad really when someone sends messages to nobody.
		missedPushes.Inc()
		// send upstream publish
		err := bucket.looker.PushUp(pubmsg.p, pubmsg.h, pubmsg.timestamp)
		if err != nil {
			// what? we're sad? todo: man up
			// we should die and reconnect
		}
	} else {
		it := watcheditem.watchers.Iterator()
		for it.Next() {
			tmp, ok := it.Key().(uint64)
			if !ok {
				continue // this is bad
			}
			key := HalfHash(tmp)
			ss, ok := it.Value().(ContactInterface)
			if !ok {
				continue // real bad
			}
			if key != pubmsg.ss.GetKey() {
				if me.checkForBadSS(ss, watcheditem) == false {
					ss.WriteDownstream(pubmsg.p, pubmsg.timestamp)
					sentMessages.Inc()
				}
			}
		}
		err := bucket.looker.PushUp(pubmsg.p, pubmsg.h, pubmsg.timestamp)
		if err != nil {
			// what? we're sad? todo: man up
			// we should die and reconnect
			fmt.Println("FIXME tws0")
		}
	}

}

func processPublishDown(me *LookupTableStruct, bucket *subscribeBucket, pubmsg *publishMessageDown) {

	watcheditem, ok := getWatchers(bucket, &pubmsg.h) //bucket.mySubscriptions[pubmsg.h]
	if ok == false {
		// no publish possible !
		// it's sad really when someone sends messages to nobody.
		// this should not happen going down !
		missedPushes.Inc()
		// NO send upstream publish

	} else {
		it := watcheditem.watchers.Iterator()
		for it.Next() {
			tmp, ok := it.Key().(uint64)
			if !ok {
				continue // this is bad
			}
			key := HalfHash(tmp)
			ss, ok := it.Value().(ContactInterface)
			if !ok {
				continue // real bad
			}
			if key != pubmsg.ss.GetKey() {
				if me.checkForBadSS(ss, watcheditem) == false {
					ss.WriteDownstream(pubmsg.p, pubmsg.timestamp)
					sentMessages.Inc()
				}
			}
		}
	}

}
