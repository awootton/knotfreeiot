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
	"errors"
	"fmt"
	"reflect"
	"sync"

	"github.com/awootton/knotfreeiot/packets"
	"github.com/dgryski/go-maglev"
	"github.com/emirpasic/gods/trees/redblacktree"
	"github.com/emirpasic/gods/utils"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// DEBUG because I don't know a better way.
const DEBUG = true

// LookupTableStruct is what we're up to
type LookupTableStruct struct {
	//
	allTheSubscriptions []subscribeBucket

	key HashType // unique name for me.

	myname string

	isGuru bool

	upstreamRouter *UpstreamRouterStruct

	NameResolver GuruNameResolver

	config *ContactStructConfig
}

// GuruNameResolver will return an upper contact from a name
// in prod it's a DNS lookup followed by a tcp connect
// in unit test there's a global map that can be consulted. see
type GuruNameResolver func(name string, config *ContactStructConfig) (ContactInterface, error)

// UpstreamRouterStruct is maybe virtual in the future
type UpstreamRouterStruct struct {
	//
	names    []string
	contacts []ContactInterface

	maglev         *maglev.Table
	previousmaglev *maglev.Table

	name2contact map[string]common
	mux          sync.Mutex
}

// GetUpperContact returns which contact handles i
func (router *UpstreamRouterStruct) GetUpperContact(h uint64) ContactInterface {
	index := router.maglev.Lookup(h)
	if index >= len(router.contacts) {
		fmt.Println("oops")
	}
	return router.contacts[index]
}

// PushUp is. may need q per contact
func (me *LookupTableStruct) PushUp(p packets.Interface, h HashType) error {

	router := me.upstreamRouter
	if me.isGuru || router.maglev == nil {
		// some of us don't have superiors so no pushup
		return nil
	}
	cc := router.GetUpperContact(h.GetUint64())
	if cc != nil {
		cc.WriteUpstream(p)
	} else {
		fmt.Println("where is our socket?")
		return errors.New("missing upper contact")
	}
	return nil
}

// FlushMarkerAndWait puts a command into the head of all the q's
// and waits for it. This way we can wait
func (me *LookupTableStruct) FlushMarkerAndWait() {

	command := callBackCommand{}
	command.callback = flushMarkerCallback
	for _, bucket := range me.allTheSubscriptions {
		command.wg.Add(1)
		bucket.incoming <- &command
	}
	command.wg.Wait()
}
func flushMarkerCallback(me *LookupTableStruct, bucket *subscribeBucket, cmd *callBackCommand) {
	cmd.wg.Done()
}

// SetGuruUpstreamNames because the guru needs to know also
func (me *LookupTableStruct) SetGuruUpstreamNames(names []string) {

	router := me.upstreamRouter

	router.previousmaglev = router.maglev
	maglevsize := maglev.SmallM
	if DEBUG {
		maglevsize = 97
	}
	router.maglev = maglev.New(names, uint64(maglevsize))

	myindex := -1
	for i, n := range names {
		if n == me.myname {
			myindex = i
		}
	}

	// iterate all subscriptions and delete the ones that don't map here anymore.
	command := callBackCommand{}
	command.callback = guruDeleteRemappedAndGoneTopics // inline?
	command.index = myindex

	for _, bucket := range me.allTheSubscriptions {
		command.wg.Add(1)
		bucket.incoming <- &command
	}
	command.wg.Wait()
}

type callBackCommand struct { // todo make interface
	callback func(me *LookupTableStruct, bucket *subscribeBucket, cmd *callBackCommand)
	index    int
	wg       sync.WaitGroup
}

func guruDeleteRemappedAndGoneTopics(me *LookupTableStruct, bucket *subscribeBucket, cmd *callBackCommand) {
	//fmt.Println("bucket", len(bucket.incoming))
	for _, s := range bucket.mySubscriptions {
		for h, watchedTopic := range s {
			index := me.upstreamRouter.maglev.Lookup(h.GetUint64())
			// if the index is not me then delete the topic and tell upstream.
			if index != cmd.index {
				unsub := packets.Unsubscribe{}
				unsub.AddressAlias = make([]byte, 24)
				h.GetBytes(unsub.AddressAlias)
				me.PushUp(&unsub, h)
				delete(s, h)
			}
			_ = watchedTopic
		}
	}
	cmd.wg.Done()
}

