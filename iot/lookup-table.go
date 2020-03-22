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
	"errors"
	"fmt"
	"os"
	"reflect"
	"sync"

	"github.com/awootton/knotfreeiot/packets"
	"github.com/emirpasic/gods/trees/redblacktree"
	"github.com/emirpasic/gods/utils"
	"github.com/prometheus/client_golang/prometheus"
)

// LookupTableStruct is good for message routing and address lookup.
type LookupTableStruct struct {
	//
	allTheSubscriptions []subscribeBucket

	key HashType // unique name for me.

	myname string // like a pod name, for humans.

	isGuru bool

	upstreamRouter *upstreamRouterStruct

	config *ContactStructConfig // all the contacts share this pointer

	getTime func() uint32 // can't call time directly because of test

	ex *Executive

	// Becomes a 'thread' count. The count of the queues.
	theBucketsSize     int // = uint(16)
	theBucketsSizeLog2 int // = 4
}

// watchedTopic is what we'll be collecting a lot of.
// what if *everyone* is watching this topic? and then the watchers.thetree is huge.
type watchedTopic struct {
	//
	name    HashType // not my real name
	expires uint32
	thetree *redblacktree.Tree // of uint64 to watcherItem

	optionalKeyValues *redblacktree.Tree // might be nil if no options.

	// billing: can we NOT use the optionalKeyValues ?

	nextBillingTime uint32
	lastBillingTime uint32
	jwtidAlias      string
}

type watcherItem struct {
	ci ContactInterface
}

// PushUp is to send msg up to guruness. has a q per contact.
// this is called directly by the pub/sub/look commands.
// getting an error here is kinda fatal.
func (me *LookupTableStruct) PushUp(p packets.Interface, h HashType) error {

	router := me.upstreamRouter
	if me.isGuru || router.maglev == nil {
		// some of us don't have superiors so no pushup
		return nil
	}
	upc := router.getUpperChannel(h.GetUint64())
	if upc != nil {
		//fmt.Println("upc pushing up from ", me.ex.Name, " to ", upc.name, p)
		upc.up <- p

	} else {
		fmt.Println("where is our socket?")
		return errors.New("missing upper c")
	}
	return nil
}

// NewLookupTable makes a LookupTableStruct, usually a singleton.
// In the tests we call here and then use the result to init a server.
// Starts 16 go routines that are hung on their 32 deep q's
func NewLookupTable(projectedTopicCount int, aname string, isGuru bool, getTime func() uint32) *LookupTableStruct {
	me := &LookupTableStruct{}
	me.myname = aname
	me.isGuru = isGuru
	me.getTime = getTime
	me.key.Random()

	// how many threads?
	if projectedTopicCount < 1000 {
		DEBUG = true
		me.theBucketsSize = 4 // 4 threads
		me.theBucketsSizeLog2 = 2
	} else {
		me.theBucketsSize = 64 // 64 threads in prod
		me.theBucketsSizeLog2 = 6
	}

	portion := projectedTopicCount / int(me.theBucketsSize)
	portion2 := projectedTopicCount >> me.theBucketsSizeLog2 // we can init the hash maps big
	if portion != portion2 {
		fmt.Println("EPIC FAIL me.theBucketsSizeLog2 != uint(math.Log2(float64(me.theBucketsSize)))")
	}
	me.allTheSubscriptions = make([]subscribeBucket, me.theBucketsSize)
	for i := 0; i < me.theBucketsSize; i++ {
		// mySubscriptions is not an array of 64 maps
		// for j := 0; j < len(me.allTheSubscriptions[i].mySubscriptions); j++ {
		// 	me.allTheSubscriptions[i].mySubscriptions[j] = make(map[HashType]*watchedTopic, portion)
		// }
		me.allTheSubscriptions[i].mySubscriptions = make(map[HashType]*watchedTopic, projectedTopicCount/me.theBucketsSize)
		tmp := make(chan interface{}, 32)
		me.allTheSubscriptions[i].incoming = tmp
		me.allTheSubscriptions[i].looker = me
		go me.allTheSubscriptions[i].processMessages(me)
	}
	me.upstreamRouter = new(upstreamRouterStruct)
	me.upstreamRouter.name2channel = make(map[string]*upperChannel)
	// default is no upstream gurus. SetUpstreamNames to change that
	me.upstreamRouter.maglev = nil // maglev.New([]string{"none"}, maglev.SmallM)
	me.upstreamRouter.previousmaglev = me.upstreamRouter.maglev

	return me
}

