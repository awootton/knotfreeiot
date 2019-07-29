package types

import (
	"container/list"
	"fmt"
	"math/rand"
	"net"
	"strconv"
	"sync"
	"time"
)

const testport = "knotfreeserver:6162"

// SockStruct is
// These can be public
type SockStruct struct {
	conn    net.Conn
	config  *SockStructConfig
	element list.Element
}

// Close closes the conn
// and the rest of the work too
func (ss *SockStruct) Close(err error) {
	ss.conn.Close()
	ss.config.listlock.Lock()
	ss.config.list.Remove(&ss.element)
	ss.config.listlock.Unlock()
	ss.config.closecb(ss, err)
}

// GetConn is
func (ss *SockStruct) GetConn() net.Conn {
	return ss.conn
}

// SockStructConfig could be just a stack frame but I'd like to return it.
// This needs to be an interface that implements range and len or something.
type SockStructConfig struct {
	callback func(*SockStruct)
	closecb  func(*SockStruct, error)

	listener net.Listener

	listlock *sync.RWMutex
	list     *list.List //[]*SockStruct

	id HashType
}

// SetCallback closes the listener
func (config *SockStructConfig) SetCallback(cb func(*SockStruct)) {
	config.callback = cb
}

// SetClosecb closes the listener
func (config *SockStructConfig) SetClosecb(closecb func(*SockStruct, error)) {
	config.closecb = closecb
}

// // SetListener is
// func (config *SockStructConfig) SetListener(listener net.Listener) {
// 	config.listener = listener
// }

// Len is an obvious wrapper
func (config *SockStructConfig) Len() int {
	config.listlock.Lock()
	val := config.list.Len()
	config.listlock.Unlock()
	return val
}

// Close closes the listener
func (config *SockStructConfig) Close(err error) {
	config.listener.Close()
}

// NewSockStructConfig is
func NewSockStructConfig() *SockStructConfig {
	config := SockStructConfig{}
	var alock sync.RWMutex
	config.listlock = &alock
	config.list = list.New()
	randomStr := strconv.FormatInt(rand.Int63(), 16) + strconv.FormatInt(rand.Int63(), 16)
	config.id.FromString(randomStr)
	return &config
}

// NewSockStruct is
func NewSockStruct(conn net.Conn, config *SockStructConfig) *SockStruct {
	ss := SockStruct{}
	ss.conn = conn
	ss.config = config
	//ss.element = list.Element{}
	ss.element.Value = ss

	return &ss
}

//ServeFactory ServeNoGo is same as above but without go routines
func ServeFactory(config *SockStructConfig) {

	go func(config *SockStructConfig) {
		ln, err := net.Listen("tcp", testport)
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
			ss.config.listlock.Lock()
			ss.config.list.PushBack(ss.element)
			ss.config.listlock.Unlock()
			go ss.config.callback(ss)
		}
	}(config)
}

// SocketSetup sets common options
func SocketSetup(conn net.Conn) error {
	tcpConn := conn.(*net.TCPConn)
	err := tcpConn.SetReadBuffer(1024) // 4096)
	if err != nil {
		//srvrLogThing.Collect("SS err1 " + err.Error())
		return err
	}
	err = tcpConn.SetWriteBuffer(1024) // 4096)
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
