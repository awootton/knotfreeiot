package iot

import (
	"errors"
	"fmt"
	"math/rand"
	"reflect"
	"testing"
	"time"
)

// CmdInterface a
type CmdInterface interface {
}

// TCPOverPubsubCmd - delete me?
type TCPOverPubsubCmd struct {
}

// SynCommand a
type SynCommand struct {
	TCPOverPubsubCmd
	replyTopic []byte
	sequence   int
}

// SynAckCommand a
type SynAckCommand struct {
	TCPOverPubsubCmd
	sequence int
}

// xxAckCommand a
type xxAckCommand struct {
	TCPOverPubsubCmd
	//sequence int
}

// PushCommand - data is going to have a max size, like 16 or 900, and not just whatever.
type PushCommand struct {
	TCPOverPubsubCmd
	data []byte
}

// PushAckCommand -
type PushAckCommand struct {
	TCPOverPubsubCmd
	position int
}

// FinCommand a
type FinCommand struct {
	TCPOverPubsubCmd
	//sequence int
}

// FinAckCommand a
type FinAckCommand struct {
	TCPOverPubsubCmd
	//sequence int
}

func wait(c chan CmdInterface) (CmdInterface, error) {
	select {
	case obj := <-c:
		return &obj, nil
	case <-time.After(defaultTimeout * 999999999):
		return nil, errors.New("timeout")
	}
}

func waitForSyn(c chan CmdInterface) (*SynCommand, error) {
	obj, err := wait(c)
	if err != nil {
		return nil, err
	}
	syn, ok := obj.(SynCommand)
	if ok {
		return &syn, nil
	}
	return nil, errors.New("Expected syn, got:" + reflect.TypeOf(obj).String())
}

func waitForSynAck(c chan CmdInterface) (*SynAckCommand, error) {
	obj, err := wait(c)
	if err != nil {
		return nil, err
	}
	syn, ok := obj.(SynAckCommand)
	if ok {
		return &syn, nil
	}
	return nil, errors.New("Expected synack, got:" + reflect.TypeOf(obj).String())
}

// func waitForAck(c chan CmdInterface) (*AckCommand, error) {
// 	obj, err := wait(c)
// 	if err != nil {
// 		return nil, err
// 	}
// 	syn, ok := obj.(AckCommand)
// 	if ok {
// 		return &syn, nil
// 	}
// 	return nil, errors.New("Expected ack, got:" + reflect.TypeOf(obj).String())
// }

// var east = make(chan CmdInterface, 5)
// var west = make(chan CmdInterface, 5)

// DuplexChannel use NewDuplexChannel
type DuplexChannel struct {
	east chan CmdInterface
	west chan CmdInterface
}

// NewDuplexChannel -
func NewDuplexChannel(size int) DuplexChannel {
	east := make(chan CmdInterface, size)
	west := make(chan CmdInterface, size)
	return DuplexChannel{east, west}
}

// aka client eg. your phone
// in the west, sending messages eastbound.
// receiving messages from westbound
func sender(destination chan CmdInterface, replies chan CmdInterface) error {

	sequence := rand.Int()

	incomingSequence := 0 // not used here

	syn := SynCommand{sequence: sequence}
	syn.replyTopic = []byte("atwReplyTopic")
	destination <- &syn

	synack, err := waitForSynAck(replies)
	if err != nil {
		return err
	}
	incomingSequence = synack.sequence

	//ack := AckCommand{}
	//destination <- &ack

	//  now start sending

	_ = incomingSequence

	return nil
}

// aka server eg a thermostat
func receiver(destination chan CmdInterface, replies chan CmdInterface) error {

	//sequence := rand.Int() // my seq not used
	incomingSequence := 0
	var replyTopic []byte // = nil

	syn, err := waitForSyn(replies)
	if err != nil {
		return err
	}
	incomingSequence = syn.sequence

	synack := SynAckCommand{sequence: incomingSequence}
	destination <- &synack

	// go func() {
	// 	obj := <-east
	// 	fmt.Println("received", obj)

	// 	switch v := obj.(type) {
	// 	case *SynCommand:
	// 		//sss := v.(*SynCommand)
	// 		fmt.Println("syn", v)
	// 		sequence = v.sequence
	// 		west <- SynAckCommand{}
	// 	case *AckCommand:
	// 		west <- AckCommand{}
	// 	case *SynAckCommand:
	// 		//west <- AckCommand{}

	// 	default:
	// 		fmt.Printf("I don't know about type %T!\n", v)
	// 	}

	// }()

	// syn-ack (A+1,)

	// akc ack ack the received

	// fin-ack

	_ = incomingSequence
	_ = replyTopic

	return nil
}

// RunTCPOverPubsub a
func RunTCPOverPubsub() {

	fmt.Println("RunTcpOverPubsub startting")

	cd := NewDuplexChannel(5)

	go receiver(cd.east, cd.west)
	sender(cd.west, cd.east)

	for {

	}

}

// func waitForSyn2() (*SynCommand, error) {
// 	select {
// 	case obj := <-west:
// 		syn, ok := obj.(SynCommand)
// 		if ok {
// 			return &syn, nil
// 		}
// 		return nil, errors.New("Expected syn, got:" + reflect.TypeOf(obj).String())
// 	case <-time.After(defaultTimeout * 999999999):
// 		return nil, errors.New("timeout")
// 	}
// }

// TestNewDuplexChannel -
func TestNewDuplexChannel(t *testing.T) {

}