// sendSubscriptionMessage will create a message object, copy pointers to it so it'll own them now, and queue the message.
func (me *LookupTableStruct) sendSubscriptionMessage(ss ContactInterface, p *packets.Subscribe) {

	msg := subscriptionMessage{} // TODO: use a pool.
	msg.ss = ss
	msg.p = p
	msg.h.InitFromBytes(p.AddressAlias)
	i := msg.h.GetFractionalBits(me.theBucketsSizeLog2) // is 4. The first 4 bits of the hash.
	b := me.allTheSubscriptions[i]
	b.incoming <- &msg
}

// SendUnsubscribeMessage will create a message object, copy pointers to it so it'll own them now, and queue the message.
func (me *LookupTableStruct) sendUnsubscribeMessage(ss ContactInterface, p *packets.Unsubscribe) {

	msg := unsubscribeMessage{}
	msg.ss = ss
	msg.p = p
	msg.h.InitFromBytes(p.AddressAlias)
	i := msg.h.GetFractionalBits(me.theBucketsSizeLog2)
	b := me.allTheSubscriptions[i]
	b.incoming <- &msg
}

// SendUnsubscribeMessage will create a message object, copy pointers to it so it'll own them now, and queue the message.
func (me *LookupTableStruct) sendLookupMessage(ss ContactInterface, p *packets.Lookup) {

	msg := lookupMessage{}
	msg.ss = ss
	msg.p = p
	msg.h.InitFromBytes(p.AddressAlias)
	i := msg.h.GetFractionalBits(me.theBucketsSizeLog2)
	b := me.allTheSubscriptions[i]
	b.incoming <- &msg
}

// SendPublishMessage will create a message object, copy pointers to it so it'll own them now, and queue the message.
func (me *LookupTableStruct) sendPublishMessageDown(p *packets.Send) {

	msg := publishMessageDown{}
	//msg.ss = ss
	msg.p = p
	msg.h.InitFromBytes(p.AddressAlias)
	i := msg.h.GetFractionalBits(me.theBucketsSizeLog2)
	b := me.allTheSubscriptions[i]
	b.incoming <- &msg
}

// sendSubscriptionMessage will create a message object, copy pointers to it so it'll own them now, and queue the message.
func (me *LookupTableStruct) sendSubscriptionMessageDown(p *packets.Subscribe) {

	msg := subscriptionMessageDown{}
	//msg.ss = ss
	msg.p = p
	msg.h.InitFromBytes(p.AddressAlias)
	i := msg.h.GetFractionalBits(me.theBucketsSizeLog2)
	b := me.allTheSubscriptions[i]
	b.incoming <- &msg
}

// SendUnsubscribeMessage will create a message object, copy pointers to it so it'll own them now, and queue the message.
func (me *LookupTableStruct) sendUnsubscribeMessageDown(p *packets.Unsubscribe) {

	msg := unsubscribeMessageDown{}
	//msg.ss = ss
	msg.p = p
	msg.h.InitFromBytes(p.AddressAlias)
	i := msg.h.GetFractionalBits(me.theBucketsSizeLog2)
	b := me.allTheSubscriptions[i]
	b.incoming <- &msg
}

// SendUnsubscribeMessage will create a message object, copy pointers to it so it'll own them now, and queue the message.
func (me *LookupTableStruct) sendLookupMessageDown(p *packets.Lookup) {

	msg := lookupMessageDown{}
	//msg.ss = ss
	msg.p = p
	msg.h.InitFromBytes(p.AddressAlias)
	i := msg.h.GetFractionalBits(me.theBucketsSizeLog2)
	b := me.allTheSubscriptions[i]
	b.incoming <- &msg
}

