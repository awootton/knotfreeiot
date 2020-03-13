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

func processSubscribe(me *LookupTableStruct, bucket *subscribeBucket, submsg *subscriptionMessage) {

	watcheditem, ok := getWatchers(bucket, &submsg.h)
	if !ok {
		watcheditem = &watchedTopic{}
		watcheditem.name = submsg.h
		watcheditem.thetree = NewWithInt64Comparator()
		watcheditem.expires = 20 * 60 * me.getTime()

		setWatchers(bucket, &submsg.h, watcheditem)
		TopicsAdded.Inc()
	}
	// this is the important part:  add the caller to  the set
	watcheditem.put(submsg.ss.GetKey(), submsg.ss)
	namesAdded.Inc()
	err := bucket.looker.PushUp(submsg.p, submsg.h)
	if err != nil {
		// what? we're sad? todo: man up
		fmt.Println("FIXME kad")
	}

}

func processSubscribeDown(me *LookupTableStruct, bucket *subscribeBucket, submsg *subscriptionMessage) {

	watcheditem, ok := getWatchers(bucket, &submsg.h) //bucket.mySubscriptions[submsg.h]
	if !ok {
		fmt.Println("FIXME no such thing as subscribe down kadoo")
	} else {
		fmt.Println("FIXME no such thing as subscribe down asert2")
	}
	_ = watcheditem
}

func processUnsubscribe(me *LookupTableStruct, bucket *subscribeBucket, unmsg *unsubscribeMessage) {

	watcheditem, ok := getWatchers(bucket, &unmsg.h) //bucket.mySubscriptions[unmsg.h]
	if ok == true {
		watcheditem.remove(unmsg.ss.GetKey())
		if watcheditem.getSize() == 0 {
			// if nobody here is subscribing anymore then delete the entry in the hash
			//delete(bucket.mySubscriptions, unmsg.h)
			setWatchers(bucket, &unmsg.h, nil)
			// and also tell upstream that we're not interested anymore.
			err := bucket.looker.PushUp(unmsg.p, unmsg.h)
			if err != nil {
				// we should reconnect or what?
				fmt.Println("FIXME jkd334j")
			}
		}
		topicsRemoved.Inc()
	}
}

func processUnsubscribeDown(me *LookupTableStruct, bucket *subscribeBucket, unmsg *unsubscribeMessageDown) {

	watcheditem, ok := getWatchers(bucket, &unmsg.h) //bucket.mySubscriptions[unmsg.h]
	if !ok {
		fmt.Println("FIXME no such thing as UN subscribe down kad")
	} else {
		fmt.Println("FIXME no such thing as UN subscribe down 2")
	}
	_ = watcheditem

}
