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
	"reflect"
	"sync"

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

	key HashType // unique name for me.

	upstreamRouter *UpstreamRouterStruct

	NameResolver func(name string, config *ContactStructConfig) (ContactInterface, error)

	config *ContactStructConfig
}

// UpstreamRouterStruct is maybe virtual in the future
type UpstreamRouterStruct struct {
	contacts [1024]ContactInterface
	names    [1024]string

	name2contact map[string]ContactInterface
	mux          sync.Mutex
}

// PushUp is
func (me *LookupTableStruct) PushUp(p packets.Interface, h HashType, timestamp uint32) error {

	up := me.upstreamRouter

	i := h.GetFractionalBits(10)

	if up.contacts[i] != nil {
		up.contacts[i].WriteUpstream(p, timestamp)
	}

	return nil
}

// SetUpstreamNames is
func (me *LookupTableStruct) SetUpstreamNames(names [1024]string) {

	up := me.upstreamRouter

	for i := 0; i < 1024; i++ {
		if up.names[i] != names[i] {

			go func(me *LookupTableStruct, i int) {
				up := me.upstreamRouter

				name := names[i]

				up.mux.Lock()
				newContact, ok := up.name2contact[name]
				var err error
				if !ok {
					newContact, err = me.NameResolver(names[i], me.config)
					if err != nil {
						// now what?
						newContact = nil
					}
					up.name2contact[name] = newContact
				}
				up.mux.Unlock()

				if up.contacts[i] != nil {
					// something? close the old one?
				}
				up.contacts[i] = newContact
				up.names[i] = name

				// order subscriptions to be forwarded to the new UpContact.
			}(me, i)
		}
	}
}

// NewLookupTable makes a LookupTableStruct, usually a singleton.
// In the tests we call here and then use the result to init a server.
// Starts 16 go routines that are hung on their 32 deep q's
func NewLookupTable(projectedTopicCount int) *LookupTableStruct {
	me := LookupTableStruct{}
	me.key.Random()
	portion := projectedTopicCount / int(theBucketsSize)
	portion2 := projectedTopicCount >> theBucketsSizeLog2 // we can init the hash maps big
	if portion != portion2 {
		fmt.Println("EPIC FAIL theBucketsSizeLog2 != uint(math.Log2(float64(theBucketsSize)))")
	}
	me.allTheSubscriptions = make([]subscribeBucket, theBucketsSize)
	for i := uint(0); i < theBucketsSize; i++ {
		// mySubscriptions is an array of 64 maps
		for j := 0; j < len(me.allTheSubscriptions[i].mySubscriptions); j++ {
			me.allTheSubscriptions[i].mySubscriptions[j] = make(map[HashType]*watchedTopic, portion)
		}
		tmp := make(chan interface{}, 32)
		me.allTheSubscriptions[i].incoming = tmp
		me.allTheSubscriptions[i].looker = &me
		go me.allTheSubscriptions[i].processMessages(&me)
	}
	me.upstreamRouter = new(UpstreamRouterStruct)
	me.upstreamRouter.name2contact = make(map[string]ContactInterface)
	return &me
}

// sendSubscriptionMessage will create a message object, copy pointers to it so it'll own them now, and queue the message.
func (me *LookupTableStruct) sendSubscriptionMessage(ss ContactInterface, p *packets.Subscribe, timestamp uint32) {

	msg := subscriptionMessage{}
	msg.ss = ss
	msg.p = p
	msg.h.InitFromBytes(p.AddressAlias)
	i := msg.h.GetFractionalBits(theBucketsSizeLog2) // is 4. The first 4 bits of the hash.
	b := me.allTheSubscriptions[i]
	b.incoming <- &msg
}

