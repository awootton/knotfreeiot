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

	"github.com/awootton/knotfreeiot/tokens"
)

func processSubscribe(me *LookupTableStruct, bucket *subscribeBucket, submsg *subscriptionMessage) {

	//fmt.Println("top of processSubscribe", string(submsg.p.Address.String()), " in ", me.ex.Name)

	watchedItem, ok := getWatcher(bucket, &submsg.h)
	if !ok {
		watchedItem = &watchedTopic{}
		watchedItem.name = submsg.h
		watchedItem.thetree = NewWithInt64Comparator()
		watchedItem.expires = 20 * 60 * me.getTime()

		t, _ := submsg.p.GetOption("jwtidAlias")
		if len(t) == 24 {
			watchedItem.jwtidAlias = string(t)
		}
		setWatcher(bucket, &submsg.h, watchedItem)
		TopicsAdded.Inc()

		now := me.getTime()
		watchedItem.nextBillingTime = now + 30 // 30 seconds to start with
		watchedItem.lastBillingTime = now

	}
	// this is the important part:  add the caller to  the set
	watchedItem.put(submsg.ss.GetKey(), submsg.ss)

	// check some options
	val, ok := submsg.p.GetOption("statsmax")
	if ok {
		// now we're a billing channel
		stats := &tokens.KnotFreeContactStats{}
		err := json.Unmarshal(val, stats)
		if err == nil {
			ba := &BillingAccumulator{}
			ba.name = submsg.h.String()[0:4]
			BucketCopy(stats, &ba.max)
			watchedItem.SetOption("bill", ba)
		}
		watchedItem.expires = 60 * 60 * me.getTime()
	} else {
		watchedItem.expires = 20 * 60 * me.getTime()
	}
	// todo: permanent subscription.

	// only the first subscriber can set the IPv6 address that lookup can return.
	val, ok = submsg.p.GetOption("IPv6")
	if ok {
		_, exists := watchedItem.GetOption("IPv6")
		if exists == false {
			watchedItem.SetOption("IPv6", val)
		}
	}

	val, ok = submsg.p.GetOption("misc")
	if ok {
		_, exists := watchedItem.GetOption("misc")
		if exists == false {
			watchedItem.SetOption("misc", val)
		}
	}

	namesAdded.Inc()
	err := bucket.looker.PushUp(submsg.p, submsg.h)
	if err != nil {
		// what? we're sad? todo: man up
		fmt.Println("FIXME kad", err, submsg.p)
	}
}

func processSubscribeDown(me *LookupTableStruct, bucket *subscribeBucket, submsg *subscriptionMessageDown) {

	watcheditem, ok := getWatcher(bucket, &submsg.h)
	if !ok {
		fmt.Println("FIXME no such thing as subscribe down kadoo")
	} else {
		fmt.Println("FIXME no such thing as subscribe down asert2")
	}
	_ = watcheditem
}

func processUnsubscribe(me *LookupTableStruct, bucket *subscribeBucket, unmsg *unsubscribeMessage) {

	watcheditem, ok := getWatcher(bucket, &unmsg.h)
	if ok == true {
		watcheditem.remove(unmsg.ss.GetKey())
		_, isBilling := watcheditem.GetBilling()
		if watcheditem.getSize() == 0 && !isBilling {
			// if nobody here is subscribing anymore then delete the entry in the hash
			setWatcher(bucket, &unmsg.h, nil)
			// and also tell upstream that we're not interested anymore.
			err := bucket.looker.PushUp(unmsg.p, unmsg.h)
			if err != nil {
				fmt.Println("help jkd334j")
			}
		}
		topicsRemoved.Inc()
	}
}

func processUnsubscribeDown(me *LookupTableStruct, bucket *subscribeBucket, unmsg *unsubscribeMessageDown) {

	watcheditem, ok := getWatcher(bucket, &unmsg.h) //bucket.mySubscriptions[unmsg.h]
	if !ok {
		fmt.Println("FIXME no such thing as UN subscribe down kad")
	} else {
		fmt.Println("FIXME no such thing as UN subscribe down 2")
	}
	_ = watcheditem

}
