package knotfree

import (
	"errors"
	"net"
	"strings"
)

// For subscribe it's just 's' and then the rest is the channel real name.

// PublishProtocolA  eg {"C":"channelRealName","M":"a message"}
type PublishProtocolA struct {
	C string // channel
	M string // message
}

//
// The protocol is an 'a' then an 8 bit length followed by some bytes which might start with 'p' or 's' or '{'
// we return the string

// ReadProtocolA try to get something
func ReadProtocolA(conn net.Conn, buffer []byte) (string, error) {

	n, err := conn.Read(buffer)
	if n <= 2 {
		return "", errors.New("needed two bytes")
	}
	if buffer[0] != 'a' {
		return "", errors.New("expecting an 'a'")
	}
	msglen := int(buffer[1]) & 0x00FF
	var sb strings.Builder
	s := string(buffer[2:n])
	sb.WriteString(s)
	msglen -= len(s)
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
		return errors.New("string too long")
	}

	prefix[1] = byte(len(s))

	n, err := conn.Write(prefix)
	if n != 2 || err != nil {
		// log
		return err
	}
	n, err = conn.Write([]byte(s))
	if n != len(s) || err != nil {
		// log
		return err
	}
	return nil
}