// SendUnsubscribeMessage will create a message object, copy pointers to it so it'll own them now, and queue the message.
func (me *LookupTableStruct) sendUnsubscribeMessage(ss ContactInterface, p *packets.Unsubscribe, timestamp uint32) {

	msg := unsubscribeMessage{}
	msg.ss = ss
	msg.p = p
	msg.h.InitFromBytes(p.AddressAlias)
	i := msg.h.GetFractionalBits(theBucketsSizeLog2)
	b := me.allTheSubscriptions[i]
	b.incoming <- &msg
}

// SendUnsubscribeMessage will create a message object, copy pointers to it so it'll own them now, and queue the message.
func (me *LookupTableStruct) sendLookupMessage(ss ContactInterface, p *packets.Lookup, timestamp uint32) {

	msg := lookupMessage{}
	msg.ss = ss
	msg.p = p
	msg.h.InitFromBytes(p.AddressAlias)
	i := msg.h.GetFractionalBits(theBucketsSizeLog2)
	b := me.allTheSubscriptions[i]
	b.incoming <- &msg
}

// SendPublishMessage will create a message object, copy pointers to it so it'll own them now, and queue the message.
func (me *LookupTableStruct) sendPublishMessageDown(ss ContactInterface, p *packets.Send, timestamp uint32) {

	msg := publishMessageDown{}
	msg.ss = ss
	msg.p = p
	msg.h.InitFromBytes(p.AddressAlias)
	i := msg.h.GetFractionalBits(theBucketsSizeLog2)
	b := me.allTheSubscriptions[i]
	b.incoming <- &msg
}

// sendSubscriptionMessage will create a message object, copy pointers to it so it'll own them now, and queue the message.
func (me *LookupTableStruct) sendSubscriptionMessageDown(ss ContactInterface, p *packets.Subscribe, timestamp uint32) {

	msg := subscriptionMessageDown{}
	msg.ss = ss
	msg.p = p
	msg.h.InitFromBytes(p.AddressAlias)
	i := msg.h.GetFractionalBits(theBucketsSizeLog2)
	b := me.allTheSubscriptions[i]
	b.incoming <- &msg
}

// SendUnsubscribeMessage will create a message object, copy pointers to it so it'll own them now, and queue the message.
func (me *LookupTableStruct) sendUnsubscribeMessageDown(ss ContactInterface, p *packets.Unsubscribe, timestamp uint32) {

	msg := unsubscribeMessageDown{}
	msg.ss = ss
	msg.p = p
	msg.h.InitFromBytes(p.AddressAlias)
	i := msg.h.GetFractionalBits(theBucketsSizeLog2)
	b := me.allTheSubscriptions[i]
	b.incoming <- &msg
}

// SendUnsubscribeMessage will create a message object, copy pointers to it so it'll own them now, and queue the message.
func (me *LookupTableStruct) sendLookupMessageDown(ss ContactInterface, p *packets.Lookup, timestamp uint32) {

	msg := lookupMessageDown{}
	msg.ss = ss
	msg.p = p
	msg.h.InitFromBytes(p.AddressAlias)
	i := msg.h.GetFractionalBits(theBucketsSizeLog2)
	b := me.allTheSubscriptions[i]
	b.incoming <- &msg
}

// SendPublishMessage will create a message object, copy pointers to it so it'll own them now, and queue the message.
func (me *LookupTableStruct) sendPublishMessage(ss ContactInterface, p *packets.Send, timestamp uint32) {

	msg := publishMessage{}
	msg.ss = ss
	msg.p = p
	msg.h.InitFromBytes(p.AddressAlias)
	i := msg.h.GetFractionalBits(theBucketsSizeLog2)
	b := me.allTheSubscriptions[i]
	b.incoming <- &msg
}

// GetAllSubsCount returns the count of subscriptions and the
// average depth of the channels.
func (me *LookupTableStruct) GetAllSubsCount() (int, int) {
	count := 0
	qdepth := 0
	for _, b := range me.allTheSubscriptions {
		count += len(b.mySubscriptions)
		qdepth += (len(b.incoming))
	}
	qdepth = qdepth / len(me.allTheSubscriptions)
	return count, qdepth
}

