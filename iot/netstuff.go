package iot

import (
	"bytes"
	"errors"
	"fmt"
	"time"
)

var defaultTimeout = 10 * time.Second

// ByteChanReadWriter - implements io.Reader and io.Writer using a channel
// We can use these as adapters.
type ByteChanReadWriter struct {
	wire    chan byte
	timeout time.Duration
}

// NewByteChanReadWriter -
func NewByteChanReadWriter(cap int) ByteChanReadWriter {
	return ByteChanReadWriter{make(chan byte, cap), defaultTimeout}
}

// SetTimeout - change the timeout from the default.
func (me *ByteChanReadWriter) SetTimeout(t time.Duration) {
	me.timeout = t
}

func (me *ByteChanReadWriter) Read(p []byte) (n int, err error) {
	for i := range p {
		select {
		case ch := <-me.wire:
			p[i] = ch
		case <-time.After(me.timeout):
			return i, errors.New("Timeout ByteChanReadWriter Read")
		}
	}
	return len(p), nil
}

func (me *ByteChanReadWriter) Write(p []byte) (n int, err error) {
	for i, ch := range p {
		select {
		case me.wire <- ch:
		case <-time.After(me.timeout):
			return i, errors.New("Timeout ByteChanReadWriter Write")
		}
	}
	return len(p), nil
}

// ByteChunkedReadWriter implements io.Reader and io.Writer using a channel
// This is similar to the ByteChanReadWriter above except the bytes are passed through
// the channel in batches. No batch has more than maxSize bytes.
type ByteChunkedReadWriter struct {
	wire       chan []byte
	leftovers  []byte
	timeout    time.Duration
	maxSize    int
	debugPrint bool
}

// type chunkStruct struct {
// 	amt  int
// 	data []byte
// }

// NewByteChunkedReadWriter -
func NewByteChunkedReadWriter(cap int) ByteChunkedReadWriter {
	return ByteChunkedReadWriter{wire: make(chan []byte, cap), timeout: defaultTimeout, maxSize: 16}
}

// SetTimeout - change the timeout from the default.
func (me *ByteChunkedReadWriter) SetTimeout(t time.Duration) {
	me.timeout = t
}

// SetDebugPrint causes read and write to print the chunks
func (me *ByteChunkedReadWriter) SetDebugPrint(b bool) {
	me.debugPrint = b
}

func (me *ByteChunkedReadWriter) Read(readDest []byte) (n int, err error) {
	amt := 0
	need := len(readDest)
	do := func(bblen int, bb []byte) {

		me.leftovers = nil
		size := min(need-amt, bblen)
		if me.debugPrint {
			fmt.Println("adding:", string(bb[0:size]))
		}
		for i := 0; i < size; i++ {
			readDest[amt+i] = bb[i]
		}
		amt += size
		if size < bblen { // if there's some left - save for dinner
			me.leftovers = bb[size:]
		}
	}
	if me.leftovers != nil {
		do(len(me.leftovers), me.leftovers)
	}
	for amt < need {
		select {
		case ch := <-me.wire: // get
			do(len(ch), ch[0:])
		case <-time.After(me.timeout):
			return amt, errors.New("Timeout ByteChunkedReadWriter Read")
		}
	}
	return len(readDest), nil
}

func (me *ByteChunkedReadWriter) Write(p []byte) (n int, err error) {
	amt := 0
	need := len(p)
	for amt < need {
		size := min(need-amt, me.maxSize)
		ch := make([]byte, size)    // alloc
		for i := 0; i < size; i++ { // copy
			ch[i] = p[amt+i]
		}
		select {
		case me.wire <- ch: // send
			if me.debugPrint {
				fmt.Println("sent:", string(ch))
			}
			amt += len(ch)
		case <-time.After(me.timeout):
			return amt, errors.New("Timeout ByteChunkedReadWriter Write")
		}
	}
	return need, nil
}

//
//
//

//
//
//

// A common utility function.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ByteArrayBuilder is just a struct with a bytes.Buffer except the constuctor
// connects a chan of bytes to it.
type ByteArrayBuilder struct {
	dest bytes.Buffer
}

// NewByteArrayBuilder returns a that is pulling from a chan of bytes.
func NewByteArrayBuilder(from ByteChanReadWriter) *ByteArrayBuilder {
	bcb := ByteArrayBuilder{bytes.Buffer{}}
	go func() {
		for {
			b := [1]byte{}
			b[0] = <-from.wire

			_, _ = bcb.dest.Write(b[0:])
		}
	}()
	return &bcb
}

// NewByteArrayBuilderChunked returns a Builder pulling from a chan of chunks.
func NewByteArrayBuilderChunked(from ByteChunkedReadWriter) *ByteArrayBuilder {
	bcb := ByteArrayBuilder{bytes.Buffer{}}
	go func() {
		for {
			bbb := <-from.wire
			_, _ = bcb.dest.Write(bbb)
		}
	}()
	return &bcb
}

func (me *ByteArrayBuilder) String() string {
	return me.dest.String()
}
