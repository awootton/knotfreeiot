package knotfree

import (
	types "knotfree/knotfree/types"
	"strconv"
)

type subrEventsReporter struct {
}

func (collector *subrEventsReporter) report(seconds float32) []string {
	strlist := make([]string, 0, 5)
	count := 0
	for _, b := range allTheSubscriptions {
		count += len(b.mySubscriptions)
	}
	strlist = append(strlist, "Topic count="+strconv.Itoa(count))
	return strlist
}

func init() {
	AddReporter(&subrEventsReporter{})
}

//
var sPREAD = 4

// SubscriptionMessage for real
type SubscriptionMessage struct {
	Topic        types.HashType // not my real name
	ConnectionID types.HashType
}

// UnsubscribeMessage for real
type UnsubscribeMessage struct {
	SubscriptionMessage
}

// PublishMessage used here
type PublishMessage struct {
	SubscriptionMessage
	Message []byte
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
//var allTheSubscriptions = make(map[HashType]*Subscription)

var allTheSubscriptions []subscribeBucket

func init() {
	allTheSubscriptions = make([]subscribeBucket, sPREAD)
	for i := 0; i < sPREAD; i++ {
		allTheSubscriptions[i].mySubscriptions = make(map[types.HashType]*subscription)
		allTheSubscriptions[i].incoming = make(chan interface{}, 256)
		go allTheSubscriptions[i].processMessages()
	}
}

func (bucket *subscribeBucket) processMessages() {

	for {
		msg := <-bucket.incoming // wait right here
		//fmt.Println("processMessages got message " + reflect.TypeOf(msg).String()) // + string(json.Marshal(msg)))
		switch msg.(type) {

		case SubscriptionMessage:
			submsg := msg.(SubscriptionMessage)
			//fmt.Println("submsg.Topic " + submsg.Topic.String())
			substruct, ok := bucket.mySubscriptions[submsg.Topic]
			if ok == false {
				substruct = &subscription{}
				substruct.name.FromHashType(&submsg.Topic)
				substruct.watchers = make(map[types.HashType]bool)
				bucket.mySubscriptions[submsg.Topic] = substruct
			}
			// this is the important part:
			// add the caller to  the set
			substruct.watchers[submsg.ConnectionID] = true

		case PublishMessage:
			pubmsg := msg.(PublishMessage)
			//fmt.Println("pubmsg.Topic " + pubmsg.Topic.String())
			pubstruct, ok := bucket.mySubscriptions[pubmsg.Topic]
			if ok == false {
				// no publish possible !
			} else {
				// pubstruct is not nil
				for key := range pubstruct.watchers {
					//fmt.Println("pubmsg.Topic " + pubmsg.Topic.String())
					if key != pubmsg.ConnectionID {

						mmm := types.IncomingMessage{}
						mmm.Message = pubmsg.Message

						_ = QueueMessageToConnection(&key, &mmm)
					}
				}
			}

		case UnsubscribeMessage:

			unmsg := msg.(UnsubscribeMessage)
			unstruct, ok := bucket.mySubscriptions[unmsg.Topic]
			if ok == true {
				delete(unstruct.watchers, unmsg.ConnectionID)
				if len(unstruct.watchers) == 0 {
					// forget the entire topic
					delete(bucket.mySubscriptions, unmsg.Topic)
				}
			}

		default:
			// no match. do nothing
		}

		_ = msg

	}

}

// AddSubscription entry point 1
func AddSubscription(msg SubscriptionMessage) {
	i := (int(msg.Topic[0]) << 8) | (int(msg.Topic[1]) & 0x00FF)
	i = i & (sPREAD - 1)
	b := allTheSubscriptions[i]
	b.incoming <- msg
}

// AddUnsubscribe entry point 1
func AddUnsubscribe(msg UnsubscribeMessage) {
	i := (int(msg.Topic[0]) << 8) | (int(msg.Topic[1]) & 0x00FF)
	i = i & (sPREAD - 1)
	b := allTheSubscriptions[i]
	b.incoming <- msg
}

// AddPublish entry point 1
func AddPublish(msg PublishMessage) {
	i := (int(msg.Topic[0]) << 8) | (int(msg.Topic[1]) & 0x00FF)
	i = i & (sPREAD - 1)
	b := allTheSubscriptions[i]
	b.incoming <- msg
}
