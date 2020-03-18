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
	"encoding/json"
	"fmt"
)

func processPublish(me *LookupTableStruct, bucket *subscribeBucket, pubmsg *publishMessage) {

	watchedItem, ok := getWatchers(bucket, &pubmsg.h) //[pubmsg.h]
	if ok == false {
		// no publish possible !
		// it's sad really when someone sends messages to nobody.
		missedPushes.Inc()
		// send upstream publish
		err := bucket.looker.PushUp(pubmsg.p, pubmsg.h)
		if err != nil {
			// what? we're sad? todo: man up
			// we should die and reconnect
		}
	} else {

		billingAccumulator, ok := watchedItem.GetBilling()
		if ok {
			// statsHandled.Inc()
			// it's a billing channel
			// publishing to a billing channel is a special case
			billstr, ok := pubmsg.p.GetOption("stats")
			if ok {
				msg := &StatsWithTime{}
				err := json.Unmarshal(billstr, msg)
				if err == nil {

					billingAccumulator.Add(&msg.KnotFreeContactStats, msg.Start)

				} else {
					// statsUnmarshalFail.Inc()
				}
			} else {
				// statsMissingStats.Inc()
			}
			watchedItem.expires = 60 * 60 * me.getTime()

		} else {
			watchedItem.expires = 20 * 60 * me.getTime()

			it := watchedItem.Iterator()
			for it.Next() {

				key, item := it.KeyValue()
				ci := item.ci

				_, selfReturn := pubmsg.p.GetOption("toself")
				if selfReturn || key != pubmsg.ss.GetKey() {
					if me.checkForBadContact(ci, watchedItem) == false {
						//fmt.Println("pub down", string(pubmsg.p.Payload))
						if string(pubmsg.p.Payload) == "a_test_message2_45" {
							//fmt.Println("pub down", string(pubmsg.p.Payload))
						}
						ci.WriteDownstream(pubmsg.p)
						sentMessages.Inc()
					}
				}
			}
		}
		pubmsg.p.DeleteOption("toself")

		err := bucket.looker.PushUp(pubmsg.p, pubmsg.h)
		if err != nil {
			// what? we're sad? todo: man up
			// we should die and reconnect
			fmt.Println("FIXME tws0")
			// sendPushUpFail.Inc()
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
		watcheditem.expires = 20 * 60 * me.getTime()
		it := watcheditem.Iterator()
		for it.Next() {

			key, item := it.KeyValue()
			ci := item.ci

			if key != pubmsg.ss.GetKey() {
				if me.checkForBadContact(ci, watcheditem) == false {
					ci.WriteDownstream(pubmsg.p)
					sentMessages.Inc()
				}
			}
		}
	}

}
