package iot2

import (
	"bufio"
	"errors"
	"fmt"
	"strings"
)

// ServerOfStrings - implement string messages
func ServerOfStrings(subscribeMgr PubsubIntf, addr string) *SockStructConfig {

	config := NewSockStructConfig(subscribeMgr)

	ServerOfStringsInit(config)

	ServeFactory(config, addr)

	return config
}

// ServerOfStringsInit is to set default callbacks.
func ServerOfStringsInit(config *SockStructConfig) {

	config.SetCallback(strServeCallback)

	servererr := func(ss *SockStruct, err error) {
		fmt.Println("server is closing", err)
	}
	config.SetClosecb(servererr)

	//  the writer
	handleTopicPayload := func(ss *SockStruct, topic string, payload *[]byte) error {

		// make a 'command' (called 'got'), serialize it and write it to the sock
		str := string(*payload)
		cmd := `got "` + topic + `" ` + str
		// TODO: warning. this will BLOCK and jam up the whole machine.
		n, err := ss.conn.Write([]byte(cmd + "\n"))
		if err != nil || n != (len(cmd)+1) {
			fmt.Println("error in str writer", n, err, cmd)
			return err
		}
		return nil
	}

	config.SetWriter(handleTopicPayload)
}

// ServerOfStringsWrite translates the object 'str' into bytes and sends it.
// Clients will need this and it's NOT the same as config.writer for ServerOfStrings
func ServerOfStringsWrite(ss *SockStruct, str string) error {
	bytes := []byte(str + "\n")
	n, err := ss.conn.Write(bytes)
	if err != nil {
		ss.Close(err)
		return err
	}
	_ = n // fixme
	return nil
}

// strServeCallback is the default callback which implements an api
// to the pub sub manager.
// This protol echos back commands.
func strServeCallback(ss *SockStruct) {

	reader := bufio.NewReader(ss.conn)

	for {
		text, err := reader.ReadString('\n')
		if err != nil {
			err = ServerOfStringsWrite(ss, "error: "+err.Error())
			ss.Close(err)
			return
		}
		if len(text) <= 1 {
			err = ServerOfStringsWrite(ss, "error: Empty sent")
			if err != nil {
				ss.Close(err)
			}
			continue
		}
		text = text[0 : len(text)-1] // remove the \n
		first, remaining := GetFirstWord(text)
		switch first {
		case "exit":
			ServerOfStringsWrite(ss, "exit")
			err := errors.New("exit")
			ss.Close(err)

		case "sub":
			topic := strings.Trim(remaining, " ")
			if len(topic) <= 0 {
				ServerOfStringsWrite(ss, "error: say 'sub mytopic' and not "+text)
			} else {
				ss.config.subscriber.SendSubscriptionMessage(topic, ss)
				ServerOfStringsWrite(ss, "sub "+topic)
			}

		case "unsub":
			topic := strings.Trim(remaining, " ")
			if len(topic) <= 0 {
				ServerOfStringsWrite(ss, "error: say 'unsub mytopic' and not "+text)
			} else {
				ss.config.subscriber.SendUnsubscribeMessage(topic, ss)
				ServerOfStringsWrite(ss, "unsub "+topic)
			}

		case "pub":
			topic, payload := GetFirstWord(remaining)
			if len(topic) <= 0 || len(payload) < 0 {
				ServerOfStringsWrite(ss, "error: say 'pub mytopic mymessage' and not "+text)
			} else {
				topicHash := HashType{}
				topicHash.FromString(topic)
				bytes := []byte(payload)
				ss.config.subscriber.SendPublishMessage(topic, ss, &bytes)
				ServerOfStringsWrite(ss, "pub "+topic+" "+payload)
			}

		default:
			ServerOfStringsWrite(ss, "error: unknown command "+text)
		}
	}
}

// GetFirstWord will return the first word of str where the words are delimited by spaces.
// It will also return the remaining words in string. If there is a '"' at the beginning of str then we'll match quotes to
// get the first word. There will be no escaping of quotes.
func GetFirstWord(str string) (string, string) {

	str = strings.Trim(str, " ")
	if len(str) <= 1 {
		return str, ""
	}
	if str[0:1] == "\"" {
		str := str[1:]
		pos := strings.IndexByte(str, '"')
		if pos <= 0 {
			// no 2nd quote
			return strings.Trim(str, " "), ""
		}
		first := str[0:pos]
		second := str[pos+1:]
		return strings.Trim(first, " "), strings.Trim(second, " ")
	}
	// else look for a space
	pos := strings.IndexByte(str, ' ')
	if pos <= 0 {
		// no 2nd space
		return str, ""
	}
	first := str[0:pos]
	second := str[pos:]
	return strings.Trim(first, " "), strings.Trim(second, " ")
}
