// Copyright 2019 Alan Tracey Wootton

package protocolaa

import (
	"errors"
	"io"
	"knotfree/types"
	"net"
	"reflect"
	"strings"
	"time"
)

// ProtocolAa is a lame pub/sub protocol with a length byte followed by a string of len 0 to 255.
// The first char in the string
// indicates which type of message we're getting so a zero string is going to be an error.

// Push into 'west' chan headed for the east. Used by clients and not the server
func (me *Handler) Push(cmd interface{}) error {
	tmp := cmd
	aathing, ok := tmp.(aaInterface)
	if !ok {
		return errors.New("expected aaInterface{} got " + reflect.TypeOf(cmd).String())
	}
	select {
	case me.wire.west <- aathing:
	case <-time.After(10 * time.Millisecond):
		return errors.New("Aa Push slow")
	}
	return nil
}

// Pop blocks. Actually returns aaInterface. See above.
// This is used by test clients so it must check for the
// PipedError case and return the error.
func (me *Handler) Pop(timeout time.Duration) (interface{}, error) {
	select {
	case obj := <-me.wire.east:
		errObj, ok := obj.(*PipedError)
		if ok {
			return nil, errObj.err
		}
		return obj, nil
	case <-time.After(timeout):
		return nil, errors.New("Aa read too slow")
	}
}

type aaDuplexChannel struct {
	east chan aaInterface // from the wire to the connection
	west chan aaInterface // from the connection to the wire
}

var aaDefaultTimeout = 21 * time.Minute

func newAaDuplexChannel(capacity int, conn *net.TCPConn) aaDuplexChannel {
	adc := aaDuplexChannel{}
	adc.east = make(chan aaInterface, capacity)
	adc.west = make(chan aaInterface, capacity)
	// We'll put the socket in the east.
	go func() {
		for {
			str, err := readProtocolAstr(conn)
			if err != nil {
				cmd := PipedError{}
				cmd.Msg = err.Error()
				cmd.err = err
				adc.east <- &cmd
			} else {
				obj := unMarshalAa(str[:1], str[1:])
				adc.east <- obj
			}
		}
	}()

	go func() {
		for {
			obj := <-adc.west
			str := obj.marshal()
			err := writeProtocolAaStr(conn, str)
			if err != nil {
				cmd := PipedError{}
				cmd.Msg = err.Error()
				cmd.err = err
				adc.east <- &cmd
			}
		}
	}()

	return adc
}

func unMarshalAa(firstChar string, str string) aaInterface {
	switch firstChar[0] {
	case 's':
		return &Subscribe{str}
	case 't':
		return &SetTopic{str}
	case 'p':
		return &Publish{str}
	}
	return &Ping{}
}

// HandleWrite from ProtocolHandler interface
func (me *ServerHandler) HandleWrite(msg *types.IncomingMessage) error {

	realName, ok := me.c.GetRealTopicName(msg.Topic)
	if !ok {
		return errors.New("missing real name")
	}
	// TODO: optimize redundant SetTopic commands.
	select {
	case me.wire.west <- &SetTopic{realName}:
	case <-time.After(100 * time.Millisecond):
		return errors.New("Aa wr slow")
	}
	select {
	case me.wire.west <- &Publish{string(*msg.Message)}:
	case <-time.After(100 * time.Millisecond):
		return errors.New("Aa wr slow2")
	}
	return nil
}

// Serve implementing  ProtocolHandler interface
// if there was a tcp sock error then the execute of the PipedError
// will return that error to the loop in server
func (me *ServerHandler) Serve() error {
	select {
	case obj := <-me.wire.east:
		err := obj.execute(me)
		if err != nil {
			return err
		}
	case <-time.After(21 * time.Minute):
		return errors.New("Aa read slow")
	}
	return nil
}

// readProtocolAstr will block trying to get a string until the conn times out.
func readProtocolAstr(conn net.Conn) (string, error) {

	buffer := make([]byte, 256)
	ch := []byte{'a'}
	n, err := conn.Read(ch)
	if err != nil {
		if err == io.EOF {
			//os.Exit(0)
		}
		return "", err
	}
	msglen := int(ch[0]) & 0x00FF
	var sb strings.Builder
	for msglen > 0 {
		n, err = conn.Read(buffer[:msglen])
		if err != nil {
			if err == io.EOF {
				//os.Exit(0)
			}
			return "", err
		}
		s := string(buffer[:n])
		sb.WriteString(s)
		msglen -= n
	}
	return sb.String(), nil
}

// writeProtocolAaStr writes our lame protocol to the conn
func writeProtocolAaStr(conn net.Conn, str string) error {

	strbytes := []byte(str)
	if len(strbytes) > 255 {
		return errors.New("WriteProtocolA string too long")
	}
	prefix := []byte{byte(len(strbytes))}
	n, err := conn.Write(prefix)
	if err != nil {
		if err == io.EOF {
			//os.Exit(0)
		}
		return err
	}
	if n != 1 || err != nil {
		return err
	}
	n, err = conn.Write(strbytes)
	if err != nil {
		if err == io.EOF {
			//os.Exit(0)
		}
		return err
	}
	if n != len(strbytes) || err != nil {
		return err
	}
	return nil
}