// SetUpstreamNames is
func (me *LookupTableStruct) SetUpstreamNames(names []string) {

	if me.isGuru {
		me.SetGuruUpstreamNames(names)
		return
	}

	router := me.upstreamRouter

	router.contacts = make([]ContactInterface, len(names))
	router.names = make([]string, len(names))
	copy(router.names, names)

	var wg sync.WaitGroup

	for i, name := range names {
		wg.Add(1)
		go func(me *LookupTableStruct, i int, name string) {
			defer wg.Done()
			me.upstreamRouter.mux.Lock()
			com, ok := me.upstreamRouter.name2contact[name]
			var err error
			var newContact ContactInterface
			if !ok {
				newContact, err = me.NameResolver(name, me.config)
				if err != nil {
					// now what?
					fmt.Println("we cna't have this fixme", name)
					newContact = nil
					newContact, err = me.NameResolver(name, me.config)
				}
				com = common{newContact, HashType{}}
				me.upstreamRouter.name2contact[name] = com
			} else {
				newContact = com.ss
			}
			me.upstreamRouter.mux.Unlock()
			me.upstreamRouter.contacts[i] = newContact
			//fmt.Println("set", newContact)
			//me.upstreamRouter.names[i] = name

		}(me, i, name)

	}

	// wait for the contacts to fill in
	wg.Wait()

	router.previousmaglev = router.maglev
	maglevsize := maglev.SmallM
	if DEBUG {
		maglevsize = 97
	}
	router.maglev = maglev.New(names, uint64(maglevsize))
	// order subscriptions to be forwarded to the new UpContact.

	// iterate all the subscriptions and push up (again) the ones that have been remapped.
	// iterate all subscriptions and delete the ones that don't map here anymore.
	command := callBackCommand{}
	command.callback = reSubscribeRemappedTopics

	//var wg sync.WaitGroup
	for _, bucket := range me.allTheSubscriptions {
		bucket.incoming <- &command
	}

	//fmt.Println("")
	for _, cc := range router.contacts {
		if cc == nil {
			fmt.Println("fixm 88")
		}
	}
}

func reSubscribeRemappedTopics(me *LookupTableStruct, bucket *subscribeBucket, cmd *callBackCommand) {
	//fmt.Println("bucket", len(bucket.incoming))
	for _, s := range bucket.mySubscriptions {
		for h, watchedTopic := range s {
			indexNew := me.upstreamRouter.maglev.Lookup(h.GetUint64())
			indexOld := -1
			if me.upstreamRouter.previousmaglev != nil {
				indexOld = me.upstreamRouter.previousmaglev.Lookup(h.GetUint64())
			}
			// if the index has changed then push up a subscribe
			if indexNew != indexOld {
				unsub := packets.Subscribe{}
				unsub.AddressAlias = make([]byte, 24)
				h.GetBytes(unsub.AddressAlias)
				me.PushUp(&unsub, h)
			}
			_ = watchedTopic
		}
	}
}

// NewLookupTable makes a LookupTableStruct, usually a singleton.
// In the tests we call here and then use the result to init a server.
// Starts 16 go routines that are hung on their 32 deep q's
func NewLookupTable(projectedTopicCount int, aname string, isGuru bool) *LookupTableStruct {
	me := LookupTableStruct{}
	me.myname = aname
	me.isGuru = isGuru
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
	me.upstreamRouter.name2contact = make(map[string]common)
	// default is no upstream gurus. SetUpstreamNames to change that
	me.upstreamRouter.maglev = nil // maglev.New([]string{"none"}, maglev.SmallM)
	me.upstreamRouter.previousmaglev = me.upstreamRouter.maglev
	return &me
}

// sendSubscriptionMessage will create a message object, copy pointers to it so it'll own them now, and queue the message.
func (me *LookupTableStruct) sendSubscriptionMessage(ss ContactInterface, p *packets.Subscribe) {

	msg := subscriptionMessage{}
	msg.ss = ss
	msg.p = p
	msg.h.InitFromBytes(p.AddressAlias)
	i := msg.h.GetFractionalBits(theBucketsSizeLog2) // is 4. The first 4 bits of the hash.
	b := me.allTheSubscriptions[i]
	b.incoming <- &msg
}

