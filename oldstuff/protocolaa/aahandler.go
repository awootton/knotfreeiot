// Copyright 2019 Alan Tracey Wootton

package protocolaa

import (
	"errors"
	"io"
	"knotfreeiot/oldstuff/types"
	"net"
	"reflect"
	"time"
)

// AaServeCallback is in iot2  now.
// func AaServeCallback(ss *types.SockStruct) {

// 	// implement the protocol
// 	cmdReader := NewReader(ss.GetConn())
// 	for {
// 		obj, err := cmdReader.Read()
// 		if err != nil {
// 			// say back something snarky?
// 			// we don't have a write channel
// 			// the client might want to know!
// 			ee := Error{}
// 			ee.Msg = err.Error()
// 			//ss.GetConn().Write() fixme
// 			ss.Close(err)
// 			return
// 		}
// 		aaobj := obj.(aaInterface)
// 		fmt.Println("received obj", aaobj.marshal())
// 		// execute the obj I think.
// 		// but not here. let's
// 		// keep the stack frames here smaller
// 	}
// }

// ProtocolAa is a lame pub/sub protocol with a length byte followed by a string of len 0 to 255.
// The first char in the string
// indicates which type of message we're getting so a zero string is going to be an error.

// Push into 'west' chan headed for the east. Used by clients and not the server
func (me *Handler) Push(cmd interface{}) error {
	tmp := cmd
	aathing, ok := tmp.(*aaInterface)
	if !ok {
		return errors.New("expected aaInterface{} got " + reflect.TypeOf(cmd).String())
	}
	select {
	case me.wire.west <- *aathing:
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

func newAaDuplexChannel(capacity int, conn *net.TCPConn) *aaDuplexChannel {
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

	return &adc
}

// fixme add error
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

// Reader is
// type Reader struct {
// 	Src io.Reader
// }

// func (me *Reader) Read() (interface{}, error) {
// 	str, err := readProtocolAstr(me.Src)
// 	if err != nil {
// 		return nil, err
// 	}
// 	aa := unMarshalAa(str[:1], str[1:])
// 	return aa, nil
// }

// // NewReader is the local version of an object reading interface
// func NewReader(src io.Reader) *Reader {
// 	r := Reader{}
// 	r.Src = src
// 	return &r
// }

// readProtocolAstr will block trying to get a string until the conn times out.
func readProtocolAstr(conn io.Reader) (string, error) {

	ch := []byte{'a'}
	n, err := conn.Read(ch)
	if err != nil {
		if err == io.EOF {
			//os.Exit(0)
		}
		return "", err
	}
	msglen := int(ch[0]) & 0x00FF
	pos := 0
	buffer := make([]byte, msglen)
	for pos < msglen {
		n, err = conn.Read(buffer[pos:msglen])
		if err != nil {
			if err == io.EOF {
				//os.Exit(0)
			}
			return "", err
		}
		pos += n
	}
	return string(buffer), nil
}

//WriteStr is
// func WriteStr(conn net.Conn, str string) error {
// 	return writeProtocolAaStr(conn, str)
// }

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
