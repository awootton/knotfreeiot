package iot

import (
	types "knotfree/types"
	"strconv"
)

// theBucketsSize is the
const theBucketsSize = uint(4)
const theBucketsSizeLog2 = uint(2) // int(math.Log2(theBucketsSize))

// SubscriptionMessage for real
type subscriptionMessage struct {
	Topic        *types.HashType // not my real name
	ConnectionID *types.HashType
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

// subscription, this is private here
type subscription struct {
	name     types.HashType          // not my real name
	watchers map[types.HashType]bool // these are ID's for tcp Connection mgr
}

type subscribeBucket struct {
	mySubscriptions map[types.HashType]*subscription
	incoming        chan interface{} //SubscriptionMessage
}

// this is the whole point:
// implements SubscriptionsIntf
type pubSubManager struct {
	allTheSubscriptions []subscribeBucket
}

var psMgr pubSubManager

// GetSubscriptionsMgr returns the singleton mgr here.
func GetSubscriptionsMgr() types.SubscriptionsIntf {
	return &psMgr
}

func init() {
	psMgr = pubSubManager{}
	psMgr.allTheSubscriptions = make([]subscribeBucket, theBucketsSize)
	for i := uint(0); i < theBucketsSize; i++ {
		psMgr.allTheSubscriptions[i].mySubscriptions = make(map[types.HashType]*subscription)
		psMgr.allTheSubscriptions[i].incoming = make(chan interface{}, 256)
		go psMgr.allTheSubscriptions[i].processMessages()
	}
}

func (bucket *subscribeBucket) processMessages() {

	for {
		msg := <-bucket.incoming // wait right here
		//fmt.Println("processMessages got message " + reflect.TypeOf(msg).String()) // + string(json.Marshal(msg)))
		switch msg.(type) {

		case subscriptionMessage:
			submsg := msg.(subscriptionMessage)
			//fmt.Println("submsg.Topic " + submsg.Topic.String())
			substruct, ok := bucket.mySubscriptions[*submsg.Topic]
			if ok == false {
				substruct = &subscription{}
				substruct.name.FromHashType(submsg.Topic)
				substruct.watchers = make(map[types.HashType]bool)
				bucket.mySubscriptions[*submsg.Topic] = substruct
			}
			// this is the important part:
			// add the caller to  the set
			substruct.watchers[*submsg.ConnectionID] = true

		case publishMessage:
			pubmsg := msg.(publishMessage)
			//fmt.Println("pubmsg.Topic " + pubmsg.Topic.String())
			pubstruct, ok := bucket.mySubscriptions[*pubmsg.Topic]
			if ok == false {
				// no publish possible !
			} else {
				// pubstruct is not nil
				for key := range pubstruct.watchers {
					//fmt.Println("pubmsg.Topic " + pubmsg.Topic.String())
					if key != *pubmsg.ConnectionID {

						mmm := types.IncomingMessage{}
						mmm.Message = pubmsg.payload
						mmm.Topic = pubmsg.Topic

						_ = QueueMessageToConnection(&key, &mmm)
					}
				}
			}

		case unsubscribeMessage:

			unmsg := msg.(unsubscribeMessage)
			unstruct, ok := bucket.mySubscriptions[*unmsg.Topic]
			if ok == true {
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

// SendSubscriptionMessage will create a message object, copy pointers to it so it'll own them now, and queue the message.
func (me *pubSubManager) SendSubscriptionMessage(Topic *types.HashType, ConnectionID *types.HashType) {
	msg := subscriptionMessage{Topic, ConnectionID}
	i := Topic.GetFractionalBits(theBucketsSizeLog2)
	b := me.allTheSubscriptions[i]
	b.incoming <- msg
}

// SendUnsubscribeMessage will create a message object, copy pointers to it so it'll own them now, and queue the message.
func (me *pubSubManager) SendUnsubscribeMessage(Topic *types.HashType, ConnectionID *types.HashType) {
	msg := unsubscribeMessage{}
	msg.Topic = Topic
	msg.ConnectionID = ConnectionID
	i := Topic.GetFractionalBits(theBucketsSizeLog2)
	b := me.allTheSubscriptions[i]
	b.incoming <- msg
}

// SendPublishMessage will create a message object, copy pointers to it so it'll own them now, and queue the message.
func (me *pubSubManager) SendPublishMessage(Topic *types.HashType, ConnectionID *types.HashType, payload *[]byte) {
	msg := publishMessage{}
	msg.Topic = Topic
	msg.ConnectionID = ConnectionID
	msg.payload = payload
	i := Topic.GetFractionalBits(theBucketsSizeLog2)
	b := me.allTheSubscriptions[i]
	b.incoming <- msg
}

type subrEventsReporter struct {
}

func (collector *subrEventsReporter) report(seconds float32) []string {
	strlist := make([]string, 0, 5)
	count := 0
	for _, b := range psMgr.allTheSubscriptions {
		count += len(b.mySubscriptions)
	}
	strlist = append(strlist, "Topic count="+strconv.Itoa(count))
	return strlist
}

func init() {
	AddReporter(&subrEventsReporter{})
}