// TODO: implement a pool of the incoming types.

func (bucket *subscribeBucket) processMessages(me *LookupTableStruct) {

	for {
		msg := <-bucket.incoming // wait right here
		switch msg.(type) {

		case *subscriptionMessage:
			processSubscribe(me, bucket, msg.(*subscriptionMessage))
		case *subscriptionMessageDown:
			processSubscribeDown(me, bucket, msg.(*subscriptionMessage))

		case *lookupMessage:
			processLookup(me, bucket, msg.(*lookupMessage))
		case *lookupMessageDown:
			processLookupDown(me, bucket, msg.(*lookupMessage))

		case *publishMessage:
			processPublish(me, bucket, msg.(*publishMessage))
		case *publishMessageDown:
			processPublishDown(me, bucket, msg.(*publishMessageDown))

		case *unsubscribeMessage:
			processUnsubscribe(me, bucket, msg.(*unsubscribeMessage))
		case *unsubscribeMessageDown:
			processUnsubscribeDown(me, bucket, msg.(*unsubscribeMessageDown))

		default:
			// no match. do nothing. apnic?
			fmt.Println("FIXME missing case for ", reflect.TypeOf(msg))
			fatalMessups.Inc()
		}
	}
}

type common struct {
	ss        ContactInterface
	h         HashType // 3*8 bytes
	timestamp uint32   // timestamp
}

type subscriptionMessage struct {
	common
	p *packets.Subscribe
}

// unsubscribeMessage for real
type unsubscribeMessage struct {
	common
	p *packets.Unsubscribe
}

// publishMessage used here
type publishMessage struct {
	common
	p *packets.Send
}

type lookupMessage struct {
	common
	p *packets.Lookup
}

type subscriptionMessageDown struct {
	common
	p *packets.Subscribe
}

// unsubscribeMessage for real
type unsubscribeMessageDown struct {
	common
	p *packets.Unsubscribe
}

// publishMessage used here
type publishMessageDown struct {
	common
	p *packets.Send
}

type lookupMessageDown struct {
	common
	p *packets.Lookup
}

// watchedTopic is what we'll be collecting a lot of.
// what if *everyone* is watching this topic? and then the watchers is huge.
type watchedTopic struct {
	name     HashType // not my real name
	watchers *redblacktree.Tree
}

// theBucketsSize is 16 and there's 16 channels
// it's just to keep the threads busy.
const theBucketsSize = uint(16)
const theBucketsSizeLog2 = 4

// each bucket has 64 maps so 1024 maps total
type subscribeBucket struct {
	mySubscriptions [64]map[HashType]*watchedTopic
	incoming        chan interface{} //SubscriptionMessage
	looker          *LookupTableStruct
}

var (
	namesAdded = promauto.NewCounter(prometheus.CounterOpts{
		Name: "look_names_added",
		Help: "The total number of subscriptions requests",
	})
	// TopicsAdded is
	TopicsAdded = promauto.NewCounter(prometheus.CounterOpts{
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

func getWatchers(bucket *subscribeBucket, h *HashType) (*watchedTopic, bool) {

	// the first 4 bits were used to select the bucket
	// the next 6 will select the hash table inside the bucket.
	sixbits := h.GetFractionalBits(10) & 0x3F
	hashtable := bucket.mySubscriptions[sixbits]
	watcheditem, ok := hashtable[*h]
	return watcheditem, ok
}

func setWatchers(bucket *subscribeBucket, h *HashType, watcher *watchedTopic) {

	// the first 4 bits were used to select the bucket
	// the next 6 will select the hash table inside the bucket.
	sixbits := h.GetFractionalBits(10) & 0x3F
	hashtable := bucket.mySubscriptions[sixbits]
	if watcher != nil {
		hashtable[*h] = watcher
	} else {
		delete(hashtable, *h)
	}

}
