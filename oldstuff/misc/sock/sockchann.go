package main

import (
	"fmt"
	"knotfree/oldstuff/types"
	"net"
	"time"
)

var trace = true

const testport = "localhost:9877"

// BytesDuplexChannel is
type BytesDuplexChannel struct {
	Up     *chan []byte
	Down   *chan []byte
	conn   net.Conn
	config *BytesDuplexChannelConfig
}

// BytesDuplexChannelConfig - TODO; reuse these as they will be all the same.
type BytesDuplexChannelConfig struct {
	reverse bool
	retry   bool
	//	buffersize uint8

	callback func(*BytesDuplexChannel)
	closer   func(*BytesDuplexChannel, error)

	ln net.Listener
}

// NewTCPDuplexChann is
func NewTCPDuplexChann(conn net.Conn, size int, config *BytesDuplexChannelConfig) *BytesDuplexChannel {
	dc := BytesDuplexChannel{}
	upload := make(chan []byte, size)
	dc.Up = &upload
	download := make(chan []byte, size)
	dc.Down = &download
	dc.conn = conn
	dc.config = config // &BytesDuplexChannelConfig{}

	return &dc
}

// implements read and write

func (dc *BytesDuplexChannel) Read(buffer []byte) (n int, err error) {

	n, err = dc.conn.Read(buffer)

	if err != nil {
		//fmt.Println("BytesDuplexChannel read error ", err)
		//close(*dc.Up)
		//close(*dc.Down)
		dc.conn.Close()
		dc.config.closer(dc, err)
		return 0, err
	}
	return n, nil
	//fmt.Println("BytesDuplexChannel.read ", string(buffer[0:n]))
	// slice := buffer[0:n]
	// if dc.config.reverse {
	// 	*dc.Down <- slice
	// } else {
	// 	*dc.Up <- slice
	// }
}

func (dc *BytesDuplexChannel) Write(buffer []byte) (n int, err error) {
	need := len(buffer)
	for need > 0 {
		n, err := dc.conn.Write(buffer)
		if err != nil {
			//fmt.Println("Write error ", err)
			//close(*dc.Down)
			//close(*dc.Up)
			dc.conn.Close()
			dc.config.closer(dc, err)
			return 0, err
		}
		//fmt.Println("wrote ", string(buffer[0:n]))
		need -= n
		buffer = buffer[n:]
	}
	return n, nil
}

// Close closes the socket. Those reading it should notice.
func (dc *BytesDuplexChannel) Close() {
	dc.conn.Close()
	//close(*dc.Down)
	//close(*dc.Up)
}

// read tcp, push the buffer into the chan
func (dc *BytesDuplexChannel) read() {
	for {
		buffer := make([]byte, 32)
		n, err := dc.conn.Read(buffer)
		if err != nil {
			fmt.Println("BytesDuplexChannel read error ", err)
			//close(*dc.Up)
			//close(*dc.Down)
			dc.conn.Close()
			dc.config.closer(dc, err)
			return
		}
		//fmt.Println("BytesDuplexChannel.read ", string(buffer[0:n]))
		slice := buffer[0:n]
		if dc.config.reverse {
			*dc.Down <- slice
		} else {
			*dc.Up <- slice
		}
	}
}

//
func (dc *BytesDuplexChannel) write() {
	for {
		var buffer []byte
		if dc.config.reverse {
			buffer = <-*dc.Up
		} else {
			buffer = <-*dc.Down
		}
		//fmt.Println("buffer ", string(buffer))
		need := len(buffer)
		for need > 0 {
			n, err := dc.conn.Write(buffer)
			if err != nil {
				fmt.Println("Write error ", err)
				//close(*dc.Down)
				//close(*dc.Up)
				dc.conn.Close()
				dc.config.closer(dc, err)
				return
			}
			//fmt.Println("wrote ", string(buffer[0:n]))
			need -= n
			buffer = buffer[n:]
		}
	}
}

