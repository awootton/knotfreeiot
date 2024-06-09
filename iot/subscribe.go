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
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/awootton/knotfreeiot/packets"
	"github.com/awootton/knotfreeiot/tokens"
	"golang.org/x/crypto/nacl/box"
)

// the 'heartbeat' comes through here and times them out

func processSubscribe(me *LookupTableStruct, bucket *subscribeBucket, submsg *subscriptionMessage) {

	wereSpecial := false
	SpecialPrint(&submsg.p.PacketCommon, func() {
		fmt.Println(me.ex.Name, "processSubscribe top con= ", submsg.ss.GetKey().Sig(), submsg.p.Sig())
		wereSpecial = true
	})

	// weAreTheFirst := false // if we're not the first then we don't need to propogate upwards
	watchedTopic, ok := getWatcher(bucket, &submsg.topicHash)
	if !ok {
		// weAreTheFirst = true
		// make a new one as necessary
		watchedTopic = &WatchedTopic{}
		watchedTopic.name = submsg.topicHash
		watchedTopic.thetree = NewWithInt64Comparator()
		watchedTopic.expires = 20*60 + me.getTime()

		t, _ := submsg.p.GetOption("jwtidAlias") // don't they ALL have this?, except billing topics
		if len(t) != 0 {                         // it's always 64 bytes binary
			watchedTopic.jwtidAlias = string(t)
		}
		// if watchedTopic.jwtidAlias == "123456" {
		// 	fmt.Println("have 123456 in new watcher", me.myname)
		// }
		setWatcher(bucket, &submsg.topicHash, watchedTopic)
		TopicsAdded.Inc()

		now := me.getTime()
		watchedTopic.nextBillingTime = now + 30 // 30 seconds to start with
		watchedTopic.lastBillingTime = now
	}

	wi := &watcherItem{}
	wi.contactInterface = submsg.ss

	// is this rigtht?
	opt, ok := submsg.p.GetOption("pub2self")
	if ok { // TODO: tear this out. Who uses pub2self ?
		_ = opt // assume it's 0 which means false
		wi.pub2self = false
	} else {
		// this is the first stop in the sub process - straight from a client.
		// we assume he wants publish to come back to him if he also is a sub
		submsg.p.SetOption("pub2self", []byte("0"))
		wi.pub2self = true // default is true
	}

	// The contact is going to send up this subscribe to the billing channel
	val, ok := submsg.p.GetOption("statsmax")
	if ok {

		// we need to make a note that we're a billing sub even if this is an aide
		// just add the bill ?

		// how do we keep anyone from sending a message to fake this?
		// check if there's already a BillingAccumulator
		_, haveBilling := watchedTopic.GetOption("bill")
		if !haveBilling {
			//fmt.Println("new BillingAccumulator", watchedTopic.name)
			stats := &tokens.KnotFreeContactStats{}
			err := json.Unmarshal(val, stats)
			if err == nil {
				ba := &BillingAccumulator{}
				ba.name = submsg.topicHash.String()[0:4]
				BucketCopy(stats, &ba.max)
				watchedTopic.SetOption("bill", ba)
			}
		}
		// else {
		// 	//fmt.Println("found BillingAccumulator", watchedTopic.name)
		// }
		watchedTopic.expires = 60*60 + me.getTime()
	}

	watchedTopic.expires = 26*60 + me.getTime()

	// only the first subscriber can set the IPv6 address that lookup can return.
	val, ok = submsg.p.GetOption("AAAA")
	if ok {
		_, exists := watchedTopic.GetOption("AAAA")
		if !exists {
			watchedTopic.SetOption("AAAA", val)
		}
	}

	val, ok = submsg.p.GetOption("misc")
	if ok {
		_, exists := watchedTopic.GetOption("misc")
		if !exists {
			watchedTopic.SetOption("misc", val)
		}
	}

	// done: permanent subscription.
	// Pretty sure this is broken.
	subs := submsg.p
	box_bytes2, ok1 := subs.GetOption("reserved")
	pubk, ok2 := subs.GetOption("pubk")
	nonce, ok3 := subs.GetOption("nonce")
	if ok1 && ok2 && ok3 {

		fmt.Println(me.ex.Name, "Subscribe setting permanent node=", me.ex.Name)

		hadError := ""

		var nonce2 [24]byte
		copy(nonce2[:], nonce)
		var pubk2 [32]byte
		copy(pubk2[:], pubk)

		clusterSecret := me.config.ce.PrivateKeyTemp

		dest_buffer := make([]byte, len(box_bytes2)-box.Overhead)
		dest_buffer = dest_buffer[:0]
		open_bytes, err := box.Open(dest_buffer, box_bytes2, &nonce2, &pubk2, clusterSecret)
		_ = err
		// this should be our original jwt for a name res
		//fmt.Println("recovered name jwt", string(open_bytes))

		publicKeyBytes := tokens.FindPublicKey("yRst")
		namePayload, ok := tokens.VerifyNameToken([]byte(open_bytes), []byte(publicKeyBytes))
		if !ok {
			fmt.Printf("ERROR tokens.VerifyNameToken got %v, want %v", "false", "true")
			hadError += fmt.Sprintf("ERROR tokens.VerifyNameToken got %v, want %v", "false", "true")
		}
		//fmt.Println("payload of name token ", namePayload)

		// and here's the trick
		// the public key in the namePayload must
		// match the pubk for the box

		if namePayload.JWTID != base64.RawURLEncoding.EncodeToString(pubk) {
			fmt.Printf(" pub key should match got %v, want %v", base64.RawURLEncoding.EncodeToString(pubk), namePayload.JWTID)
			hadError += fmt.Sprintf(" pub key should match got %v, want %v", base64.RawURLEncoding.EncodeToString(pubk), namePayload.JWTID)
		}

		// also the names must match
		unused := packets.Unsubscribe{}
		unused.Address.FromString(namePayload.Name)
		unused.Address.EnsureAddressIsBinary()
		if !bytes.Equal(subs.Address.Bytes, unused.Address.Bytes) {
			fmt.Printf("names must match '%v', want '%v'", subs.Address.String(), namePayload.Name)
			hadError += fmt.Sprintf("names must match '%v', want '%v'", subs.Address.String(), namePayload.Name)
		}

		if len(hadError) == 0 {
			watchedTopic.permanent = true
			watchedTopic.SetOption("reserved", namePayload)

			subMsgKey := submsg.ss.GetKey()
			watchedTopic.removeAll()
			watchedTopic.put(subMsgKey, wi)
		}

	} else {
		// this is the important part:  add the caller to the set
		// if watchedTopic.getSize() == 0 {
		// 	weAreTheFirst = true
		// }
		contactKey := submsg.ss.GetKey()
		if wereSpecial {
			fmt.Println(me.ex.Name, "Subscribe remembering con= ", submsg.ss.GetKey().Sig(), " for ", submsg.p.Sig())
		}
		// do we exist already?
		foundWi, exists := watchedTopic.get(contactKey)
		if exists {
			if wereSpecial {
				fmt.Println(me.ex.Name, "Subscribe already exists", contactKey.Sig(), " for ", submsg.p.Sig())
			}
			_ = foundWi
		} else {
			if wereSpecial {
				fmt.Println(me.ex.Name, "Subscribe adding new contact:", contactKey.Sig(), " for", submsg.p.Sig())
			}
			watchedTopic.put(contactKey, wi)
		}
	}

	// the common case is that we are the first subscriber.
	// are we the top or a guru ?

	if me.isGuru {
		_, ok := submsg.p.GetOption("noack")
		if !ok {
			if wereSpecial {
				fmt.Println(me.ex.Name, "Subscribe writing down:", submsg.ss.GetKey().Sig(), " for", submsg.p.Sig())
			}
			submsg.ss.WriteDownstream(submsg.p) // subs going down are suback's
		}
	} else {
		// we're an aide
		noUpstream := len(me.upstreamRouter.channels) == 0
		// there's a case when we are local and just running an aide.
		if noUpstream {
			if wereSpecial {
				fmt.Println(me.ex.Name, "Subscribe noUpstream writing down:", submsg.ss.GetKey().Sig(), " for", submsg.p.Sig())
			}
			// if bucket.index == 49 {
			// 	fmt.Println("subscribe noUpstream writing down TOP for bucket 49")
			// }

			submsg.ss.WriteDownstream(submsg.p) // subs going down are suback's

			// if bucket.index == 49 {
			// 	fmt.Println("subscribe noUpstream writing down DONE for bucket 49")
			// }
		}
	}

	namesAdded.Inc()
	err := bucket.looker.PushUp(submsg.p, submsg.topicHash)
	if err != nil {
		// what? we're sad? todo: man up
		fmt.Println("FIXME khjjkkkad", err, submsg.p)
	}
}

