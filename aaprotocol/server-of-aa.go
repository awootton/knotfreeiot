package aaprotocol

import (
	"errors"
	"fmt"
	"io"
	"knotfreeiot/iot"
	"knotfreeiot/iot/reporting"
	"net"
	"reflect"
)

/** Here is the protocol. Each command is a string with a preceeding length byte
	followed by a single char to indicate the type of command.

There is no echo back of successful commands. There is not a typical pub command with two arguements.
Instead there is a set topic command follolwed by a pub command.


	switch firstChar[0] {
	case 's':
		return &subscribe{str}
	case 't':
		return &setTopic{str}
	case 'p':
		return &publish{str}
	}



*/

// ServerOfAa - use the reader arch and implement aa
// returns a config to keep a handle to the sockets.
func ServerOfAa(subscribeMgr iot.PubsubIntf, addr string) *iot.SockStructConfig {

	config := iot.NewSockStructConfig(subscribeMgr)

	ServerOfAaInit(config)

	iot.ServeFactory(config, addr)

	return config
}

// ServerOfAaInit is to set default callbacks.
func ServerOfAaInit(config *iot.SockStructConfig) {

	config.SetCallback(aaServeCallback)

	servererr := func(ss *iot.SockStruct, err error) {
		aaLogThing.Collect("aa server closing")
	}
	config.SetClosecb(servererr)

	config.SetWriter(HandleTopicPayload)
}

// HandleTopicPayload writes a publish onto the  wire.
// It's also the callback the pubsub uses.
// we don't have a command with two arguments.
func HandleTopicPayload(ss *iot.SockStruct, topic []byte, payload []byte, returnAddress []byte) error {

	t := setTopic{topic}
	err := aaWrite(ss, &t)
	if err != nil {
		aaLogThing.Collect("aa bad topic write")
		return err
	}
	p := publish{payload}
	err = aaWrite(ss, &p)
	if err != nil {
		aaLogThing.Collect("aa bad write")
		return err
	}
	return nil
}

// WriteStr is is shortcut
func aaWrite(ss *iot.SockStruct, cmd aaInterface) error {
	return writeProtocolAa(ss.GetConn(), cmd.marshal())
}

// aaServeCallback is
func aaServeCallback(ss *iot.SockStruct) {

	// implement the protocol
	cmdReader := aaNewReader(ss.GetConn())
	var recentTopic []byte
	for {
		obj, err := cmdReader.aaRead()
		if err != nil {
			str := "aaRead err=" + err.Error()
			e := aaError{[]byte(str)}
			aaWrite(ss, &e)
			aaLogThing.Collect(str)
			ss.Close(err)
			return
		}
		// As much fun as it would be to make the following code into virtual methods
		// of the types involved (and I tried it) it's more annoying and harder to read
		// than just doing it all here.
		switch obj.(type) {
		case *setTopic:
			recentTopic = obj.(*setTopic).msg
		case *publish:
			payload := obj.(*publish).msg
			ss.SendPublishMessage(recentTopic, payload, []byte("unknown"))
		case *subscribe:
			ss.SendSubscriptionMessage(obj.(*subscribe).msg)
		case *unsubscribe:
			ss.SendUnsubscribeMessage(obj.(*unsubscribe).msg)
		case *ping:
			aaWrite(ss, obj)
		case *aaError:
			// client sent us an error. close.
			str := "got aaError=" + string(obj.(*aaError).msg)
			err := errors.New(str)
			aaLogThing.Collect(str)
			ss.Close(err)
			return
		default:
			// client sent us junk somehow
			str := "bad aa type=" + reflect.TypeOf(obj).String()
			err := errors.New(str)
			aaLogThing.Collect(str)
			ss.Close(err)
			return
		}
	}
}

func (me *aaReader) aaRead() (aaInterface, error) {
	str, err := readProtocolAa(me.Src)
	if err != nil {
		return nil, err
	}
	aa := unMarshalAa(str)
	return aa, nil
}

// NewReader is the local version of an object reading interface
func aaNewReader(src io.Reader) *aaReader {
	r := aaReader{}
	r.Src = src
	return &r
}

// Reader is
type aaReader struct {
	Src io.Reader
}

var emptyBytes = make([]byte, 0)

// readProtocolAstr will block trying to get a string until the conn times out.
func readProtocolAa(conn io.Reader) ([]byte, error) {

	ch := []byte{'a'}
	n, err := conn.Read(ch)
	if err != nil || n != 1 {
		return emptyBytes, err
	}
	msglen := int(ch[0]) & 0x00FF
	pos := 0
	buffer := make([]byte, msglen)
	for pos < msglen {
		n, err = conn.Read(buffer[pos:msglen])
		if err != nil {
			return emptyBytes, err
		}
		pos += n
	}
	return buffer, nil
}

// writeProtocolAaStr writes our lame protocol to the conn
// and blocks if the tcp write buffers are full.
func writeProtocolAa(conn net.Conn, str []byte) error {

	amount := len(str)
	if amount > 255 {
		aaLogThing.Collect("WriteProtocolAa string too long")
		amount = 255
	}
	strbytes := str[0:amount]

	prefix := []byte{byte(amount)}
	n, err := conn.Write(prefix)
	if err != nil {
		return err
	}
	if n != 1 {
		return errors.New("aa expect n==1")
	}
	n, err = conn.Write(strbytes)
	if err != nil {
		return err
	}
	if n != amount {
		return fmt.Errorf("expected %v not %v ", len(strbytes), n)
	}
	return nil
}

type aaInterface interface {
	marshal() []byte
	// execute would implement the server side functionality
	//execute() error
}

// fixme add error
func unMarshalAa(src []byte) aaInterface {
	firstChar := src[:1]
	msg := src[1:]
	switch firstChar[0] {
	case 's':
		return &subscribe{msg}
	case 't':
		return &setTopic{msg}
	case 'p':
		return &publish{msg}
	}
	return &ping{}
}

// setTopic implements aaInterface
type setTopic struct {
	msg []byte
}

func (me *setTopic) marshal() []byte {
	return append([]byte{'t'}, me.msg...)
}

// publish implements aaInterface
type publish struct {
	msg []byte
}

func (me *publish) marshal() []byte {
	return append([]byte{'p'}, me.msg...)
}

// subscribe implements aaInterface
type subscribe struct {
	msg []byte
}

func (me *subscribe) marshal() []byte {
	return append([]byte{'s'}, me.msg...)
}

// unsubscribe is 'u' aaInterface
type unsubscribe struct {
	msg []byte
}

func (me *unsubscribe) marshal() []byte {
	return append([]byte{'u'}, me.msg...)
}

// aaError aaInterface
type aaError struct {
	msg []byte
}

func (me *aaError) marshal() []byte {
	return append([]byte{'e'}, me.msg...)
}

// ping is 'g'. aaInterface
type ping struct {
	msg []byte
}

func (me *ping) marshal() []byte {
	return append([]byte{'g'}, me.msg...)
}

var aaLogThing = reporting.NewStringEventAccumulator(16)
