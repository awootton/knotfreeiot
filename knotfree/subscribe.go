package knotfree

import (
	types "knotfree/knotfree/types"
)

//
var sPREAD = 4

// assign this soon.
// ar Qmessage *func(channelID *types.HashType, message *types.IncomingMessage) bool

// SubscriptionMessage for real
type SubscriptionMessage struct {
	Channel      types.HashType // not my real name
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

//Subscription comment
// actually, this is private here
type subscription struct {
	name     types.HashType          // not my real name
	watchers map[types.HashType]bool // these are ID's for tcp Connection mgr
	//	incoming chan SubscriptionMessage
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
			//fmt.Println("submsg.Channel " + submsg.Channel.String())
			substruct, ok := bucket.mySubscriptions[submsg.Channel]
			if ok == false {
				substruct = &subscription{}
				substruct.name.FromHashType(&submsg.Channel)
				substruct.watchers = make(map[types.HashType]bool)
				bucket.mySubscriptions[submsg.Channel] = substruct
			}
			// this is the important part:
			// add the caller to  the set
			substruct.watchers[submsg.ConnectionID] = true

		case PublishMessage:
			pubmsg := msg.(PublishMessage)
			//fmt.Println("pubmsg.Channel " + pubmsg.Channel.String())
			pubstruct, ok := bucket.mySubscriptions[pubmsg.Channel]
			if ok == false {
				// no publish possible !
			} else {
				// pubstruct is not nil
				for key := range pubstruct.watchers {
					//fmt.Println("pubmsg.Channel " + pubmsg.Channel.String())
					if key != pubmsg.ConnectionID {

						mmm := types.IncomingMessage{}
						mmm.Message = pubmsg.Message

						_ = QueueMessageToConnection(&key, &mmm)
					}
				}
			}

		case UnsubscribeMessage:

			unmsg := msg.(PublishMessage)
			unstruct, ok := bucket.mySubscriptions[unmsg.Channel]
			if ok == true {
				delete(unstruct.watchers, unmsg.ConnectionID)
			}

		default:
			// no match. do nothing
		}

		_ = msg

	}

}

// AddSubscription entry point 1
func AddSubscription(msg SubscriptionMessage) {
	i := (int(msg.Channel[0]) << 8) | (int(msg.Channel[1]) & 0x00FF)
	i = i & (sPREAD - 1)
	b := allTheSubscriptions[i]
	b.incoming <- msg
}

// AddUnsubscribe entry point 1
func AddUnsubscribe(msg UnsubscribeMessage) {
	i := (int(msg.Channel[0]) << 8) | (int(msg.Channel[1]) & 0x00FF)
	i = i & (sPREAD - 1)
	b := allTheSubscriptions[i]
	b.incoming <- msg
}

// AddPublish entry point 1
func AddPublish(msg PublishMessage) {
	i := (int(msg.Channel[0]) << 8) | (int(msg.Channel[1]) & 0x00FF)
	i = i & (sPREAD - 1)
	b := allTheSubscriptions[i]
	b.incoming <- msg
}
