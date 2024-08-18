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

func processPublish(me *LookupTableStruct, bucket *subscribeBucket, pubmsg *publishMessage) {

	wereSpecial := false
	SpecialPrint(&pubmsg.p.PacketCommon, func() {
		wereSpecial = true
	})

	if wereSpecial {
		fmt.Println(me.ex.Name, "processPublish top con=", pubmsg.ss.GetKey().Sig(), " to:", pubmsg.p.Sig())
	}

	watchedTopic, ok := getWatcher(bucket, &pubmsg.topicHash)
	if !ok {

		// nobody local is subscribing to this.
		// push it up to the next level
		missedPushes.Inc()
		// send upstream publish
		if !me.isGuru {
			err := bucket.looker.PushUp(pubmsg.p, pubmsg.topicHash)
			if err != nil {
				// what? sad? todo: man up
				// we should die and reconnect
				fmt.Println(me.ex.Name, "when a q push fails", string(pubmsg.p.Payload))
			}
		}
	} else {

		haveUpstream := len(me.upstreamRouter.channels) != 0

		// it has which holds the billingAccumulator
		billingAccumulator, isBilling := watchedTopic.IsBilling() // has billingAccumulator
		// it's really only supposed to ever even have a billingAccumulator unless this is a guru
		if isBilling {
			// we could be an aide. In that case don't process the command (below)
			// but also don't push down ever?

			// statsHandled.Inc()
			// it's a billing channel
			// publishing to a billing channel is a special case
			// todo: make this a m5n commmand.
			// this should really be msg == "add stats {...stats...}"
			// todo: implement "get stats"
			billstr, hasStats := pubmsg.p.GetOption("add-stats")

			// fmt.Println("isBilling ", haveUpstream, hasStats, string(pubmsg.p.Payload))

			if hasStats && !haveUpstream {

				deltat := 10
				deltatStr, ok := pubmsg.p.GetOption("stats-deltat")
				if ok {
					tmp, err := strconv.ParseInt(string(deltatStr), 10, 32)
					if err == nil {
						deltat = int(tmp)
					} else {
						fmt.Println(me.ex.Name, "ERROR FAIL to parse "+string(deltatStr))
					}
				} else {
					fmt.Println(me.ex.Name, "ERROR FAIL to find  stats-deltat")
				}

				msg := &Stats{}
				err := json.Unmarshal(billstr, msg)
				if err == nil {

					// if billingAccumulator.max.Subscriptions == 1 { // the test in billing_test
					// 	fmt.Println("publish BillingAccumulator ADDING", msg)
					// }
					now := me.getTime()
					billingAccumulator.AddUsage(&msg.KnotFreeContactStats, now, deltat)

				}
				//  else {
				// 	statsUnmarshalFail.Inc()
				// }
			} else {
				if !haveUpstream && !hasStats {
					// it's billing but it's not add-stats

					gotsend := serveBillingCommand(pubmsg.p, billingAccumulator, me.getTime())
					if len(me.ex.channelToAnyAide) >= cap(me.ex.channelToAnyAide) {
						fmt.Println("serveBillingCommand channelToAnyAide channel is full")
					}
					me.ex.channelToAnyAide <- &gotsend
				}
			}
			watchedTopic.Expires = 60*60 + me.getTime() // one hour

		} else {
			badContacts := make([]ContactInterface, 0)
			// do the WriteDownstream
			watchedTopic.Expires = 25*60 + me.getTime() //20 min
			// this is where the typical packet comes
			// fmt.Println("pub down", string(pubmsg.p.Payload))
			if wereSpecial && watchedTopic.thetree.Size() == 0 {
				fmt.Println(me.ex.Name, "processPublish getWatcher found topic but no subs con=", pubmsg.ss.GetKey().Sig(), " p:", pubmsg.p.Sig())
			}
			pubMsgKey := pubmsg.ss.GetKey()
			it := watchedTopic.Iterator()
			for it.Next() {

				key, item := it.KeyValue()
				ci := item.contactInterface

				if !item.pub2self {
					// everybody here gets the message right now
					if !me.checkForBadContact(ci, watchedTopic) {
						ci.WriteDownstream(pubmsg.p)
						sentMessages.Inc()
						if wereSpecial {
							fmt.Println(me.ex.Name, "WriteDownstream con=", ci.GetKey().Sig(), " ", pubmsg.p.Sig())
						}
					} else {
						if wereSpecial {
							fmt.Println(me.ex.Name, "haveBadContact ", ci.GetKey().Sig(), " ", pubmsg.p.Sig())
						}
					}
				} else {
					// we don't sent right back to ourselves. this is the typical case
					if key != pubMsgKey {
						if !me.checkForBadContact(ci, watchedTopic) {
							if wereSpecial {
								fmt.Println(me.ex.Name, "WriteDownstream2 ", ci.GetKey().Sig(), " ", pubmsg.p.Sig())
							}
							ci.WriteDownstream(pubmsg.p)
							sentMessages.Inc()
						} else {
							// can't we just delete it right now?
							// we're in the interator so no: watchedTopic.remove(ci.GetKey())

							if wereSpecial {
								fmt.Println(me.ex.Name, "haveBadContact2", ci.GetKey().Sig(), " ", pubmsg.p.Sig())
							}
							// what if we don't ??
							badContacts = append(badContacts, ci)
							// ci.WriteDownstream(pubmsg.p)
						}
					}
				}
			}
			for _, ci := range badContacts {
				if wereSpecial {
					fmt.Println(me.ex.Name, "Publish removing bad contact", ci.GetKey().Sig(), " sub:", pubmsg.ss.GetKey().Sig())
				}
				watchedTopic.remove(ci.GetKey())
			}
		}

		if wereSpecial {
			fmt.Println(me.ex.Name, "pub PushUp con=", pubmsg.ss.GetKey().Sig(), pubmsg.p.Sig())
		}
		if !me.isGuru { //me.upstreamRouter != nil {
			err := bucket.looker.PushUp(pubmsg.p, pubmsg.topicHash)
			if err != nil {
				fmt.Println("ERROR PushUp in processPublish ", err, pubmsg.p.Sig(), " in ", me.ex.Name)
				// sendPushUpFail.Inc()
			}
		}
	}

}