// SendUnsubscribeMessage will create a message object, copy pointers to it so it'll own them now, and queue the message.
func (me *LookupTableStruct) sendUnsubscribeMessage(ss ContactInterface, p *packets.Unsubscribe) {

	msg := unsubscribeMessage{}
	msg.ss = ss
	msg.p = p
	msg.h.InitFromBytes(p.AddressAlias)
	i := msg.h.GetFractionalBits(theBucketsSizeLog2)
	b := me.allTheSubscriptions[i]
	b.incoming <- &msg
}

// SendUnsubscribeMessage will create a message object, copy pointers to it so it'll own them now, and queue the message.
func (me *LookupTableStruct) sendLookupMessage(ss ContactInterface, p *packets.Lookup) {

	msg := lookupMessage{}
	msg.ss = ss
	msg.p = p
	msg.h.InitFromBytes(p.AddressAlias)
	i := msg.h.GetFractionalBits(theBucketsSizeLog2)
	b := me.allTheSubscriptions[i]
	b.incoming <- &msg
}

// SendPublishMessage will create a message object, copy pointers to it so it'll own them now, and queue the message.
func (me *LookupTableStruct) sendPublishMessageDown(ss ContactInterface, p *packets.Send) {

	msg := publishMessageDown{}
	msg.ss = ss
	msg.p = p
	msg.h.InitFromBytes(p.AddressAlias)
	i := msg.h.GetFractionalBits(theBucketsSizeLog2)
	b := me.allTheSubscriptions[i]
	b.incoming <- &msg
}

// sendSubscriptionMessage will create a message object, copy pointers to it so it'll own them now, and queue the message.
func (me *LookupTableStruct) sendSubscriptionMessageDown(ss ContactInterface, p *packets.Subscribe) {

	msg := subscriptionMessageDown{}
	msg.ss = ss
	msg.p = p
	msg.h.InitFromBytes(p.AddressAlias)
	i := msg.h.GetFractionalBits(theBucketsSizeLog2)
	b := me.allTheSubscriptions[i]
	b.incoming <- &msg
}

// SendUnsubscribeMessage will create a message object, copy pointers to it so it'll own them now, and queue the message.
func (me *LookupTableStruct) sendUnsubscribeMessageDown(ss ContactInterface, p *packets.Unsubscribe) {

	msg := unsubscribeMessageDown{}
	msg.ss = ss
	msg.p = p
	msg.h.InitFromBytes(p.AddressAlias)
	i := msg.h.GetFractionalBits(theBucketsSizeLog2)
	b := me.allTheSubscriptions[i]
	b.incoming <- &msg
}

// SendUnsubscribeMessage will create a message object, copy pointers to it so it'll own them now, and queue the message.
func (me *LookupTableStruct) sendLookupMessageDown(ss ContactInterface, p *packets.Lookup) {

	msg := lookupMessageDown{}
	msg.ss = ss
	msg.p = p
	msg.h.InitFromBytes(p.AddressAlias)
	i := msg.h.GetFractionalBits(theBucketsSizeLog2)
	b := me.allTheSubscriptions[i]
	b.incoming <- &msg
}

// SendPublishMessage will create a message object, copy pointers to it so it'll own them now, and queue the message.
func (me *LookupTableStruct) sendPublishMessage(ss ContactInterface, p *packets.Send) {

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
func (me *LookupTableStruct) GetAllSubsCount() (int, float32) {
	count := 0
	totalCapacity := 0
	qdepth := 0
	for _, bucket := range me.allTheSubscriptions {
		for _, htable := range bucket.mySubscriptions {
			count += len(htable)
		}
		qdepth += len(bucket.incoming)
		totalCapacity += cap(bucket.incoming)
	}
	fract := float32(qdepth) / float32(totalCapacity)
	return count, fract
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

		case *callBackCommand:
			cbc := msg.(*callBackCommand)
			cbc.callback(me, bucket, cbc)

		default:
			// no match. do nothing. apnic?
			fmt.Println("FIXME missing case for ", reflect.TypeOf(msg))
			fatalMessups.Inc()
		}
	}
}

type common struct {
	ss ContactInterface
	h  HashType // 3*8 bytes
	// lookup has a time getter timestamp uint32   // timestamp
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
	incoming        chan interface{}
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