func processSubscribeDown(me *LookupTableStruct, bucket *subscribeBucket, submsg *subscriptionMessageDown) {

	wereSpecial := false
	SpecialPrint(&submsg.p.PacketCommon, func() {
		fmt.Println("processSubscribeDown ", submsg.p.Sig())
		wereSpecial = true
	})

	watcheditem, ok := getWatcher(bucket, &submsg.h) //bucket.mySubscriptions[pubmsg.h]
	if !ok {
		// this is weird but is it wrong? fmt.Println("processSubscribeDown ERROR no watcher for suback", submsg.p.Sig())
	} else {
		// what if there's more than one? Who get's the suback?
		// we'll do them all
		it := watcheditem.Iterator()
		for it.Next() {

			key, item := it.KeyValue()
			ci := item.contactInterface
			_ = key

			if wereSpecial {
				fmt.Println(me.ex.Name, "processSubscribeDown sending con= ", ci.GetKey().Sig(), submsg.p.Sig())
			}

			if !me.checkForBadContact(ci, watcheditem) {
				ci.WriteDownstream(submsg.p)
			}
		}
	}
}

// heartBeatCallBack is for this one bucket and we will ierate over all the subscriptions
// and expire the ones that are too old. we will also iterate over all the contacts and
// expire the ones that are closed. This blocks the entire bucket so hurry up.
// It times out after 1 sec for all 64 buckets!
func heartBeatCallBack(me *LookupTableStruct, bucket *subscribeBucket, cmd *callBackCommand) {

	haveUpstream := len(me.upstreamRouter.channels) != 0

	defer func() {
		cmd.wg.Done()
		cmd.donemap[bucket.index] = 1
	}()

	// if bucket.index == 49 {
	// 	fmt.Println("heartbeat TOP for bucket 49")
	// }

	s := bucket.mySubscriptions

	emptyTopics := make([]*WatchedTopic, 0, 10)

	channelToAnyAideMessages := make([]packets.Interface, 0, 10)

	for h, watchedItem := range s {

		if watchedItem.getSize() == 0 {
			// fmt.Println("Subscribe heartbeat expiring whole bucket", watchedItem.name.Sig())
			emptyTopics = append(emptyTopics, watchedItem)
			continue
		}

		expireAll := watchedItem.expires < cmd.now

		// FIRST, scan all the contact references and schedule the stale ones for deleteion.
		// if expireAll {
		// 	// expire the whole subscription because it's dead for too long
		// 	fmt.Println("expiring ALL", watchedItem.name)
		// }

		unsubsNeeded := make([]ContactInterface, 0, 10)

		it := watchedItem.Iterator()
		for it.Next() {
			key, item := it.KeyValue()
			// also clean up the closed ones.
			if expireAll || item.contactInterface.IsClosed() {

				// if expireAll {
				// 	fmt.Println("Subscribe heartbeat expiring all sub=", watchedItem.name.Sig(), " con=", item.contactInterface.GetKey().Sig())
				// } else {
				// 	fmt.Println("Subscribe heartbeat unsub sub=", watchedItem.name.Sig(), " con=", item.contactInterface.GetKey().Sig())
				// }

				unsubsNeeded = append(unsubsNeeded, item.contactInterface) // collect them now
			}
			_ = key
		}
		_ = h

		//go func() {
		for _, contact := range unsubsNeeded {
			// don't send an upsub. Just delete them Now
			watchedItem.remove(contact.GetKey())
		}
		//}()

		// SECOND, check if this is a billing topic
		// if it's billing and it's over limits then write 'error Send' down.

		// we don't need to do this in rea time do we?
		billingAccumulator, ok := watchedItem.IsBilling()
		if ok {
			// wtf. we cand't kill a watcher in the middle of a watcher iterator
			// if expireAll {
			// 	setWatcher(bucket, &h, nil) // kill it now<-NO, we'll do it later.
			// } else
			go func() {
				now := me.getTime()
				good, msg := billingAccumulator.AreUnderMax(now)
				if !good {
					fmt.Println("have token error", msg, watchedItem.name.GetUint64())
					p := &packets.Send{}
					p.Address.Bytes = new([24]byte)[:]
					p.Address.Type = packets.BinaryAddress
					h.GetBytes(p.Address.Bytes)
					p.Source.FromString("ping") // ie none
					p.Payload = []byte(msg)
					p.SetOption("error", p.Payload)
					// just like a publish down.
					it = watchedItem.Iterator()
					for it.Next() {
						key, item := it.KeyValue()
						_ = key
						ci := item.contactInterface
						if !me.checkForBadContact(ci, watchedItem) {
							ci.WriteDownstream(p)
						}
					}
				}
			}()
		}
		// THIRD, we'll need to send out the topic usage-stats occasionally.
		// from the guru only. For all topics that are not billing
		if len(watchedItem.jwtidAlias) > 0 && !haveUpstream {
			if watchedItem.nextBillingTime < cmd.now {
				// again, we can't do this right now.
				go func(watchedItem *WatchedTopic) {
					deltaTime := watchedItem.nextBillingTime - watchedItem.lastBillingTime
					watchedItem.lastBillingTime = watchedItem.nextBillingTime
					watchedItem.nextBillingTime += 60 // 60 secs after first time

					msg := &Stats{}

					msg.Subscriptions = float64(deltaTime) // means one per sec, one per min ... one. Q: is 300?

					// fmt.Println("sending subscribe deltat", deltaTime, "from ", me.myname)

					p := &packets.Send{}
					p.Address.FromString(watchedItem.jwtidAlias)
					p.Source.FromString("billing_stats_return_address_subscribe") // doesn't exist. use "ping" ?
					str, err := json.Marshal(msg)
					if err != nil {
						fmt.Println(" break fast ")
					}
					p.SetOption("add-stats", str)
					p.SetOption("stats-deltat", []byte(strconv.FormatInt(int64(deltaTime), 10)))
					// publish a "add-stats" command to billing topicget					me.ex.channelToAnyAide <- p
					// channelToAnyAideMessages = append(channelToAnyAideMessages, p)

					me.ex.Billing.AddUsage(&msg.KnotFreeContactStats, cmd.now, int(deltaTime))
				}(watchedItem)
			}
		}
		// they may have lost some items above
		if watchedItem.getSize() == 0 {
			emptyTopics = append(emptyTopics, watchedItem)
		}
	}

	// if bucket.index == 49 {
	// 	fmt.Println("heartbeat after watchedItems for bucket 49")
	// }

	// the http serve to a thing generates many of these.
	// we have to do this async
	for _, emptyBucket := range emptyTopics {
		// fmt.Println("Subscribe deleting entire empty bucket", emptyBucket.name)
		delete(bucket.mySubscriptions, emptyBucket.name) // the name is the hash
	}

	// if bucket.index == 49 {
	// 	fmt.Println("heartbeat after deletes for bucket 49")
	// }

	// async. we never know when PushUp might block
	go func() {
		for _, emptyBucket := range emptyTopics {
			// we have to send an unsubscribe to the upstream
			// can we watch for when the channel get's a little full?
			unmsg := new(packets.Unsubscribe)
			unmsg.Address.Type = packets.BinaryAddress
			unmsg.Address.Bytes = new([24]byte)[:]
			emptyBucket.name.GetBytes(unmsg.Address.Bytes)

			// I don't want to see "sendSubscriptionMessage channel full"
			msg := unsubscribeMessage{} // TODO: use a pool.
			// no contact msg.ss = ss
			msg.p = unmsg
			unmsg.Address.EnsureAddressIsBinary()
			msg.topicHash.InitFromBytes(unmsg.Address.Bytes)
			i := msg.topicHash.GetFractionalBits(me.theBucketsSizeLog2) // is 4. The first 4 bits of the hash.
			b := me.allTheSubscriptions[i]
			if len(b.incoming)*4 > cap(b.incoming)*3 {
				time.Sleep(time.Millisecond) // low priority
			}
			err := bucket.looker.PushUp(unmsg, emptyBucket.name)
			if err != nil {
				fmt.Println("Subscribe heartbeat unsub  PushUp error", err)
			}
		}
	}()

	// if bucket.index == 49 {
	// 	fmt.Println("heartbeat after starting unsubs for bucket 49")
	// }

	go func() {
		for _, p := range channelToAnyAideMessages {
			if len(me.ex.channelToAnyAide)*4 > cap(me.ex.channelToAnyAide)*3 {
				time.Sleep(time.Millisecond)
			}
			me.ex.channelToAnyAide <- p
		}
	}()

	// if bucket.index == 49 {
	// 	fmt.Println("heartbeat DONE for bucket 49")
	// }

}

