package iot2

import (
	"container/list"
	"errors"
	"fmt"
	"knotfree/iot2/reporting"
	"math/rand"
	"net"
	"sync"
	"time"
)

// MakeBunchOfClients a test of making test clients.
func MakeBunchOfClients(amt int, addr string, delay time.Duration, config *SockStructConfig, logThing *reporting.StringEventAccumulator) {

	defaultClientCallback := func(ss *SockStruct) {
		defer ss.Close(nil)
		for {
			fmt.Println("calling default sock handler")
		}
	}
	if config.callback == nil {
		config.SetCallback(defaultClientCallback)
	}
	servererr := func(ss *SockStruct, err error) {
		fmt.Println("default server is closing", err)
	}
	if config.closecb == nil {
		config.SetClosecb(servererr)
	}
	writef := func(ss *SockStruct, topicName string, payload *[]byte) error {
		fmt.Println("default server write")
		return errors.New("default server writer")
	}
	if config.writer == nil {
		config.SetWriter(writef)
	}

	// here's the gc factory:
	dialFunc := func() {
		for {
			conn, err := net.DialTimeout("tcp", addr, 60*time.Second)
			if err != nil {
				//fmt.Println("dial err", err)
				if logThing != nil {
					logThing.Collect("dial err")
				}
				time.Sleep(time.Duration(float64(delay) * rand.Float64()))
				continue
			}
			ss := NewSockStruct(conn, config)
			go config.callback(ss)
			break
		}
	}
	allAtOnce := false
	for i := 0; i < amt; i++ {
		if allAtOnce {
			go dialFunc()
		} else {
			// one at a time, with a delay.
			dialFunc()
			if i+1 < amt {
				time.Sleep(delay)
			}
		}
	}
}

// SockStruct is our wrapper for a net.Conn socket to the internet.
// We try to keep this small even though the *list.Element might be removed and then
// we pass around *list.Element instead of *SockStruct.
// It might be better to just add a next and prev to this struct and write a linked list.
// I'm not using a map as a set of these because we don't look them up.
type SockStruct struct {
	ele         *list.Element // tempted to get rid of this
	conn        net.Conn
	config      *SockStructConfig
	key         HalfHash
	topicToName map[HalfHash]string // a tree would be better?
}

// SockStructConfig could be just a stack frame but I'd like to return it.
// This could be an interface that implements range and len or and the callbacks.
// Instead we have function pointers.
type SockStructConfig struct {
	callback func(*SockStruct)
	closecb  func(*SockStruct, error)
	// This is supposed to implement the protocol when a message happens on a topic
	// and the socket is supposed to get a copy.
	writer func(ss *SockStruct, topicName string, payload *[]byte) error

	listener net.Listener

	// a linked list of all the *SockStruct that are open and not Close'd
	listlock *sync.RWMutex // so it's thread safe
	list     *list.List

	key HashType // everyone likes to feel special and unique

	sequence uint64 // every time we factory up a SockStruct we increment this and we never decrement.

	subscriber PubsubIntf
}

// NewSockStructConfig is
func NewSockStructConfig(subscriber PubsubIntf) *SockStructConfig {
	config := SockStructConfig{}
	config.subscriber = subscriber
	var alock sync.RWMutex
	config.listlock = &alock
	config.list = list.New()
	config.key.Random()
	config.sequence = 1
	return &config
}

// NewSockStruct does the new, initializes everything, and puts the new ss on the global
// list. It also increments the sequence number in SockStructConfig.
func NewSockStruct(conn net.Conn, config *SockStructConfig) *SockStruct {

	ss := new(SockStruct)
	ss.conn = conn
	ss.config = config

	ss.topicToName = make(map[HalfHash]string)

	config.listlock.Lock()
	seq := config.sequence
	config.sequence++
	ss.ele = ss.config.list.PushBack(ss)
	config.listlock.Unlock()

	ss.key = HalfHash(config.key.a + seq) // unique but poorly random.
	return ss
}

// ServeFactory starts a server and then is a factory for SockStruct's
func ServeFactory(config *SockStructConfig, addr string) {

	go func(config *SockStructConfig) {
		ln, err := net.Listen("tcp", addr)
		if err != nil {
			fmt.Println("net.Listen", err)
			return
		}
		fmt.Println("serving now ")
		config.listener = ln
		defer ln.Close()
		for {
			conn, err := ln.Accept()
			ss := NewSockStruct(conn, config)
			if err != nil {
				//fmt.Println("net.Accept noopsee2", err)
				ss.Close(err)
				return
			}
			err = SocketSetup(ss.conn)
			if err != nil {
				ss.Close(err)
				continue
			}
			go ss.config.callback(ss)
		}
	}(config)
}

// Close closes the conn
// and the rest of the work too
func (ss *SockStruct) Close(err error) {

	if ss.conn != nil {
		ss.conn.Close()
		ss.conn = nil
	}
	if ss.ele != nil {
		ss.config.listlock.Lock()
		ss.config.list.Remove(ss.ele)
		ss.config.listlock.Unlock()
		ss.ele = nil
		ss.config.closecb(ss, err)
	}
}

// GetConn is
func (ss *SockStruct) GetConn() net.Conn {
	return ss.conn
}

// GetSequence is
func (ss *SockStruct) GetSequence() uint64 {
	return uint64(ss.key) - ss.config.key.a
}

// SetCallback is
func (config *SockStructConfig) SetCallback(cb func(*SockStruct)) {
	config.callback = cb
}

// SetClosecb is
func (config *SockStructConfig) SetClosecb(closecb func(*SockStruct, error)) {
	config.closecb = closecb
}

// SetWriter is
func (config *SockStructConfig) SetWriter(w func(ss *SockStruct, topicName string, payload *[]byte) error) {
	config.writer = w
}

// Len is an obvious wrapper
func (config *SockStructConfig) Len() int {
	config.listlock.Lock()
	val := config.list.Len()
	config.listlock.Unlock()
	return val
}

// Close closes the listener but not all the connections.
func (config *SockStructConfig) Close(err error) {
	config.listener.Close()
}

// SocketSetup sets common options
//
func SocketSetup(conn net.Conn) error {
	tcpConn := conn.(*net.TCPConn)
	err := tcpConn.SetReadBuffer(4096)
	if err != nil {
		//srvrLogThing.Collect("SS err1 " + err.Error())
		return err
	}
	err = tcpConn.SetWriteBuffer(4096)
	if err != nil {
		//srvrLogThing.Collect("SS err2 " + err.Error())
		return err
	}
	err = tcpConn.SetNoDelay(true)
	if err != nil {
		//	srvrLogThing.Collect("SS err3 " + err.Error())
		return err
	}
	// SetReadDeadline and SetWriteDeadline
	err = tcpConn.SetDeadline(time.Now().Add(20 * time.Minute))
	if err != nil {
		// /srvrLogThing.Collect("cl err4 " + err.Error())
		return err
	}
	return nil
}
