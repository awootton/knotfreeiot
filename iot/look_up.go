// Copyright 2019 Alan Tracey Wootton
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

	"github.com/awootton/knotfreeiot/packets"
	"github.com/emirpasic/gods/trees/redblacktree"
	"github.com/emirpasic/gods/utils"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// LookupTableStruct is what we're up to
type LookupTableStruct struct {
	//
	allTheSubscriptions []subscribeBucket

	key HashType
}

// NewLookupTable makes a LookupTableStruct, usually a singleton.
// In the tests we call here and then use the result to init a server.
// Starts 64 go routines that are hung on their q's
func NewLookupTable(projectedTopicCount int) *LookupTableStruct {
	psMgr := LookupTableStruct{}
	psMgr.key.Random()
	portion := projectedTopicCount / int(theBucketsSize)
	portion2 := projectedTopicCount >> theBucketsSizeLog2 // we can init the hash maps big
	if portion != portion2 {
		fmt.Println("EPIC FAIL theBucketsSizeLog2 != uint(math.Log2(float64(theBucketsSize)))")
	}
	psMgr.allTheSubscriptions = make([]subscribeBucket, theBucketsSize)
	for i := uint(0); i < theBucketsSize; i++ {
		psMgr.allTheSubscriptions[i].mySubscriptions = make(map[HashType]*watchedTopic, portion)
		tmp := make(chan interface{}, 32)
		psMgr.allTheSubscriptions[i].incoming = &tmp
		psMgr.allTheSubscriptions[i].subscriber = &psMgr
		go psMgr.allTheSubscriptions[i].processMessages(&psMgr)
	}
	return &psMgr
}

// sendSubscriptionMessage will create a message object, copy pointers to it so it'll own them now, and queue the message.
func (me *LookupTableStruct) sendSubscriptionMessage(ss ContactInterface, p *packets.Subscribe) {

	msg := subscriptionMessage{}
	msg.ss = ss
	msg.p = p
	msg.h.InitFromBytes(p.AddressAlias)
	i := msg.h.GetFractionalBits(theBucketsSizeLog2)
	b := me.allTheSubscriptions[i]
	*b.incoming <- msg
}

// SendUnsubscribeMessage will create a message object, copy pointers to it so it'll own them now, and queue the message.
func (me *LookupTableStruct) sendUnsubscribeMessage(ss ContactInterface, p *packets.Unsubscribe) {

	msg := unsubscribeMessage{}
	msg.ss = ss
	msg.p = p
	msg.h.InitFromBytes(p.AddressAlias)
	i := msg.h.GetFractionalBits(theBucketsSizeLog2)
	b := me.allTheSubscriptions[i]
	*b.incoming <- msg
}

// SendUnsubscribeMessage will create a message object, copy pointers to it so it'll own them now, and queue the message.
func (me *LookupTableStruct) sendLookupMessage(ss ContactInterface, p *packets.Lookup) {

	msg := lookupMessage{}
	msg.ss = ss
	msg.p = p
	msg.h.InitFromBytes(p.AddressAlias)
	i := msg.h.GetFractionalBits(theBucketsSizeLog2)
	b := me.allTheSubscriptions[i]
	*b.incoming <- msg
}

// SendPublishMessage will create a message object, copy pointers to it so it'll own them now, and queue the message.
func (me *LookupTableStruct) sendPublishMessage(ss ContactInterface, p *packets.Send) {

	msg := publishMessage{}
	msg.ss = ss
	msg.p = p
	msg.h.InitFromBytes(p.AddressAlias)
	i := msg.h.GetFractionalBits(theBucketsSizeLog2)
	b := me.allTheSubscriptions[i]
	*b.incoming <- msg
}

// GetAllSubsCount returns the count of subscriptions and the
// average depth of the channels.
func (me *LookupTableStruct) GetAllSubsCount() (int, int) {
	count := 0
	qdepth := 0
	for _, b := range me.allTheSubscriptions {
		count += len(b.mySubscriptions)
		qdepth += (len(*b.incoming))
	}
	qdepth = qdepth / len(me.allTheSubscriptions)
	return count, qdepth
}

// TODO: implement a pool of the incoming types.

// A grab bag of paranoid ideas about bad states. FIXME: let's be more formal.
func (me *LookupTableStruct) checkForBadSS(badsock ContactInterface, pubstruct *watchedTopic) bool {

	// forgetme := false
	// //if badsock.conn == nil {
	// //	forgetme = true
	// //}
	// if badsock.ele == nil {
	// 	forgetme = true
	// }
	// if forgetme {
	// 	for topic, realName := range badsock.topicToName {
	// 		//me.SendUnsubscribeMessage(badsock, realName)
	// 		_ = realName
	// 		badsock.topicToName = nil
	// 		_ = topic
	// 	}
	// 	delete(pubstruct.watchers, badsock.key)
	// 	return true
	// }
	return false
}