func processUnsubscribe(me *LookupTableStruct, bucket *subscribeBucket, unmsg *unsubscribeMessage) {

	_ = me
	SpecialPrint(&unmsg.p.PacketCommon, func() {
		fmt.Println("processUnsubscribe con= ", unmsg.ss.GetKey().Sig(), "add ", unmsg.p.Sig())
	})

	watchedTopic, ok := getWatcher(bucket, &unmsg.topicHash)
	if ok {
		if watchedTopic.permanent {
			watchedTopic.remove(unmsg.ss.GetKey())
			// don't delete the entry
			err := bucket.looker.PushUp(unmsg.p, unmsg.topicHash)
			if err != nil {
				fmt.Println("help jkd334j 2")
			}

		} else {
			watchedTopic.remove(unmsg.ss.GetKey())
			_, isBilling := watchedTopic.IsBilling()
			if watchedTopic.getSize() == 0 && !isBilling {
				// if nobody here is subscribing anymore then delete the entry in the hash
				setWatcher(bucket, &unmsg.topicHash, nil)
				// and also tell upstream that we're not interested anymore.
				err := bucket.looker.PushUp(unmsg.p, unmsg.topicHash)
				if err != nil {
					fmt.Println("help jkd334j")
				}
			}
			topicsRemoved.Inc()
		}
	}
}