func processPublishDown(me *LookupTableStruct, bucket *subscribeBucket, pubmsg *publishMessageDown) {

	wereSpecial := false
	SpecialPrint(&pubmsg.p.PacketCommon, func() {
		fmt.Println(me.ex.Name, "processPublishDown ", pubmsg.p.Sig())
		wereSpecial = true
	})

	watcheditem, ok := getWatcher(bucket, &pubmsg.h) //bucket.mySubscriptions[pubmsg.h]
	if !ok {

		// SpecialPrint(&pubmsg.p.PacketCommon, func() {
		// 	//fmt.Println("special no publish possible this should not happen going down?", pubmsg.p.Address.String())
		// })
		// fmt.Println("Publish no publish possible this should not happen going down?", pubmsg.p.Address.String(), " in ", me.ex.Name)

		// there was an unsub but our parent doesnt know we should not be subscribing.
		// we should send an unsub to our parent

		if wereSpecial {
			fmt.Println(me.ex.Name, "processPublishDown no watcher, unsub in parent", pubmsg.p.Address.Sig())
		}

		unsub := packets.Unsubscribe{}
		unsub.Address = pubmsg.p.Address
		unsub.Address.EnsureAddressIsBinary()

		if !me.isGuru {
			err := bucket.looker.PushUp(&unsub, pubmsg.h)
			if err != nil {
				fmt.Println("error PushUp in processPublishDown ", err)
			}
		}
		missedPushes.Inc()

	} else {
		watcheditem.Expires = 25*60 + me.getTime() // 25 min
		if wereSpecial && watcheditem.thetree.Size() == 0 {
			fmt.Println(me.ex.Name, "processPublishDown getWatcher found topic but no subs ", " p:", pubmsg.p.Sig())
		}
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
			if !me.checkForBadContact(ci, watcheditem) {
				if wereSpecial {
					fmt.Println(me.ex.Name, "    processPublishDown WriteDownstream3 to con:", ci.GetKey().Sig(), " pub:", pubmsg.p.Sig())
				}
				ci.WriteDownstream(pubmsg.p)
				sentMessages.Inc()
			} else {
				if wereSpecial {
					fmt.Println(me.ex.Name, "    processPublishDown haveBadContact to con:", ci.GetKey().Sig(), " pub:", pubmsg.p.Sig())
				}
			}
		}
	}
}