// SendPublishMessage will create a message object, copy pointers to it so it'll own them now, and queue the message.
func (me *LookupTableStruct) sendPublishMessage(ss ContactInterface, p *packets.Send) {

	msg := publishMessage{}
	msg.ss = ss
	msg.p = p
	msg.h.InitFromBytes(p.AddressAlias)
	i := msg.h.GetFractionalBits(me.theBucketsSizeLog2)
	b := me.allTheSubscriptions[i]
	b.incoming <- &msg
}

// GetAllSubsCount returns the count of subscriptions and the
// average depth of the channels.
func (me *LookupTableStruct) GetAllSubsCount() (int, float64) {
	count := 0
	totalCapacity := 0
	qdepth := 0
	for _, bucket := range me.allTheSubscriptions {
		count += len(bucket.mySubscriptions)
		qdepth += len(bucket.incoming)
		totalCapacity += cap(bucket.incoming)
	}
	fract := float64(qdepth) / float64(totalCapacity)
	return count, fract
}

// TODO: implement a pool of the incoming types.

func (bucket *subscribeBucket) processMessages(me *LookupTableStruct) {

	for {
		msg := <-bucket.incoming // wait right here
		switch v := msg.(type) {

		case *subscriptionMessage:
			processSubscribe(me, bucket, v)
		case *subscriptionMessageDown:
			processSubscribeDown(me, bucket, v)

		case *lookupMessage:
			processLookup(me, bucket, v)
		case *lookupMessageDown:
			processLookupDown(me, bucket, v)

		case *publishMessage:
			processPublish(me, bucket, v)
		case *publishMessageDown:
			processPublishDown(me, bucket, v)

		case *unsubscribeMessage:
			processUnsubscribe(me, bucket, v)
		case *unsubscribeMessageDown:
			processUnsubscribeDown(me, bucket, v)

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

type baseMessage struct {
	h  HashType // 3*8 bytes
	ss ContactInterface
	// lookup has a time getter timestamp uint32   // timestamp
}

type subscriptionMessage struct {
	baseMessage
	p *packets.Subscribe
}

// unsubscribeMessage for real
type unsubscribeMessage struct {
	baseMessage
	p *packets.Unsubscribe
}

// publishMessage used here
type publishMessage struct {
	baseMessage
	p *packets.Send
}

type lookupMessage struct {
	baseMessage
	p *packets.Lookup
}

type subscriptionMessageDown struct {
	h HashType //baseMessage
	p *packets.Subscribe
}

// unsubscribeMessage for real
type unsubscribeMessageDown struct {
	h HashType // baseMessage
	p *packets.Unsubscribe
}

// publishMessage used here
type publishMessageDown struct {
	h HashType // baseMessage
	p *packets.Send
}

type lookupMessageDown struct {
	h HashType // baseMessage
	p *packets.Lookup
}

// me.theBucketsSize is 16 and there's 16 channels
// it's just to keep the threads busy.
//const me.theBucketsSize = uint(16)
//const me.theBucketsSizeLog2 = 4

// One map per bucket ? yes.
type subscribeBucket struct {
	mySubscriptions map[HashType]*watchedTopic //[64]map[HashType]*watchedTopic
	incoming        chan interface{}
	looker          *LookupTableStruct
}

// NewWithInt64Comparator for HalfHash
func NewWithInt64Comparator() *redblacktree.Tree {
	return &redblacktree.Tree{Comparator: utils.UInt64Comparator}
}

// A grab bag of paranoid ideas about bad states.
func (me *LookupTableStruct) checkForBadContact(badsock ContactInterface, pubstruct *watchedTopic) bool {

	if badsock.GetConfig() == nil {
		return true
	}
	return false
}

func getWatcher(bucket *subscribeBucket, h *HashType) (*watchedTopic, bool) {
	hashtable := bucket.mySubscriptions
	watcheditem, ok := hashtable[*h]
	return watcheditem, ok
}

func setWatcher(bucket *subscribeBucket, h *HashType, watcher *watchedTopic) {

	hashtable := bucket.mySubscriptions
	if watcher != nil {
		hashtable[*h] = watcher
	} else {
		delete(hashtable, *h)
	}
}

func heartBeatCallBack(me *LookupTableStruct, bucket *subscribeBucket, cmd *callBackCommand) {
	defer cmd.wg.Done()
	// we don't delete them here and now. We q up Unsubscribe packets.
	// except the billing.
	s := bucket.mySubscriptions
	for h, watchedItem := range s {
		expireAll := watchedItem.expires < cmd.expires
		// first, scan all the contact references and schedule the stale ones for deleteion.
		it := watchedItem.Iterator()
		for it.Next() {
			key, item := it.KeyValue()
			//if item.expires < cmd.expires || expireAll || item.ci.GetClosed() {
			if expireAll || item.ci.GetClosed() {

				p := new(packets.Unsubscribe)
				p.AddressAlias = new([24]byte)[:]
				watchedItem.name.GetBytes(p.AddressAlias)
				me.sendUnsubscribeMessage(item.ci, p)
			}
			_ = key
		}
		_ = h
		// second, check if this is a billing topic
		// if it's billing and it's over limits then write 'error Send' down.
		billingAccumulator, ok := watchedItem.GetBilling()
		if ok {
			if expireAll {
				setWatcher(bucket, &h, nil) // kill it now
			} else {
				good, msg := billingAccumulator.AreUnderMax(me.getTime())
				if !good {
					p := &packets.Send{}
					p.AddressAlias = new([24]byte)[:]
					h.GetBytes(p.AddressAlias)
					p.Payload = []byte(msg)
					p.SetOption("error", p.Payload)
					// just like a publish down.
					it = watchedItem.Iterator()
					for it.Next() {
						key, item := it.KeyValue()
						_ = key
						ci := item.ci
						if me.checkForBadContact(ci, watchedItem) == false {
							ci.WriteDownstream(p)
						}
					}
				}
			}
		}
		// third, we'll need to send out the topic usage-stats occasionally.
		if len(watchedItem.jwtidAlias) == 32 {

			if watchedItem.nextBillingTime < cmd.now {

				deltaTime := cmd.now - watchedItem.lastBillingTime
				watchedItem.lastBillingTime = cmd.now
				watchedItem.nextBillingTime = cmd.now + 300 // 300 secs after first time

				msg := &StatsWithTime{}
				msg.Start = cmd.now

				msg.Subscriptions = float32(deltaTime) // means one per sec, one per min ...
				p := &packets.Send{}
				p.AddressAlias = []byte(watchedItem.jwtidAlias)
				str, err := json.Marshal(msg)
				if err != nil {
					fmt.Println(" break fast ")
				}
				p.SetOption("stats", str)
				// send somewhere
				// we need some kind of pipe to the cluster front.
				me.ex.channelToAnyAide <- p
			}
		}
	}
}

// Heartbeat is every 10 sec. now is unix seconds.
func (me *LookupTableStruct) Heartbeat(now uint32) {

	timer := prometheus.NewTimer(heartbeatLookerDuration)
	defer timer.ObserveDuration()

	// drop and ex
	command := callBackCommand{}
	command.callback = heartBeatCallBack
	command.now = now

	for _, bucket := range me.allTheSubscriptions {
		command.wg.Add(1)
		bucket.incoming <- &command
	}
	command.wg.Wait() // should we wait?
}

// DEBUG because I don't know a better way.
// todo: look into conditional inclusion
var DEBUG = false

func init() {
	if os.Getenv("KUBE_EDITOR") == "atom --wait" {
		DEBUG = true
	}
}

// utility routines for watchedTopic put, get etc.

func (wt *watchedTopic) put(key HalfHash, ci ContactInterface) {
	item := new(watcherItem)
	item.ci = ci
	//item.expires = 20 * 60 * ci.GetConfig().GetLookup().getTime()
	wt.thetree.Put(uint64(key), item)
}

func (wt *watchedTopic) remove(key HalfHash) {
	wt.thetree.Remove(uint64(key))
}

func (wt *watchedTopic) getSize() int {
	return wt.thetree.Size()
}

type subIterator struct {
	rbi *redblacktree.Iterator
}

func (wt *watchedTopic) Iterator() *subIterator {
	si := new(subIterator)
	rbi := wt.thetree.Iterator()
	si.rbi = &rbi
	return si
}

func (it *subIterator) Next() bool {
	rbit := it.rbi
	return rbit.Next()
}

func (it *subIterator) KeyValue() (HalfHash, *watcherItem) {
	rbit := it.rbi
	tmp, ok := rbit.Key().(uint64)
	if !ok {
		panic("expect key to be uint64")
	}
	key := HalfHash(tmp)
	ss, ok := rbit.Value().(*watcherItem)
	if !ok {
		panic("expect val to be watcherItem")
	}
	return key, ss
}

// utility routines for watchedTopic options
// OptionSize returns key count which is same as value count
func (wt *watchedTopic) OptionSize() int {
	if wt.optionalKeyValues == nil {
		return 0
	}
	return wt.optionalKeyValues.Size()
}

// GetOption returns the value,true to go with the key or nil,false
func (wt *watchedTopic) GetOption(key string) ([]byte, bool) {
	if wt.optionalKeyValues == nil {
		return nil, false
	}
	var bytes []byte
	val, ok := wt.optionalKeyValues.Get(key)
	if !ok {
		bytes = []byte("")
	} else {
		bytes, ok = val.([]byte)
		if !ok {
			bytes = []byte("")
		}
	}
	return bytes, ok
}

// GetOption returns the value,true to go with the key or nil,false
func (wt *watchedTopic) GetBilling() (*BillingAccumulator, bool) {
	if wt.optionalKeyValues == nil {
		return nil, false
	}
	val, ok := wt.optionalKeyValues.Get("bill")
	if !ok {
		return nil, ok
	}
	stats, ok := val.(*BillingAccumulator)
	if !ok {
		return nil, ok
	}
	return stats, ok
}

// DeleteOption returns the value,true to go with the key or nil,false
func (wt *watchedTopic) DeleteOption(key string) {
	if wt.optionalKeyValues == nil {
		return
	}
	wt.optionalKeyValues.Remove(key)

}

// SetOption adds the key,value
func (wt *watchedTopic) SetOption(key string, val interface{}) {
	if wt.optionalKeyValues == nil {
		wt.optionalKeyValues = redblacktree.NewWithStringComparator()
	}
	wt.optionalKeyValues.Put(key, val)
}

// FlushMarkerAndWait puts a command into the head of *all* the q's
// and waits for *all* of them to arrive. This way we can wait. for testing.
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

type callBackCommand struct { // todo make interface
	callback func(me *LookupTableStruct, bucket *subscribeBucket, cmd *callBackCommand)
	index    int
	wg       sync.WaitGroup
	expires  uint32
	now      uint32
}

func guruDeleteRemappedAndGoneTopics(me *LookupTableStruct, bucket *subscribeBucket, cmd *callBackCommand) {
	//fmt.Println("bucket", len(bucket.incoming))
	//for _, s := range bucket.mySubscriptions {
	s := bucket.mySubscriptions
	for h, watchedTopic := range s { //s {
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
		//	}
	}
	cmd.wg.Done()
}

func reSubscribeRemappedTopics(me *LookupTableStruct, bucket *subscribeBucket, cmd *callBackCommand) {

	defer func() {
		cmd.wg.Done()
		//fmt.Println("finished reSubscribeRemappedTopics")
	}()
	s := bucket.mySubscriptions
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
