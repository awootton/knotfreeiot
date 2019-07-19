package protocolaa

import (
	"errors"
	"knotfree/types"
	"net"
)

// Handler implements types.ProtocolHandler
type Handler struct {
	theTopic    string
	hashedTopic types.HashType
	wire        aaDuplexChannel
}

// ServerHandler is a handler for the server end.
type ServerHandler struct {
	Handler
	c             types.ConnectionIntf
	subscriptions types.SubscriptionsIntf
}

// NewHandler constructor - for test client
func NewHandler(conn *net.TCPConn) types.ProtocolHandler {
	me := ServerHandler{}
	me.wire = newAaDuplexChannel(0, conn)
	return &me
}

// NewServerHandler constructor for connections.go
func NewServerHandler(c types.ConnectionIntf, s types.SubscriptionsIntf) types.ProtocolHandler {
	me := ServerHandler{}
	me.c = c
	me.subscriptions = s
	me.wire = newAaDuplexChannel(0, c.GetTCPConn())
	return &me
}

type aaInterface interface {
	marshal() string
	// execute will implement the server side functionality
	execute(me *ServerHandler) error
}

// SetTopic implements aaInterface
type SetTopic struct {
	Msg string
}

func (me *SetTopic) marshal() string {
	return "t" + me.Msg
}

func (me *SetTopic) execute(parent *ServerHandler) error {
	parent.theTopic = me.Msg
	parent.hashedTopic.FromString(me.Msg)
	return nil
}

// Publish implements aaInterface
type Publish struct {
	Msg string
}

func (me *Publish) marshal() string {
	return "p" + me.Msg
}

func (me *Publish) execute(parent *ServerHandler) error {
	if parent.theTopic == "" {
		return errors.New("there's no topic set for the publish")
	}
	bytes := []byte(me.Msg)
	parent.subscriptions.SendPublishMessage(&parent.hashedTopic, parent.c, &bytes)
	return nil
}

// Subscribe implements aaInterface
type Subscribe struct {
	Msg string
}

func (me *Subscribe) marshal() string {
	return "s" + me.Msg
}
func (me *Subscribe) execute(parent *ServerHandler) error {
	hashedTopic := types.HashType{}
	hashedTopic.FromString(me.Msg)
	parent.subscriptions.SendSubscriptionMessage(&hashedTopic, me.Msg, parent.c)
	return nil
}

// Unsubscribe is 'u'
type Unsubscribe struct {
	Msg string
}

func (me *Unsubscribe) marshal() string {
	return "u" + me.Msg
}
func (me *Unsubscribe) execute(parent *ServerHandler) error {
	hashedTopic := types.HashType{}
	hashedTopic.FromString(me.Msg)
	parent.subscriptions.SendUnsubscribeMessage(&hashedTopic, parent.c)
	return nil
}

// Error is . For clients
type Error struct {
	Msg string
}

func (me *Error) marshal() string {
	return "e" + me.Msg
}

func (me *Error) execute(parent *ServerHandler) error {
	// why would a client send a server an error?
	return nil
}

// Ping is 'g'. For clients
type Ping struct {
	Msg string
}

func (me *Ping) marshal() string {
	return "g" + me.Msg
}

func (me *Ping) execute(parent *ServerHandler) error {
	// server does what?
	return nil
}

// PipedError happened at the socket so we push this in the pipe
// then everyone can get it.
type PipedError struct {
	Msg string
	err error
}

func (me *PipedError) marshal() string {
	return "d" + me.Msg
}

func (me *PipedError) execute(parent *ServerHandler) error {
	return me.err
}
