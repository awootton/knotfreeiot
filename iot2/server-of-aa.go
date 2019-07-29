package iot2

import (
	"errors"
	"fmt"
	"io"
	"net"
)

// ServerOfAa - use the reader arch and implement aa
// returns but
func ServerOfAa(subscribeMgr PubsubIntf, addr string) *SockStructConfig {

	config := NewSockStructConfig(subscribeMgr)

	config.SetCallback(aaServeCallback) // below
	servererr := func(ss *SockStruct, err error) {
		fmt.Println("server is closing", err)

	}
	config.SetClosecb(servererr)

	ServeFactory(config, addr)

	return config
}

// aaServeCallback is
func aaServeCallback(ss *SockStruct) {

	// implement the protocol
	cmdReader := aaNewReader(ss.GetConn())
	for {
		obj, err := cmdReader.aaRead()
		if err != nil {
			// say back something snarky?
			// we don't have a write channel
			// the client might want to know!
			ee := aaError{}
			ee.Msg = err.Error()
			//ss.GetConn().Write() fixme
			ss.Close(err)
			return
		}
		aaobj := obj.(aaInterface)
		fmt.Println("received obj", aaobj.marshal())
		// execute the obj I think.
		// but not here. let's
		// keep the stack frames here smaller
	}
}

func (me *aaReader) aaRead() (aaInterface, error) {
	str, err := readProtocolAstr(me.Src)
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
func aaWriteStr(conn net.Conn, str string) error {
	return writeProtocolAaStr(conn, str)
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
		return err
	}
	if n != 1 {
		return errors.New("Expect n==1")
	}
	n, err = conn.Write(strbytes)
	if err != nil {
		return err
	}
	if n != len(strbytes) {
		return fmt.Errorf("Expected %v not %v ", len(strbytes), n)
	}
	return nil
}

type aaInterface interface {
	marshal() string
	// execute will implement the server side functionality
	//execute(me *ServerHandler) error
}

// fixme add error
func unMarshalAa(src string) aaInterface {
	firstChar := src[:1]
	str := src[1:]
	switch firstChar[0] {
	case 's':
		return &subscribe{str}
	case 't':
		return &setTopic{str}
	case 'p':
		return &publish{str}
	}
	return &ping{}
}

// setTopic implements aaInterface
type setTopic struct {
	Msg string
}

func (me *setTopic) marshal() string {
	return "t" + me.Msg
}

// func (me *SetTopic) execute(parent *ServerHandler) error {
// 	parent.theTopic = me.Msg
// 	parent.hashedTopic.FromString(me.Msg)
// 	return nil
// }

// publish implements aaInterface
type publish struct {
	Msg string
}

func (me *publish) marshal() string {
	return "p" + me.Msg
}

// func (me *Publish) execute(parent *ServerHandler) error {
// 	if parent.theTopic == "" {
// 		return errors.New("there's no topic set for the publish")
// 	}
// 	bytes := []byte(me.Msg)
// 	parent.subscriptions.SendPublishMessage(&parent.hashedTopic, parent.c, &bytes)
// 	return nil
// }

// subscribe implements aaInterface
type subscribe struct {
	Msg string
}

func (me *subscribe) marshal() string {
	return "s" + me.Msg
}

// func (me *Subscribe) execute(parent *ServerHandler) error {
// 	hashedTopic := HashType{}
// 	hashedTopic.FromString(me.Msg)
// 	parent.subscriptions.SendSubscriptionMessage(&hashedTopic, me.Msg, parent.c, nil)
// 	return nil
// }

// unsubscribe is 'u' aaInterface
type unsubscribe struct {
	Msg string
}

func (me *unsubscribe) marshal() string {
	return "u" + me.Msg
}

// func (me *Unsubscribe) execute(parent *ServerHandler) error {
// 	hashedTopic := HashType{}
// 	hashedTopic.FromString(me.Msg)
// 	parent.subscriptions.SendUnsubscribeMessage(&hashedTopic, parent.c)
// 	return nil
// }

// aaError aaInterface
type aaError struct {
	Msg string
}

func (me *aaError) marshal() string {
	return "e" + me.Msg
}

// func (me *Error) execute(parent *ServerHandler) error {
// 	// why would a client send a server an error?
// 	return nil
// }

// ping is 'g'. aaInterface
type ping struct {
	Msg string
}

func (me *ping) marshal() string {
	return "g" + me.Msg
}
