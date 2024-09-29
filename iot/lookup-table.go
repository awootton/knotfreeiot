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
	"errors"
	"fmt"
	"os"
	"reflect"
	"sync"
	"time"

	"github.com/awootton/knotfreeiot/packets"
	"github.com/emirpasic/gods/trees/redblacktree"
	"github.com/emirpasic/gods/utils"
	"github.com/prometheus/client_golang/prometheus"
	"go.mongodb.org/mongo-driver/bson"
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

type MyRedblacktree struct {
	tree redblacktree.Tree
}

// watchedTopic is what we'll be collecting a lot of. Aka, a subscription. Aka a name.
// what if *everyone* is watching this topic? and then the watchers.thetree is huge.
// these normally time out. See the heartbeat
// Keep this in sync with the ts in knotfree-net-homepage
type WatchedTopic struct {
	//
	// not my real name, the hash name
	Name HashType `bson:"name,omitempty" json:"name,omitempty"`

	// the real name, if known
	NameStr string `bson:"namestr,omitempty" json:"namestr,omitempty"`

	Expires uint32 `bson:"exp,omitempty" json:"exp,omitempty"`

	// todo: ttl ?

	// like map[uint64]*watcherItem
	thetree *redblacktree.Tree // of uint64 to watcherItem

	// like map[string]string
	OptionalKeyValues *MyRedblacktree `bson:"opt,omitempty" json:"opt,omitempty"` // might be nil if no options.

	Bill *BillingAccumulator `bson:"bill,omitempty" json:"bill,omitempty"` // might be nil if no billing. used by billing.

	nextBillingTime uint32
	lastBillingTime uint32
	Jwtid           string `bson:"jwtid,omitempty" json:"jwtid,omitempty"` // aka billkey is the id fronm the auth token

	// presense of Pubk implies this Permanent bool `bson:"perm,omitempty"`   // keep it around always, until it expires.
	// presense of Users enforces this Single    bool `bson:"simgle,omitempty"` // just the one subscriber

	// OwnedBroadcast means that this is a broadcast channel. All posts must be signed by an owner.
	OwnedBroadcast bool `bson:"owned,omitempty" json:"owned,omitempty"` // only one client allowed to post to this channel

	// Owners []string `bson:"own,omitempty"`   // the public key of the owners who have permission to make changes.
	Owner string   `bson:"own" json:"own"`                         // the public key of the owners who have permission to make changes.
	Users []string `bson:"users,omitempty" json:"users,omitempty"` // the public key of things that can subscribe to this topic. None means anyone.
}

type watcherItem struct {
	contactInterface ContactInterface
	// When someone publishes to a topic they are also subscribed to, do they get a copy back?
	// No, that's an abnormal case. The default is false.
	// if they want this they have to explicitly ask for it.
	// it creates multiple replies.
	pub2self bool // if true then publish back to caller if subscribed. The default is false everywhere else.
}

