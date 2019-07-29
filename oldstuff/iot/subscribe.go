// Copyright 2019 Alan Tracey Wootton

package iot

import (
	"fmt"
	types "knotfree/oldstuff/types"
)

// NewPubsubManager makes a SubscriptionsIntf, usually a singleton.
// In the tests we call here and then use the result to init some Connenctions.
func NewPubsubManager(amt int) types.SubscriptionsIntf {
	psMgr := pubSubManager{}
	portion := amt / int(theBucketsSize)
	portion2 := amt >> theBucketsSizeLog2 // we can init the hash maps big
	if portion != portion2 {
		fmt.Printf("theBucketsSizeLog2 != uint(math.Log2(float64(theBucketsSize)))")
	}
	psMgr.allTheSubscriptions = make([]subscribeBucket, theBucketsSize)
	for i := uint(0); i < theBucketsSize; i++ {
		psMgr.allTheSubscriptions[i].mySubscriptions = make(map[types.HashType]*subscription, portion)
		tmp := make(chan interface{}, 32)
		psMgr.allTheSubscriptions[i].incoming = &tmp
		go psMgr.allTheSubscriptions[i].processMessages()
	}
	return &psMgr
}

// SendSubscriptionMessage will create a message object, copy pointers to it so it'll own them now, and queue the message.
func (me *pubSubManager) SendSubscriptionMessage(topic *types.HashType, realName string, c types.ConnectionIntf, ss *types.SockStruct) {
	c.SetRealTopicName(topic, realName)
	msg := subscriptionMessage{}
	msg.ConnectionID = c.GetKey()
	msg.Topic = topic
	msg.ss = ss
	i := topic.GetFractionalBits(theBucketsSizeLog2)
	b := me.allTheSubscriptions[i]
	*b.incoming <- msg
}

// SendUnsubscribeMessage will create a message object, copy pointers to it so it'll own them now, and queue the message.
func (me *pubSubManager) SendUnsubscribeMessage(topic *types.HashType, c types.ConnectionIntf) {
	msg := unsubscribeMessage{}
	msg.Topic = topic
	msg.ConnectionID = c.GetKey()
	i := topic.GetFractionalBits(theBucketsSizeLog2)
	b := me.allTheSubscriptions[i]
	*b.incoming <- msg
}

// SendPublishMessage will create a message object, copy pointers to it so it'll own them now, and queue the message.
func (me *pubSubManager) SendPublishMessage(Topic *types.HashType, c types.ConnectionIntf, payload *[]byte) {
	msg := publishMessage{}
	msg.Topic = Topic
	msg.ConnectionID = c.GetKey()
	msg.payload = payload
	i := Topic.GetFractionalBits(theBucketsSizeLog2)
	b := me.allTheSubscriptions[i]
	*b.incoming <- msg
}

func (bucket *subscribeBucket) processMessages() {

	for {
		msg := <-*bucket.incoming // wait right here
		switch msg.(type) {

		case subscriptionMessage:
			submsg := msg.(subscriptionMessage)
			substruct := bucket.mySubscriptions[*submsg.Topic]
			if substruct == nil {
				substruct = &subscription{}
				substruct.name.FromHashType(submsg.Topic)
				substruct.watchers = make(map[types.HashType]*fan, 0)
				bucket.mySubscriptions[*submsg.Topic] = substruct
				subscribeEvents.Collect("new subscription")
			}
			// this is the important part:
			// add the caller to  the set
			newfan := fan{}
			newfan.key = submsg.ConnectionID
			newfan.ss = submsg.ss
			substruct.watchers[*submsg.ConnectionID] = &newfan

		case publishMessage:
			pubmsg := msg.(publishMessage)
			pubstruct, ok := bucket.mySubscriptions[*pubmsg.Topic]
			if ok == false {
				// no publish possible !
			} else {
				// pubstruct is not nil
				for key := range pubstruct.watchers {
					if key != *pubmsg.ConnectionID {

						mmm := types.IncomingMessage{}
						mmm.Message = pubmsg.payload
						mmm.Topic = pubmsg.Topic

						if !ConnectionExists(&key) {
							subscribeEvents.Collect("lost conn deleted")
							delete(pubstruct.watchers, key)
						} else {
							QueueMessageToConnection(&key, &mmm)
						}

					}
				}
			}

		case unsubscribeMessage:

			unmsg := msg.(unsubscribeMessage)
			unstruct, ok := bucket.mySubscriptions[*unmsg.Topic]
			if ok == true {
				subscribeEvents.Collect("unsubscribe")
				delete(unstruct.watchers, *unmsg.ConnectionID)
				if len(unstruct.watchers) == 0 {
					// forget the entire topic
					delete(bucket.mySubscriptions, *unmsg.Topic)
				}
			}

		default:
			// no match. do nothing
		}

		_ = msg

	}

}

// var subscrFRepofrtFunct = func(seconds float32) []string {
// 	strlist := make([]string, 0, 5)
// 	count := 0
// 	for _, b := range psMgr.allTheSubscriptions {
// 		count += len(b.mySubscriptions)
// 	}
// 	strlist = append(strlist, "Topic count="+strconv.Itoa(count))
// 	return strlist
// }

var subscribeEvents *types.StringEventAccumulator

func init() {
	subscribeEvents = types.NewStringEventAccumulator(12)
	subscribeEvents.SetQuiet(true)
	//types.NewGenericEventAccumulator(subscrFRepofrtFunct)
}

// theBucketsSize is 4 for debug and 1024 for prod
const theBucketsSize = uint(8) // uint(1024)
const theBucketsSizeLog2 = 3   // 10 // uint(math.Log2(float64(theBucketsSize)))

type subscriptionMessage struct {
	Topic        *types.HashType // not my real name
	ConnectionID *types.HashType
	ss           *types.SockStruct
}

// unsubscribeMessage for real
type unsubscribeMessage struct {
	subscriptionMessage
}

// publishMessage used here
type publishMessage struct {
	subscriptionMessage
	payload *[]byte
}

type fan struct {
	key *types.HashType
	ss  *types.SockStruct
}

// subscription, this is private here
type subscription struct {
	name     types.HashType          // not my real name
	watchers map[types.HashType]*fan // these are ID's for tcp Connection mgr
}

type subscribeBucket struct {
	mySubscriptions map[types.HashType]*subscription
	incoming        *chan interface{} //SubscriptionMessage
}

// // this is the whole point:
// // implements SubscriptionsIntf
type pubSubManager struct {
	allTheSubscriptions []subscribeBucket
}

func (me *pubSubManager) GetAllSubsCount() uint64 {
	c := uint64(0)

	for _, b := range me.allTheSubscriptions {
		c += uint64(len(b.mySubscriptions))
	}
	return c
}
