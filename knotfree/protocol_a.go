package knotfree

import (
	"errors"
	"net"
	"strings"
)

// For subscribe it's just 's' and then the rest is the topic real name.

// PublishProtocolA  eg {"C":"TopicRealName","M":"a message"}
type PublishProtocolA struct {
	T string // topic
	M string // message
}

//
// The protocol is an 'a' then an 8 bit length followed by some bytes which might start with 'p' or 's' or '{'
// we return the string

// ReadProtocolA try to get something
func ReadProtocolA(conn net.Conn, buffer []byte) (string, error) {

	pairbuf := []byte{'a', 'a'}
	n, err := conn.Read(pairbuf)
	if n != 2 {
		return string(buffer), errors.New(" needed two bytes. got " + string(buffer))
	}
	if pairbuf[0] != 'a' {
		return "", errors.New("expecting an 'a'")
	}
	msglen := int(pairbuf[1]) & 0x00FF
	var sb strings.Builder
	//s := string(buffer[2:n])
	//sb.WriteString(s)
	//msglen -= len(s)
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

// WriteProtocolA write our lame protocol
func WriteProtocolA(conn net.Conn, s string) error {
	prefix := []byte{'a', 'a'}

	if len(s) > 120 {
		// log too long
		return errors.New("WriteProtocolA string too long")
	}

	prefix[1] = byte(len(s))

	//fmt.Println("WriteProtocolA part1 " + string(prefix))

	n, err := conn.Write(prefix)
	if n != 2 || err != nil {
		// log
		return err
	}

	//fmt.Println("WriteProtocolA part2 " + string(s))

	n, err = conn.Write([]byte(s))
	if n != len(s) || err != nil {
		// log
		return err
	}
	return nil
}
