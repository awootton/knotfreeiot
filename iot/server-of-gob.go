package iot

import (
	"knotfreeiot/iot/reporting"
)

type subMessage struct {
	topic HashType
}

// ServerOfAa - use the reader arch and implement aa
// returns a config to keep a handle to the sockets.
func ServerOfAa(subscribeMgr PubsubIntf, addr string) *SockStructConfig {

	config := NewSockStructConfig(subscribeMgr)

	ServerOfGobInit(config)

	ServeFactory(config, addr)

	return config
}

// ServerOfGobInit is to set default callbacks.
func ServerOfGobInit(config *SockStructConfig) {

	//config.SetCallback(aaServeCallback)

	servererr := func(ss *SockStruct, err error) {
		gobLogThing.Collect("gob server closing")
	}
	config.SetClosecb(servererr)

	config.SetWriter(HandleTopicPayload)
}

// HandleTopicPayload writes a publish onto the  wire.
// It's also the callback the pubsub uses.
// we don't have a command with two arguments.
func HandleTopicPayload(ss *SockStruct, topic []byte, payload []byte, returnAddress []byte) error {

	_ = ss
	_ = topic
	_ = returnAddress
	// t := setTopic{topic}
	// err := aaWrite(ss, &t)
	// if err != nil {
	// 	aaLogThing.Collect("aa bad topic write")
	// 	return err
	// }
	// p := publish{payload}
	// err = aaWrite(ss, &p)
	// if err != nil {
	// 	aaLogThing.Collect("aa bad write")
	// 	return err
	// }
	return nil
}

// WriteStr is is shortcut
// func aaWrite(ss *SockStruct, cmd aaInterface) error {
// 	return writeProtocolAa(ss.GetConn(), cmd.marshal())
// }

// aaServeCallback is
// func aaServeCallback(ss *SockStruct) {

// 	// implement the protocol
// 	cmdReader := aaNewReader(ss.GetConn())
// 	var recentTopic []byte
// 	for {
// 		obj, err := cmdReader.aaRead()
// 		if err != nil {
// 			str := "aaRead err=" + err.Error()
// 			e := aaError{[]byte(str)}
// 			aaWrite(ss, &e)
// 			aaLogThing.Collect(str)
// 			ss.Close(err)
// 			return
// 		}
// 		// As much fun as it would be to make the following code into virtual methods
// 		// of the types involved (and I tried it) it's more annoying and harder to read
// 		// than just doing it all here.
// 		switch obj.(type) {
// 		case *setTopic:
// 			recentTopic = obj.(*setTopic).msg
// 		case *publish:
// 			payload := obj.(*publish).msg
// 			ss.SendPublishMessage(recentTopic, payload)
// 		case *subscribe:
// 			ss.SendSubscriptionMessage(obj.(*subscribe).msg)
// 		case *unsubscribe:
// 			ss.SendUnsubscribeMessage(obj.(*unsubscribe).msg)
// 		case *ping:
// 			aaWrite(ss, obj)
// 		case *aaError:
// 			// client sent us an error. close.
// 			str := "got aaError=" + string(obj.(*aaError).msg)
// 			err := errors.New(str)
// 			aaLogThing.Collect(str)
// 			ss.Close(err)
// 			return
// 		default:
// 			// client sent us junk somehow
// 			str := "bad aa type=" + reflect.TypeOf(obj).String()
// 			err := errors.New(str)
// 			aaLogThing.Collect(str)
// 			ss.Close(err)
// 			return
// 		}
// 	}
// }

var gobLogThing = reporting.NewStringEventAccumulator(16)
