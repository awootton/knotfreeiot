package knotfree

import (
	"errors"
	"knotfree/knotfree/types"
	"net"
	"strings"
)

// This lame pub/sub protocol is going to be a length byte followed by a string of len 0 to 255.
// The first char in the string
// indicates which type of message we're getting so a zero string is going to be an error.

// ProtocolAaSetTopic is 't'
type ProtocolAaSetTopic struct {
	msg string
}

// ProtocolAaPublish is 'p'
type ProtocolAaPublish struct {
	msg string
}

// ProtocolAaSubscribe is 's'
type ProtocolAaSubscribe struct {
	msg string
}

// ProtocolAaUnsubscribe is 'u'
type ProtocolAaUnsubscribe struct {
	msg string
}

// ProtocolAaError is 'e'. For clients
type ProtocolAaError struct {
	msg string
}

// ProtocolAaPing is 'g'. For clients
type ProtocolAaPing struct {
	msg string
}

// ProtocolAaServerHandler a
type ProtocolAaServerHandler struct {
	theTopic    string
	hashedTopic types.HashType
	c           *Connection
}

// Serve will
func (me *ProtocolAaServerHandler) Serve() error {
	str, err := ReadProtocolAstr(me.c.tcpConn)
	if err != nil {
		connLogThing.Collect("rProtA err " + str + err.Error())
		return err
	}
	connLogThing.Sum("Aa r bytes", len(str))
	// ok, so what is the message? subscribe or publish?
	// fmt.Println("Have Server str _a " + str)
	// eg sAchannel

	// CONNECT c
	// PUBLISH p
	// SUBSCRIBE s
	// UNSUBSCRIBE u
	// PING g
	// DISCONNECT d

	switch str[0] {
	case 's':
		subTopic := str[1:]
		hashedTopic := types.HashType{}
		hashedTopic.FromString(subTopic)
		//fmt.Println("sub CONN key " + c.Key.String())
		//fmt.Println("Have subTopic " + subTopic)
		// we'll fill in a sub request and 'mail' it to the sub handler
		// TODO: change to proc call
		//subr := SubscriptionMessage{}
		//	subr.Topic.FromString(subTopic)
		//fmt.Println("Have subChan becomes " + subr.Channel.String())
		//	subr.ConnectionID.FromHashType(&me.c.Key)
		//fmt.Println("subscribe ConnectionID is " + subr.ConnectionID.String())
		me.c.realTopicNames[hashedTopic] = subTopic
		SendSubscriptionMessage(&hashedTopic, &me.c.Key)
	case 't':
		me.theTopic = str[1:]
		me.hashedTopic = types.HashType{}
		me.hashedTopic.FromString(me.theTopic)
	case 'p':
		if me.theTopic == "" {
			return errors.New("there's no topic set for the publish")
		}
		payload := []byte(str[1:])
		SendPublishMessage(&me.hashedTopic, &me.c.Key, &payload)
	}
	// we don't have an unsubscribe yet.
	return nil
}

// ReadProtocolAstr will block trying to get a string until the conn times out.
func ReadProtocolAstr(conn net.Conn) (string, error) {

	buffer := make([]byte, 256)
	ch := []byte{'a'}
	n, err := conn.Read(ch)
	if n != 1 {
		if err != nil { // probably timed out
			return string(buffer), err
		}
		return string(buffer), errors.New(" needed 1 bytes. got " + string(buffer))
	}
	msglen := int(ch[0]) & 0x00FF
	var sb strings.Builder
	for msglen > 0 {
		n, err = conn.Read(buffer[:msglen])
		s := string(buffer[:n])
		sb.WriteString(s)
		msglen -= n
		if err != nil {
			return "", err
		}
	}
	return sb.String(), nil
}

// WriteProtocolAaStr writes our lame protocol to the conn
func WriteProtocolAaStr(conn net.Conn, str string) error {

	strbytes := []byte(str)
	if len(strbytes) > 255 {
		return errors.New("WriteProtocolA string too long")
	}
	prefix := []byte{byte(len(strbytes))}
	n, err := conn.Write(prefix)
	if n != 1 || err != nil {
		return err
	}
	n, err = conn.Write(strbytes)
	if n != len(strbytes) || err != nil {
		return err
	}
	return nil
}
