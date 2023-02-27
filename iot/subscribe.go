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

	"github.com/awootton/knotfreeiot/packets"
	"github.com/awootton/knotfreeiot/tokens"
	"golang.org/x/crypto/nacl/box"
)

// the 'heartbeat' comes through here and times them out

func processSubscribe(me *LookupTableStruct, bucket *subscribeBucket, submsg *subscriptionMessage) {

	SpecialPrint(&submsg.p.PacketCommon, func() {
		fmt.Println("processSubscribe ", submsg.p.Address.String(), " in ", me.ex.Name)
	})

	watchedTopic, ok := getWatcher(bucket, &submsg.topicHash)
	if !ok {
		// make a new one as necessary
		watchedTopic = &WatchedTopic{}
		watchedTopic.name = submsg.topicHash
		watchedTopic.thetree = NewWithInt64Comparator()
		watchedTopic.expires = 20*60 + me.getTime()

		t, _ := submsg.p.GetOption("jwtidAlias") // don't they ALL have this?, except billing topics
		if len(t) != 0 {                         // it's always 64 bytes binary
			watchedTopic.jwtidAlias = string(t)
		}
		if watchedTopic.jwtidAlias == "123456" {
			// fmt.Println("have 123456 in new watcher", me.myname)
		}
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
	if ok {
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
		} else {
			//fmt.Println("found BillingAccumulator", watchedTopic.name)
		}
		watchedTopic.expires = 60*60 + me.getTime()
	} else {

	}

	watchedTopic.expires = 26*60 + me.getTime()

	// only the first subscriber can set the IPv6 address that lookup can return.
	val, ok = submsg.p.GetOption("IPv6")
	if ok {
		_, exists := watchedTopic.GetOption("IPv6")
		if exists == false {
			watchedTopic.SetOption("IPv6", val)
		}
	}

	val, ok = submsg.p.GetOption("misc")
	if ok {
		_, exists := watchedTopic.GetOption("misc")
		if exists == false {
			watchedTopic.SetOption("misc", val)
		}
	}

	// done: permanent subscription.
	// Pretty sure this is broken.
	subs := submsg.p
	box_bytes2, ok1 := subs.GetOption("reserved")
	pubk, ok2 := subs.GetOption("pubk")
	//tokn, _ := subs.GetOption("tokn")
	nonce, ok3 := subs.GetOption("nonce")
	if ok1 && ok2 && ok3 {

		fmt.Println("setting permanent node=", me.ex.Name)

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
		subMsgKey := submsg.ss.GetKey()
		watchedTopic.put(subMsgKey, wi)
	}

	namesAdded.Inc()
	err := bucket.looker.PushUp(submsg.p, submsg.topicHash)
	if err != nil {
		// what? we're sad? todo: man up
		fmt.Println("FIXME khjjkkkad", err, submsg.p)
	}
}

func heartBeatCallBack(me *LookupTableStruct, bucket *subscribeBucket, cmd *callBackCommand) {

	haveUpstream := len(me.upstreamRouter.channels) != 0

	//fmt.Println("Heartbeat for sub ", me.myname)
	defer cmd.wg.Done()
	// we don't delete them here and now. We queue up Unsubscribe packets.
	s := bucket.mySubscriptions
	for h, watchedItem := range s {
		expireAll := watchedItem.expires < cmd.now
		// FIRST, scan all the contact references and schedule the stale ones for deleteion.
		if expireAll {
			// expire the whole subscription because it's dead for too long
			//fmt.Println("expiring ALL", watchedItem.name)
		}

		it := watchedItem.Iterator()
		for it.Next() {
			key, item := it.KeyValue()
			if expireAll || item.contactInterface.GetClosed() {

				//fmt.Println("expiring subscription", watchedItem.name)

				p := new(packets.Unsubscribe)
				p.Address.Type = packets.BinaryAddress
				p.Address.Bytes = new([24]byte)[:]
				watchedItem.name.GetBytes(p.Address.Bytes)
				me.sendUnsubscribeMessage(item.contactInterface, p)
			}
			_ = key
		}
		_ = h
		// SECOND, check if this is a billing topic
		// if it's billing and it's over limits then write 'error Send' down.
		billingAccumulator, ok := watchedItem.IsBilling()
		if ok {
			if expireAll {
				setWatcher(bucket, &h, nil) // kill it now
			} else {
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
						if me.checkForBadContact(ci, watchedItem) == false {
							ci.WriteDownstream(p)
						}
					}
				}
			}
		}

		// THIRD, we'll need to send out the topic usage-stats occasionally.
		// from the guru only. For all topics that are not billing
		if watchedItem.jwtidAlias == "123456" && !haveUpstream {
			// fmt.Println("have 123456 in sub heart")
		}
		if len(watchedItem.jwtidAlias) > 0 && !haveUpstream {

			if watchedItem.nextBillingTime < cmd.now {

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
				// publish a "add-stats" command to billing topic
				// fmt.Println(" push to channelToAnyAide ", p)
				me.ex.channelToAnyAide <- p

				me.ex.Billing.AddUsage(&msg.KnotFreeContactStats, cmd.now, int(deltaTime))
			}
		}
	}
}

func processUnsubscribe(me *LookupTableStruct, bucket *subscribeBucket, unmsg *unsubscribeMessage) {

	watchedTopic, ok := getWatcher(bucket, &unmsg.topicHash)
	if ok == true {
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
