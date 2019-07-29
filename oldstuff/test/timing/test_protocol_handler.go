// Copyright 2019 Alan Tracey Wootton

package timing

import (
	"errors"
	"fmt"
	"knotfree/oldstuff/types"
	"time"
)

// ProtocolHandler does read and write of the various messages involved
// with an over the wire iot pub/sub protocol.
// The 'aa' protocol is an example.
// type ProtocolHandler interface {
// 	Serve() error
// 	// HandleWrite needs renaming FIXME.
// 	HandleWrite(*IncomingMessage) error

// 	// Push will Q the command and should return immediately. Used by clients
// 	Push(cmd interface{}) error
// 	// Pop will block for a timeout that could be as long as 30 minutes.
// 	// used by clients
// 	Pop(timeout time.Duration) (interface{}, error)
// }

// TestProtocolHandler is
type TestProtocolHandler struct {
	wire *MyDuplexChannel

	index int
}

// MyDuplexChannel  should be local and private. fixme:
type MyDuplexChannel struct {
	east chan string // from the wire to the connection
	west chan string // from the connection to the wire
}

// NewTestProtocolHandler cons...
func NewTestProtocolHandler(index int) types.ProtocolHandlerIntf {
	me := TestProtocolHandler{}
	wire := MyDuplexChannel{}
	wire.east = make(chan string, 2)
	wire.west = make(chan string, 2)
	me.wire = &wire
	me.index = index
	return &me
}

// Push and HandleWrite are the same. SAME. Server uses HandleWrite and Client uses Push
// Pop and Serve are the same. They block. Server uses Serve and client uses Pop
// FIXME: redesign atw

// HandleWrite is
func (me *TestProtocolHandler) HandleWrite(m *types.IncomingMessage) error {
	trace("TestProtocolHandler HandleWrite sending to west "+string(*m.Message), me.index)
	select {
	case me.wire.west <- string(*m.Message):
	case <-time.After(100 * time.Millisecond):
		return errors.New("My wr slow")
	}
	return nil
}

// Push is
func (me *TestProtocolHandler) Push(cmd interface{}) error {

	sss, ok := cmd.(string)
	trace("TestProtocolHandler Push ", sss, ok)

	select {
	case me.wire.west <- sss:
	case <-time.After(10 * time.Millisecond):
		return errors.New("My Push slow")
	}
	return nil
}

// Serve is. In the useful version of this we cast the obj coming off the wire
// and then call a method that does the work
func (me *TestProtocolHandler) Serve() error {
	trace("TestProtocolHandler Serve")
	select {
	case obj := <-me.wire.east:

		trace("we received Serve ", obj)

	case <-time.After(10 * time.Second):
		return errors.New("My Serve read slow")
	}
	return nil
}

// Pop is like Serve except it returns the object and does no work.
func (me *TestProtocolHandler) Pop(timeout time.Duration) (interface{}, error) {
	trace("in Pop  ", me.index)
	select {
	case obj := <-me.wire.east:

		trace("we received Pop ", obj)
		return obj, nil

	case <-time.After(10 * time.Second):
		return "10 sec timeout waiting for Pop", errors.New("My Pop read slow")
	}
	// return "unreachable", nil
}

const doTrace = false

func trace(a ...interface{}) {
	if doTrace {
		fmt.Println(a...)
	}
}