// Serve is
func Serve(cb func(*BytesDuplexChannel), closecb func(*BytesDuplexChannel, error), chanDepth int) {
	config := &BytesDuplexChannelConfig{}
	config.callback = cb
	config.closer = closecb
	go func(config *BytesDuplexChannelConfig) {
		ln, err := net.Listen("tcp", testport)
		config.ln = ln
		if err != nil {
			fmt.Println("net.Listen oops9i", err)
			config.closer(NewTCPDuplexChann(nil, chanDepth, config), err)
			return
		}
		defer ln.Close()
		for {
			conn, err := ln.Accept()
			dc := NewTCPDuplexChann(conn, chanDepth, config)
			if err != nil {
				//fmt.Println("net.Accept oopsee", err)
				dc.conn.Close()
				dc.config.closer(dc, err)
				continue
			}
			err = types.SocketSetup(conn)
			if err != nil {
				dc.config.closer(dc, err)
				continue
			}
			go dc.read()
			go dc.write()
			go cb(dc)
		}
	}(config)
}

// ServeNoGo is same as above but without go routines
func ServeNoGo(cb func(*BytesDuplexChannel), closecb func(*BytesDuplexChannel, error), chanDepth int) *BytesDuplexChannelConfig {

	config := &BytesDuplexChannelConfig{}
	config.callback = cb
	config.closer = closecb

	go func(config *BytesDuplexChannelConfig) {
		ln, err := net.Listen("tcp", testport)
		if err != nil {
			fmt.Println("net.Listen oops9i2", err)
			return
		}
		config.ln = ln
		defer ln.Close()
		for {
			conn, err := ln.Accept()
			dc := NewTCPDuplexChann(conn, chanDepth, config)
			if err != nil {
				//fmt.Println("net.Accept noopsee2", err)
				dc.config.closer(dc, err)
				return
			}
			err = types.SocketSetup(dc.conn)
			if err != nil {
				dc.config.closer(dc, err)
				continue
			}
			// go dc.read()
			// go dc.write()
			go cb(dc)
		}
	}(config)
	return config
}

// Call is like dial
func Call(cb func(*BytesDuplexChannel), closecb func(*BytesDuplexChannel, error), chanDepth int) (*BytesDuplexChannel, error) {

	config := &BytesDuplexChannelConfig{}
	config.callback = cb
	config.closer = closecb
	connectStr := testport
	conn, err := net.DialTimeout("tcp", connectStr, 60*time.Second)
	if err != nil {
		fmt.Println("net.Call err", err)
		return nil, err
	}
	err = types.SocketSetup(conn)
	if err != nil {
		return nil, err
	}
	//fmt.Println("called conn ", conn, err)
	dc := NewTCPDuplexChann(conn, chanDepth, config)
	dc.config.reverse = true
	dc.config.closer = closecb
	go dc.read()
	go dc.write()
	go cb(dc)
	return dc, nil
}

// CallNoGo is like dial
func CallNoGo(cb func(*BytesDuplexChannel), closecb func(*BytesDuplexChannel, error), chanDepth int) (*BytesDuplexChannel, error) {

	config := &BytesDuplexChannelConfig{}
	config.callback = cb
	config.closer = closecb

	connectStr := testport
	conn, err := net.DialTimeout("tcp", connectStr, 60*time.Second)
	if err != nil {
		fmt.Println("net.Call err", err)
		return nil, err
	}
	err = types.SocketSetup(conn)
	if err != nil {
		return nil, err
	}
	//fmt.Println("called conn ", conn, err)
	dc := NewTCPDuplexChann(conn, chanDepth, config)
	dc.config.reverse = true
	dc.config.closer = closecb
	// go dc.read()
	// go dc.write()
	go cb(dc)
	return dc, nil
}

func main() {
	fmt.Println("hello socker")

	ChanAndSubWithTCP3(4, 4)

	//ChanAndSubWithTCP2(1, 1)

	for {
		time.Sleep(time.Second)
		fmt.Println("tick")
		time.Sleep(time.Second)
		fmt.Println("tock")
	}

}
