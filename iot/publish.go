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
)

func processPublish(me *LookupTableStruct, bucket *subscribeBucket, pubmsg *publishMessage) {

	var logs []string
	wereSpecial := false
	SpecialPrint(&pubmsg.p.PacketCommon, func() {
		wereSpecial = true
	})
	if wereSpecial {
		log := fmt.Sprintln("processPublish ", pubmsg.p.Address.String(), " p=", string(pubmsg.p.Payload), " in ", me.ex.Name, " from:", pubmsg.ss.GetKey())
		logs = append(logs, log)
		str := fmt.Sprint(pubmsg.ss.GetKey())
		_ = str
	}
	defer func() {
		for _, log := range logs {
			fmt.Print(log)
		}
	}()

	watchedTopic, ok := getWatcher(bucket, &pubmsg.topicHash)
	if ok == false {
		// nobody local is subscribing to this.
		// push it up to the next level
		missedPushes.Inc()
		// send upstream publish
		err := bucket.looker.PushUp(pubmsg.p, pubmsg.topicHash)
		if err != nil {
			// what? sad? todo: man up
			// we should die and reconnect
			fmt.Println("when a q push fails", string(pubmsg.p.Payload))
		}
	} else {

		billingAccumulator, ok := watchedTopic.IsBilling()
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
			watchedTopic.expires = 60 * 60 * me.getTime()

		} else {
			watchedTopic.expires = 20 * 60 * me.getTime()
			// this is where the typical packet comes
			// fmt.Println("pub down", string(pubmsg.p.Payload))
			pubMsgKey := pubmsg.ss.GetKey()
			it := watchedTopic.Iterator()
			for it.Next() {

				key, item := it.KeyValue()
				ci := item.contactInterface

				if item.pub2self == true {
					// everybody here gets the message right now
					if me.checkForBadContact(ci, watchedTopic) == false {
						ci.WriteDownstream(pubmsg.p)
						sentMessages.Inc()
						if wereSpecial {
							logs = append(logs, fmt.Sprintln("    WriteDownstream to", ci.GetKey(), " in ", me.ex.Name))
						}
					}
				} else {
					//   we don't sent right back to the
					// who just sent it to us.
					if key != pubMsgKey {
						if me.checkForBadContact(ci, watchedTopic) == false {
							ci.WriteDownstream(pubmsg.p)
							sentMessages.Inc()
						}
						if wereSpecial {
							logs = append(logs, fmt.Sprintln("    WriteDownstream2 to", ci.GetKey(), " in ", me.ex.Name, "on"))
						}
					}
				}

				//_, selfReturn := pubmsg.p.GetOption("toself")
				// //	if selfReturn || key != pubMsgKey
				// {
				// 	if me.checkForBadContact(ci, watchedItem) == false {
				// 		//fmt.Println("pub to contact", string(pubmsg.p.Address.String()), string(pubmsg.p.Payload), " in ", me.ex.Name)
				// 		ci.WriteDownstream(pubmsg.p)
				// 		sentMessages.Inc()
				// 	}
				// }
			}
		}
		//pubmsg.p.DeleteOption("toself")

		//fmt.Println("pub PushUp", string(pubmsg.p.Address), string(pubmsg.p.Payload), " in ", me.ex.Name)

		err := bucket.looker.PushUp(pubmsg.p, pubmsg.topicHash)
		if err != nil {
			// what? we're sad? todo: man up
			// we should die and reconnect
			fmt.Println("FIXME tws0")
			// sendPushUpFail.Inc()
		}
	}

}

func processPublishDown(me *LookupTableStruct, bucket *subscribeBucket, pubmsg *publishMessageDown) {

	var logs []string
	wereSpecial := false
	SpecialPrint(&pubmsg.p.PacketCommon, func() {
		//fmt.Println("processPublishDown ", pubmsg.p.Address.String(), " in ", me.ex.Name)
		wereSpecial = true
	})
	if wereSpecial {
		logs = append(logs, fmt.Sprintln("processPublishDown ", pubmsg.p.Address.String(), " in ", me.ex.Name))
	}
	defer func() {
		for _, log := range logs {
			fmt.Print(log)
		}
	}()

	watcheditem, ok := getWatcher(bucket, &pubmsg.h) //bucket.mySubscriptions[pubmsg.h]
	if ok == false {

		SpecialPrint(&pubmsg.p.PacketCommon, func() {
			//fmt.Println("special no publish possible this should not happen going down?", pubmsg.p.Address.String())
		})
		fmt.Println("no publish possible this should not happen going down?", pubmsg.p.Address.String(), " in ", me.ex.Name)
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
			ci := item.contactInterface
			_ = key
			// key is a watched item key which is a Contact key
			// pubmsg.h is a HashType. 24 bytes, of the topic
			// comparing them makes no sense
			//if key != pubmsg.h.GetHalfHash() {// why would these EVER not be ==
			// always send to everyone
			// need: system test watching for duplicates.
			if me.checkForBadContact(ci, watcheditem) == false {
				ci.WriteDownstream(pubmsg.p)
				sentMessages.Inc()
				if wereSpecial {
					logs = append(logs, fmt.Sprintln("    WriteDownstream3 to", ci.GetKey(), " in ", me.ex.Name, "on", ci.GetKey()))
				}
			}
		}
	}
}