// PushUp is to send msg up to guruness. has a q per contact.
// this is called directly by the pub/sub/look commands.
// getting an error here is kinda fatal.
func (me *LookupTableStruct) PushUp(p packets.Interface, h HashType) error {

	router := me.upstreamRouter
	if router.maglev == nil {
		// some of us don't have superiors so no pushup
		// unless we have a superior cluster in which case there's
		// just the one upper channel trying to go up. FIXME: needs test.
		//
		return errors.New("no upstreamRouter")
	}
	if len(router.channels) == 0 {
		// can't pushup to no channels
		if !me.isGuru {
			fmt.Println("ERROR len(router.channels) == 0 in PushUp for ", me.myname)
		}
		return errors.New("no upstream channels")
	}
	upc := router.getUpperChannel(h.GetUint64())
	if upc != nil {

		if !upc.isRunning() || upc.founderr != nil || upc.conn == nil {
			fmt.Println("upc.running == false or founderr or conn==nil in PushUp for ", me.myname)
		}
		SpecialPrint(&packets.PacketCommon{}, func() {
			fmt.Println("upc pushing up from ", me.ex.Name, " to ", upc.name, p)
		})
		if len(upc.up) >= cap(upc.up) {
			fmt.Println("me.ex.channelToAnyAide channel full")
		}
		upc.up <- p

	} else {
		fmt.Println("where is our socket?")
		return errors.New("missing upper channel")
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
		me.allTheSubscriptions[i].mySubscriptions = make(map[HashType]*WatchedTopic, projectedTopicCount/me.theBucketsSize)
		tmp := make(chan interface{}, 256)
		me.allTheSubscriptions[i].incoming = tmp
		me.allTheSubscriptions[i].looker = me
		me.allTheSubscriptions[i].index = i
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
	p.Address.EnsureAddressIsBinary()
	msg.topicHash.InitFromBytes(p.Address.Bytes)
	i := msg.topicHash.GetFractionalBits(me.theBucketsSizeLog2) // is 4. The first 4 bits of the hash.
	b := me.allTheSubscriptions[i]
	// fmt.Println("sendSubscriptionMessage pushing #", b.index, len(b.incoming))
	if len(b.incoming) >= cap(b.incoming) {
		fmt.Println("sendSubscriptionMessage channel full", i)
		for item := range b.incoming {
			cb, ok := item.(*callBackCommand)
			if ok {
				fmt.Println("sendSubscriptionMessage callback ", cb.name)
			} else {
				fmt.Println("sendSubscriptionMessage type ", reflect.TypeOf(item))
			}
		}
	}
	b.incoming <- &msg
	// fmt.Println("sendSubscriptionMessage pushed to q")
}

// SendUnsubscribeMessage will create a message object, copy pointers to it so it'll own them now, and queue the message.
func (me *LookupTableStruct) sendUnsubscribeMessage(ss ContactInterface, p *packets.Unsubscribe) {

	msg := unsubscribeMessage{}
	msg.ss = ss
	msg.p = p
	p.Address.EnsureAddressIsBinary()
	msg.topicHash.InitFromBytes(p.Address.Bytes)
	i := msg.topicHash.GetFractionalBits(me.theBucketsSizeLog2)
	b := me.allTheSubscriptions[i]
	if len(b.incoming) >= cap(b.incoming) {
		fmt.Println("sendUnsubscribeMessage channel full")
	}
	b.incoming <- &msg
}

// SendUnsubscribeMessage will create a message object, copy pointers to it so it'll own them now, and queue the message.
func (me *LookupTableStruct) sendLookupMessage(ss ContactInterface, p *packets.Lookup) {

	msg := lookupMessage{}
	msg.ss = ss
	msg.p = p
	p.Address.EnsureAddressIsBinary()
	msg.topicHash.InitFromBytes(p.Address.Bytes)
	i := msg.topicHash.GetFractionalBits(me.theBucketsSizeLog2)
	b := me.allTheSubscriptions[i]
	if len(b.incoming) >= cap(b.incoming) {
		fmt.Println("sendLookupMessage channel full")
	}
	b.incoming <- &msg
}

// SendPublishMessage will create a message object, copy pointers to it so it'll own them now, and queue the message.
func (me *LookupTableStruct) sendPublishMessageDown(p *packets.Send) {

	msg := publishMessageDown{}
	//msg.ss = ss
	msg.p = p
	p.Address.EnsureAddressIsBinary()
	msg.h.InitFromBytes(p.Address.Bytes)
	i := msg.h.GetFractionalBits(me.theBucketsSizeLog2)
	b := me.allTheSubscriptions[i]
	if len(b.incoming) >= cap(b.incoming) {
		fmt.Println("sendPublishMessageDown channel full")
	}
	b.incoming <- &msg
}

// sendSubscriptionMessageDown is for pushing down subscriptions which are now suback's
func (me *LookupTableStruct) sendSubscriptionMessageDown(p *packets.Subscribe) {

	msg := subscriptionMessageDown{}
	//msg.ss = ss
	msg.p = p
	p.Address.EnsureAddressIsBinary()
	msg.h.InitFromBytes(p.Address.Bytes)
	i := msg.h.GetFractionalBits(me.theBucketsSizeLog2)
	b := me.allTheSubscriptions[i]
	if len(b.incoming) >= cap(b.incoming) {
		fmt.Println("sendSubscriptionMessageDown channel full")
	}
	b.incoming <- &msg
}

// SendPublishMessage will create a message object, copy pointers to it so it'll own them now, and queue the message.
func (me *LookupTableStruct) sendPublishMessage(ss ContactInterface, p *packets.Send) {

	msg := publishMessage{}
	msg.ss = ss
	msg.p = p
	p.Address.EnsureAddressIsBinary()
	msg.topicHash.InitFromBytes(p.Address.Bytes)
	i := msg.topicHash.GetFractionalBits(me.theBucketsSizeLog2)
	b := me.allTheSubscriptions[i]
	if len(b.incoming) >= cap(b.incoming) {
		fmt.Println("sendPublishMessage channel full")
	}
	b.incoming <- &msg
}

// and example of a more Modern way to do this
type countingCallBackCmd struct {
	callBackCommand
	count         int
	totalCount    int
	totalCapacity int
	qdepth        int
	mut           sync.Mutex
}

func (cb *countingCallBackCmd) Run(me *LookupTableStruct, bucket *subscribeBucket) {
	defer cb.wg.Done()
	cb.mut.Lock()
	cb.count += len(bucket.mySubscriptions)
	cb.totalCount += len(bucket.mySubscriptions)
	// TODO: check for types of subscriptions eg billing vs topis
	for _, wt := range bucket.mySubscriptions {
		_, is := wt.IsBilling()
		if is {
			cb.count--
		}
	}
	cb.qdepth += len(bucket.incoming)
	cb.totalCapacity += cap(bucket.incoming)
	defer cb.mut.Unlock()
}

// GetAllSubsCount returns the count of subscriptions and the
// average depth of the channels.
// It's not right. It needs a lock or something.
func (me *LookupTableStruct) GetAllSubsCount() (int, float64) {

	countingCB := countingCallBackCmd{}
	done := make(chan bool)

	go func() {
		fullestBucketSize := -1
		fullestBucket := -1
		for _, bucket := range me.allTheSubscriptions {
			countingCB.wg.Add(1)
			if len(bucket.incoming)*4 >= cap(bucket.incoming)*3 {
				fmt.Println("error: GetAllSubsCount bucket.incoming is full", bucket.index)
			}
			if len(bucket.incoming) > fullestBucketSize {
				fullestBucketSize = len(bucket.incoming)
				fullestBucket = bucket.index
			}
			bucket.incoming <- &countingCB
		}
		_ = fullestBucket
		// fmt.Println("GetAllSubsCount biggest bucket is ", fullestBucket, " with ", fullestBucketSize, " items")
		countingCB.wg.Wait()
		done <- true
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		fmt.Println("timeout in GetAllSubsCount ")
	}

	fract := float64(countingCB.qdepth) / float64(countingCB.totalCapacity)
	return countingCB.count, fract
}

// TODO: implement a pool of the incoming types.

func (bucket *subscribeBucket) processMessages(me *LookupTableStruct) {

	for {
		msg := <-bucket.incoming // wait right here

		// if bucket.index == 49 {
		// 	fmt.Println("have processMessages ", reflect.TypeOf(msg), "bucket", bucket.index)
		// }

		// if reflect.TypeOf(msg) != reflect.TypeOf(&callBackCommand{}) {
		// 	fmt.Println("have processMessages ", reflect.TypeOf(msg), "#", bucket.index)
		// }

		// TODO: use virtual methods or function pointers to avoid this switch.
		// eg use the CallBackInterface for everything.

		switch v := msg.(type) {

		case *subscriptionMessage:
			processSubscribe(me, bucket, v)
		case *subscriptionMessageDown:
			processSubscribeDown(me, bucket, v)
		case *lookupMessage:
			processLookup(me, bucket, v)
		case *publishMessage:
			processPublish(me, bucket, v)
		case *publishMessageDown:
			processPublishDown(me, bucket, v)

		case *unsubscribeMessage:
			processUnsubscribe(me, bucket, v)
		case CallBackInterface:
			// fmt.Println("callback in", bucket.index)
			cbc := msg.(CallBackInterface)
			cbc.Run(me, bucket)
			// fmt.Println("callback out", bucket.index)

		default:
			// no match. do nothing. panic?
			fmt.Println("ERROR processMessages missing case for ", reflect.TypeOf(msg))
			fatalMessups.Inc()
		}
	}
	// fmt.Println("FATAL processMessages exited loop")
}

type baseMessage struct {
	topicHash HashType // 3*8 bytes  // the first 24 bytes of the sha256 of the topic
	ss        ContactInterface
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
// type unsubscribeMessageDown struct {
// 	h HashType // baseMessage
// 	p *packets.Unsubscribe
// }

// publishMessage used here
type publishMessageDown struct {
	h HashType // baseMessage
	p *packets.Send
}

// me.theBucketsSize is 16 and there's 16 channels
// it's just to keep the threads busy.
//const me.theBucketsSize = uint(16)
//const me.theBucketsSizeLog2 = 4

// One map per bucket ? yes.
type subscribeBucket struct {
	// maplock         sync.Mutex
	mySubscriptions map[HashType]*WatchedTopic //[64]map[HashType]*watchedTopic
	incoming        chan interface{}
	looker          *LookupTableStruct
	index           int
}

// NewWithInt64Comparator for HalfHash
func NewWithInt64Comparator() *redblacktree.Tree {
	return &redblacktree.Tree{Comparator: utils.UInt64Comparator}
}

// A grab bag of paranoid ideas about bad states. Returns true if the contact is bad.
func (me *LookupTableStruct) checkForBadContact(badsock ContactInterface, pubstruct *WatchedTopic) bool {
	_ = pubstruct
	return badsock.IsClosed() //.GetConfig() == nil
}

func getWatcher(bucket *subscribeBucket, h *HashType) (*WatchedTopic, bool) {
	hashtable := bucket.mySubscriptions
	watcheditem, ok := hashtable[*h]
	return watcheditem, ok
}

func setWatcher(bucket *subscribeBucket, h *HashType, watcher *WatchedTopic) {

	hashtable := bucket.mySubscriptions
	if watcher != nil {
		if watcher.thetree == nil {
			watcher.thetree = NewWithInt64Comparator()
		}
		hashtable[*h] = watcher
	} else {
		delete(hashtable, *h)
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
	command.name = "heartBeatCallBack"
	command.donemap = make([]byte, me.theBucketsSize)
	isDone := make(chan bool)
	go func() {
		for _, bucket := range me.allTheSubscriptions {
			command.wg.Add(1)
			if len(bucket.incoming)*4 >= cap(bucket.incoming)*3 {
				fmt.Println("Heartbeat channel full", bucket.index)
			}
			bucket.incoming <- &command
		}
		command.wg.Wait() // should we wait?
		isDone <- true
	}()
	select {
	case <-isDone:
	case <-time.After(1 * time.Second):
		fmt.Println("LookupTableStruct Heartbeat timeout ERROR")
		for _, bucket := range me.allTheSubscriptions {
			if command.donemap[bucket.index] == 0 {
				fmt.Print(" bucket=", bucket.index, len(bucket.incoming))
			}
		}
		fmt.Println()
	}
	// fmt.Println("LookupTableStruct Heartbeat DONE")
}

// DEBUG because I don't know a better way.
// todo: look into conditional inclusion
var DEBUG = false

func init() {
	// hack alert.
	if os.Getenv("KUBE_EDITOR") == "atom --wait" {
		DEBUG = true
	}
}

// utility routines for WatchedTopic put, get etc.

// func (wt *WatchedTopic) get(key HalfHash) (*watcherItem, bool) {
// 	item, ok := wt.thetree.Get(uint64(key))
// 	if ok {
// 		item2, ok2 := item.(*watcherItem)
// 		return item2, ok2
// 	} else {
// 		return nil, false
// 	}
// }

// put is supposed to just save the item but there are doubts about doing it twice.
func (wt *WatchedTopic) put(key HalfHash, item *watcherItem) {
	wt.thetree.Put(uint64(key), item)
}

func (wt *WatchedTopic) get(key HalfHash) (*watcherItem, bool) {

	thing, ok := wt.thetree.Get(uint64(key))
	if !ok {
		return nil, ok
	}
	item, ok := thing.(*watcherItem)
	if !ok {
		fmt.Println("ERROR everything MUST be a watcherItem")
		return nil, ok
	}
	return item, ok
}

func (wt *WatchedTopic) remove(key HalfHash) {
	wt.thetree.Remove(uint64(key))
}

func (wt *WatchedTopic) removeAll() {
	wt.thetree.Clear()
}

func (wt *WatchedTopic) getSize() int {
	return wt.thetree.Size()
}

type subIterator struct {
	rbi *redblacktree.Iterator
}

func (wt *WatchedTopic) Iterator() *subIterator {
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
func (wt *WatchedTopic) OptionSize() int {
	if wt.OptionalKeyValues == nil {
		return 0
	}
	return wt.OptionalKeyValues.tree.Size()
}

// GetOption returns the value,true to go with the key or nil,false
func (wt *WatchedTopic) GetOption(key string) ([]byte, bool) {
	if wt.OptionalKeyValues == nil {
		return nil, false
	}
	var bytes []byte
	val, ok := wt.OptionalKeyValues.tree.Get(key)
	if !ok {
		bytes = []byte("")
	} else {
		bytes, ok = val.([]byte)
		if !ok {
			var str string
			str, ok = val.(string) // our bson unmarshaler is a bit loose
			if ok {
				bytes = []byte(str)
			} else {
				bytes = []byte("")
			}
		}
	}
	return bytes, ok
}

// GetOption returns the value,true to go with the key or nil,false
func (wt *WatchedTopic) IsBilling() (*BillingAccumulator, bool) {
	return wt.Bill, wt.Bill != nil
}

// DeleteOption returns the value,true to go with the key or nil,false
func (wt *WatchedTopic) DeleteOption(key string) {
	if wt.OptionalKeyValues == nil {
		return
	}
	wt.OptionalKeyValues.tree.Remove(key)
}

// ReplaceOptions replaces all the options in bulk
func (wt *WatchedTopic) ReplaceOptions(amap map[string]string) {
	if true { // !! always !!! delete everything wt.OptionalKeyValues == nil {
		tree := redblacktree.NewWithStringComparator()
		rbt := MyRedblacktree{}
		rbt.tree = *tree
		wt.OptionalKeyValues = &rbt
	}
	for k, v := range amap {
		wt.OptionalKeyValues.tree.Put(k, v)
	}
}

func (rbt *MyRedblacktree) Iterator() redblacktree.Iterator {
	return rbt.tree.Iterator()
}

func (rbt *MyRedblacktree) MarshalBSON() ([]byte, error) {

	amap := make(map[string]interface{})
	it := rbt.Iterator()
	for it.Next() {
		s, ok := it.Key().(string)
		if !ok {
			return nil, fmt.Errorf("key is not a string %v", it.Key())
		}
		val := it.Value()

		amap[s] = fmt.Sprint(val)
	}
	t, b, err := bson.MarshalValue(amap)
	_ = t
	return b, err
}

func (r *MyRedblacktree) UnmarshalBSON(data []byte) error {

	amap := make(map[string]interface{})
	err := bson.Unmarshal(data, &amap)
	if err != nil {
		return err
	}
	tree := redblacktree.NewWithStringComparator()
	r.tree = *tree
	for k, v := range amap {
		r.tree.Put(k, v)
	}
	return nil
}

func (rbt *MyRedblacktree) MarshalJSON() ([]byte, error) {

	amap := make(map[string]interface{})
	it := rbt.Iterator()
	for it.Next() {
		s, ok := it.Key().(string)
		if !ok {
			return nil, fmt.Errorf("key is not a string %v", it.Key())
		}
		val := it.Value()

		amap[s] = fmt.Sprint(val)
	}
	b, err := json.Marshal(amap)
	return b, err
}

func (r *MyRedblacktree) UnmarshalJSON(data []byte) error {

	amap := make(map[string]interface{})
	err := json.Unmarshal(data, &amap)
	if err != nil {
		return err
	}
	tree := redblacktree.NewWithStringComparator()
	r.tree = *tree
	for k, v := range amap {
		r.tree.Put(k, v)
	}
	return nil
}

// SetOption adds the key,value
func (wt *WatchedTopic) SetOption(key string, val string) {
	if wt.OptionalKeyValues == nil {
		tree := redblacktree.NewWithStringComparator()
		rbt := MyRedblacktree{}
		rbt.tree = *tree
		wt.OptionalKeyValues = &rbt // &MyRedblacktree.Tree.NewWithStringComparator()
	}
	wt.OptionalKeyValues.tree.Put(key, val)
}

// FlushMarkerAndWait puts a command into the head of *all* the q's
// and waits for *all* of them to arrive. This way we can wait. for testing.
func (me *LookupTableStruct) FlushMarkerAndWait() {

	command := callBackCommand{}
	command.name = "FlushMarkerAndWait"
	command.callback = flushMarkerCallback
	for _, bucket := range me.allTheSubscriptions {
		command.wg.Add(1)
		if len(bucket.incoming) >= cap(bucket.incoming) {
			fmt.Println("FlushMarkerAndWait channel full")
		}
		bucket.incoming <- &command
	}
	command.wg.Wait()
}
func flushMarkerCallback(me *LookupTableStruct, bucket *subscribeBucket, cmd *callBackCommand) {
	cmd.wg.Done()
}

type CallBackInterface interface {
	Run(me *LookupTableStruct, bucket *subscribeBucket)
}

// this is the generic one but one can't add fields to it.
// see countingCallBackCmd for a more modern example
type callBackCommand struct { // todo make interface
	index int
	name  string
	wg    sync.WaitGroup
	// expires  uint32
	now uint32
	// cmd is 'this' aka 'self'.
	callback func(me *LookupTableStruct, bucket *subscribeBucket, cmd *callBackCommand)
	donemap  []byte
}

func (cb *callBackCommand) Run(me *LookupTableStruct, bucket *subscribeBucket) {
	cb.callback(me, bucket, cb)
}

func guruDeleteRemappedAndGoneTopics(me *LookupTableStruct, bucket *subscribeBucket, cmd *callBackCommand) {
	// fmt.Println("guruDeleteRemappedAndGoneTopics bucket", bucket.index, len(bucket.mySubscriptions))
	//for _, s := range bucket.mySubscriptions {
	s := bucket.mySubscriptions
	for h, WatchedTopic := range s { //s {
		index := me.upstreamRouter.maglev.Lookup(h.GetUint64())
		// if the index is not me then delete the topic and tell upstream.
		if index != cmd.index {
			unsub := packets.Unsubscribe{}
			unsub.Address.Type = packets.BinaryAddress
			unsub.Address.Bytes = make([]byte, 24)
			h.GetBytes(unsub.Address.Bytes)
			me.PushUp(&unsub, h)
			delete(s, h)
		}
		_ = WatchedTopic
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
			sub := packets.Subscribe{}
			sub.Address.Type = packets.BinaryAddress
			sub.Address.Bytes = make([]byte, 24)
			h.GetBytes(sub.Address.Bytes)
			me.PushUp(&sub, h)
		}
		_ = watchedTopic
	}
}

func reSubscribeMyTopics(me *LookupTableStruct, bucket *subscribeBucket, cmd *callBackCommand) {

	defer func() {
		cmd.wg.Done()
		//fmt.Println("finished reSubscribeRemappedTopics")
	}()
	s := bucket.mySubscriptions
	for h, watchedTopic := range s {

		if me.upstreamRouter == nil {
			break
		}
		if me.upstreamRouter.maglev == nil {
			break
		}

		indexNew := me.upstreamRouter.maglev.Lookup(h.GetUint64())
		if indexNew != cmd.index {
			continue
		}
		sub := packets.Subscribe{}
		// messy sub.SetOption("debg", []byte("12345678"))
		sub.SetOption("noack", []byte("y"))
		sub.Address.Type = packets.BinaryAddress
		sub.Address.Bytes = make([]byte, 24)
		h.GetBytes(sub.Address.Bytes)
		me.PushUp(&sub, h)
		_ = watchedTopic
	}
}