func (bucket *subscribeBucket) processMessages(me *LookupTableStruct) {

	for {
		msg := <-*bucket.incoming // wait right here
		switch msg.(type) {
		case subscriptionMessage:
			submsg := msg.(subscriptionMessage)
			substruct := bucket.mySubscriptions[submsg.h]
			if substruct == nil {
				substruct = &watchedTopic{}
				substruct.name = submsg.h
				substruct.watchers = NewWithInt64Comparator() //make(map[HalfHash]ContactInterface, 0)
				bucket.mySubscriptions[submsg.h] = substruct
				topicsAdded.Inc()
			}
			// this is the important part:  add the caller to  the set
			substruct.watchers.Put(uint64(submsg.ss.GetKey()), submsg.ss)
			namesAdded.Inc()

		case publishMessage:
			pubmsg := msg.(publishMessage)
			pubstruct, ok := bucket.mySubscriptions[pubmsg.h]
			if ok == false {
				// no publish possible !
				// it's sad really when someone sends messages to nobody.
				missedPushes.Inc()
				// send upstream publish
			} else {
				it := pubstruct.watchers.Iterator()
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
						if me.checkForBadSS(ss, pubstruct) == false {
							ss.WriteDownstream(pubmsg.p)
							sentMessages.Inc()
						}
					}
				}
				// send upstream publish
			}

		case unsubscribeMessage:

			unmsg := msg.(unsubscribeMessage)
			unstruct, ok := bucket.mySubscriptions[unmsg.h]
			if ok == true {
				unstruct.watchers.Remove(uint64(unmsg.ss.GetKey()))
				if unstruct.watchers.Size() == 0 {
					delete(bucket.mySubscriptions, unmsg.h)
				}
				topicsRemoved.Inc()
			}
			// send upstream unsubscribe

		case lookupMessage:

			lookmsg := msg.(lookupMessage)
			pubstruct, ok := bucket.mySubscriptions[lookmsg.h]
			count := uint32(0) // people watching
			if ok == false {
				// nobody watching
			} else {
				count = uint32(pubstruct.watchers.Size())
				// todo: add more info
			}
			// set count, in decimal
			str := strconv.FormatUint(uint64(count), 10)
			lookmsg.p.SetOption("count", []byte(str))
			lookmsg.ss.WriteDownstream(lookmsg.p)

		default:
			// no match. do nothing. apnic?
			fatalMessups.Inc()
		}
	}
}

// theBucketsSize is 64 for debug and 64 for prod
// it's just to keep the threads busy.
const theBucketsSize = uint(64) // uint(1024)
const theBucketsSizeLog2 = 6    // 10 // uint(math.Log2(float64(theBucketsSize)))

type subscriptionMessage struct {
	p  *packets.Subscribe
	ss ContactInterface
	h  HashType // 3*8 bytes
}

// unsubscribeMessage for real
type unsubscribeMessage struct {
	p  *packets.Unsubscribe
	ss ContactInterface
	h  HashType // 3*8 bytes
}

// publishMessage used here
type publishMessage struct {
	p  *packets.Send
	ss ContactInterface
	h  HashType // 3*8 bytes
}

type lookupMessage struct {
	p  *packets.Lookup
	ss ContactInterface
	h  HashType // 3*8 bytes
}

// watchedTopic is what we'll be collecting a lot of.
// what if *everyone* is watching this topic? and then the watchers is huge.
type watchedTopic struct {
	name     HashType // not my real name
	watchers *redblacktree.Tree
}

type subscribeBucket struct {
	mySubscriptions map[HashType]*watchedTopic
	incoming        *chan interface{} //SubscriptionMessage
	subscriber      *LookupTableStruct
}

var (
	namesAdded = promauto.NewCounter(prometheus.CounterOpts{
		Name: "look_names_added",
		Help: "The total number of subscriptions requests",
	})

	topicsAdded = promauto.NewCounter(prometheus.CounterOpts{
		Name: "look_topics_added",
		Help: "The total number new topics/subscriptions] added",
	})

	topicsRemoved = promauto.NewCounter(prometheus.CounterOpts{
		Name: "look_topics_removed",
		Help: "The total number new topics/subscriptions] deleted",
	})

	missedPushes = promauto.NewCounter(prometheus.CounterOpts{
		Name: "look_missed_pushes",
		Help: "The total number of publish to empty topic",
	})

	sentMessages = promauto.NewCounter(prometheus.CounterOpts{
		Name: "look_sent_messages",
		Help: "The total number of messages sent down",
	})

	fatalMessups = promauto.NewCounter(prometheus.CounterOpts{
		Name: "look_fatal_messages",
		Help: "The total number garbage messages",
	})
)

// NewWithInt64Comparator for HalfHash
func NewWithInt64Comparator() *redblacktree.Tree {
	return &redblacktree.Tree{Comparator: utils.UInt64Comparator}
}
