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
	"os"
	"reflect"
	"sync"
	"time"

	"github.com/awootton/knotfreeiot/packets"
	"github.com/dgryski/go-maglev"
	"github.com/emirpasic/gods/trees/redblacktree"
	"github.com/emirpasic/gods/utils"
	"github.com/prometheus/client_golang/prometheus"
)

// LookupTableStruct is what we're up to
type LookupTableStruct struct {
	//
	allTheSubscriptions []subscribeBucket

	key HashType // unique name for me.

	myname string // like a pod name

	isGuru bool

	upstreamRouter *UpstreamRouterStruct

	NameResolver GuruNameResolver // pointer to function

	config *ContactStructConfig // all the contacts share this pointer

	getTime func() uint32 // can't call time directly because of test
}

// GuruNameResolver will return an upper contact from a name
// in prod it's a DNS lookup followed by a tcp connect
// in unit test there's a global map that can be consulted.
type GuruNameResolver func(name string, config *ContactStructConfig) (ContactInterface, error)

// UpstreamRouterStruct is maybe virtual in the future
type UpstreamRouterStruct struct {
	//
	names    []string
	contacts []ContactInterface

	maglev         *maglev.Table
	previousmaglev *maglev.Table

	name2contact map[string]baseMessage
	mux          sync.Mutex
}

// watchedTopic is what we'll be collecting a lot of.
// what if *everyone* is watching this topic? and then the watchers is huge.
type watchedTopic struct {
	name    HashType // not my real name
	expires uint32
	thetree *redblacktree.Tree // of uint64 to watcherItem

	optionalKeyValues *redblacktree.Tree // might be nil if no options.

	// billing?? can we NOT use the optionalKeyValues ?
}

type watcherItem struct {
	// expires uint32 expire the contact instead
	ci ContactInterface
}

// GetUpperContact returns which contact handles i
func (router *UpstreamRouterStruct) GetUpperContact(h uint64) ContactInterface {
	index := router.maglev.Lookup(h)
	if index >= len(router.contacts) {
		fmt.Println("oops")
	}
	return router.contacts[index]
}

// PushUp is to send msg up to guruness. may need q per contact
// this is called directly by the pub/sub/look commands.
func (me *LookupTableStruct) PushUp(p packets.Interface, h HashType) error {

	router := me.upstreamRouter
	if me.isGuru || router.maglev == nil {
		// some of us don't have superiors so no pushup
		return nil
	}
	cc := router.GetUpperContact(h.GetUint64())
	if cc != nil && cc.GetConfig() != nil {
		err := cc.WriteUpstream(p)
		if err != nil {
			fmt.Println("upstream write fail needs to reattach", err)
			cc.Close(err)
		}
	} else {
		fmt.Println("where is our socket?")
		return errors.New("missing upper contact")
	}
	return nil
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

// SetGuruUpstreamNames because the guru needs to know also.
// recalc the maglev. reveal all the subs and delete the ones we wouldn't have.
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
	expires  uint32
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

// SetUpstreamNames is. the resolver will try to get a tcp connection.
func (me *LookupTableStruct) SetUpstreamNames(names []string, addresses []string) {

	router := me.upstreamRouter

	if reflect.DeepEqual(router.names, names) {
		// nothing changed
		return
	}

	if me.isGuru {
		me.SetGuruUpstreamNames(names)
		return
	}

	router.contacts = make([]ContactInterface, len(names))
	router.names = make([]string, len(names))
	copy(router.names, names)

	var wg sync.WaitGroup

	for i, name := range names {
		address := addresses[i]
		wg.Add(1)
		go func(me *LookupTableStruct, i int, name string, address string) {
			defer wg.Done()
			me.upstreamRouter.mux.Lock()
			defer me.upstreamRouter.mux.Unlock()
			com, ok := me.upstreamRouter.name2contact[name]
			var err error
			var newContact ContactInterface
			if ok {
				newContact = com.ss
				if newContact.GetConfig() == nil {
					ok = false
				}
			}
			if ok == false {
				newContact, err = me.NameResolver(address, me.config)
				counter := 0
				for err != nil {
					fmt.Println("we cna't have this fixme", address, err) // should never happen
					// now what?
					if counter > 5 {
						break
					}
					counter++
					time.Sleep(1 << counter * time.Millisecond)
					newContact = nil // try again
					newContact, err = me.NameResolver(address, me.config)
				}
				if newContact.GetConfig() == nil {
					fmt.Println("break here ss")
				}
				com = baseMessage{HashType{}, newContact}
				me.upstreamRouter.name2contact[name] = com
			} else {
				newContact = com.ss
			}

			if newContact.GetConfig() == nil {
				fmt.Println("break here ss2")
			}

			me.upstreamRouter.contacts[i] = newContact

		}(me, i, name, address)

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
func NewLookupTable(projectedTopicCount int, aname string, isGuru bool, getTime func() uint32) *LookupTableStruct {
	me := LookupTableStruct{}
	me.myname = aname
	me.isGuru = isGuru
	me.getTime = getTime
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
	me.upstreamRouter.name2contact = make(map[string]baseMessage)
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
func (me *LookupTableStruct) GetAllSubsCount() (int, float64) {
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
	fract := float64(qdepth) / float64(totalCapacity)
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
	baseMessage
	p *packets.Subscribe
}

// unsubscribeMessage for real
type unsubscribeMessageDown struct {
	baseMessage
	p *packets.Unsubscribe
}

// publishMessage used here
type publishMessageDown struct {
	baseMessage
	p *packets.Send
}

type lookupMessageDown struct {
	baseMessage
	p *packets.Lookup
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

func guruDeleteExpiredTopics(me *LookupTableStruct, bucket *subscribeBucket, cmd *callBackCommand) {
	defer cmd.wg.Done()
	// we don't delete them here and now. We q up Unsubscribe packets.
	// except the billing.
	for _, s := range bucket.mySubscriptions {
		for h, watchedItem := range s {
			expireAll := watchedItem.expires < cmd.expires
			it := watchedItem.Iterator()
			for it.Next() {
				key, item := it.KeyValue()
				//if item.expires < cmd.expires || expireAll || item.ci.GetClosed() {
				if expireAll || item.ci.GetClosed() {

					p := new(packets.Unsubscribe)
					p.AddressAlias = new([32]byte)[:]
					watchedItem.name.GetBytes(p.AddressAlias)
					me.sendUnsubscribeMessage(item.ci, p)
				}
				_ = key
			}
			_ = h
			// check if this is a billing topic
			// if it's billing then those unsubscribes get rid of it
			billingAccumulator, ok := watchedItem.GetBilling()
			if ok {
				if expireAll {
					setWatchers(bucket, &h, nil) // kill it now
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
		}
	}
}

// Heartbeat is every 10 sec. now is unix seconds.
func (me *LookupTableStruct) Heartbeat(now uint32) {

	timer := prometheus.NewTimer(heartbeatLookerDuration)
	defer timer.ObserveDuration()

	// drop and ex
	command := callBackCommand{}
	command.callback = guruDeleteExpiredTopics // inline?

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
