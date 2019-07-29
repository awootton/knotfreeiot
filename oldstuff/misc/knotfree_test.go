// Copyright 2019 Alan Tracey Wootton

package misc_test

import (
	"fmt"
	"knotfree/oldstuff/misc"
	"math/rand"
	"strings"
	"time"
)

// Do one write to a ByteChanReadWriter
// which is connected to a ChanBuilder
func ExampleByteChanReadWriter_suffix1() {
	testStr := "Testing, testing. This will be todays test string. How is everyone doing?"

	src := misc.NewByteChanReadWriter(10)

	sink := misc.NewByteArrayBuilder(src)

	n, err := src.Write([]byte(testStr))

	for i := 0; i < 10; i++ {
		time.Sleep(100 * time.Microsecond)
	}

	fmt.Println(n, err, sink.String())
	// Output: 73 <nil> Testing, testing. This will be todays test string. How is everyone doing?
}

// Do many write to a ByteChanReadWriter
// which is connected to a ChanBuilder
func ExampleByteChanReadWriter_suffix2() {
	testStr := "Testing, testing. This will be todays test string. How is everyone doing?"

	src := misc.NewByteChanReadWriter(10)

	sink := misc.NewByteArrayBuilder(src)

	for _, ch := range []byte(testStr) {
		_, _ = src.Write([]byte{ch})
		time.Sleep(100 * time.Microsecond)
	}
	time.Sleep(100 * time.Microsecond)
	fmt.Println(sink.String())
	// Output: Testing, testing. This will be todays test string. How is everyone doing?
}

// go for the timeout. There is nothing at the other end of the chan in ByteChanReadWriter.
// It holds 2 and we're writing 3 so --- timeout.
func ExampleByteChanReadWriter_suffix3() {

	src := misc.NewByteChanReadWriter(2)
	src.SetTimeout(time.Millisecond)
	n, err := src.Write([]byte{'a', 'b', 'c'})

	fmt.Println(n, err)
	// Output:2 Timeout ByteChanReadWriter Write
}

// let's try the read
func ExampleByteChanReadWriter_suffix4() {

	bcrw := misc.NewByteChanReadWriter(2)
	bcrw.SetTimeout(1000 * time.Millisecond)

	// pipe the bcrw to a stringBuilder.
	var result strings.Builder
	go func() {
		for {
			ch := []byte{'a'}
			n, err := bcrw.Read(ch)
			if n != 1 {
				fmt.Println("expecting 1")
			}
			if err != nil {
				fmt.Println("got err", err.Error())
			}
			result.Write(ch)
		}
	}()

	// writing many into a chan holding 2
	n, err := bcrw.Write([]byte("abcdefghijklmnopq"))

	for i := 0; i < 1000; i++ {
		time.Sleep(100 * time.Microsecond)
	}
	fmt.Println(n, err, result.String())
	time.Sleep(100 * time.Microsecond)
	// Output: 17 <nil> abcdefghijklmnopq

}

func ExampleByteChunkedReadWriter_suffix1() {
	testStr := "Testing, testing. This will be todays test string. How is everyone doing?"

	bcrw := misc.NewByteChunkedReadWriter(1)
	bcrw.SetDebugPrint(true)

	sink := misc.NewByteArrayBuilderChunked(bcrw)

	n, err := bcrw.Write([]byte(testStr))

	for i := 0; i < 10; i++ {
		time.Sleep(100 * time.Microsecond)
	}

	fmt.Println(n, err, sink.String())
	// Output: sent: Testing, testing
	// sent: . This will be t
	// sent: odays test strin
	// sent: g. How is everyo
	// sent: ne doing?
	// 73 <nil> Testing, testing. This will be todays test string. How is everyone doing?

}

// Let's try the read. Reading variable amounts from a channel with chunks is tricky.
// Some parts of every chunk might not be used in every call to read.
func ExampleByteChunkedReadWriter_suffix2() {

	rand.NewSource(33)

	testStr := "Testing, testing. This will be todays test string. How is everyone doing?"

	bcrw := misc.NewByteChunkedReadWriter(1)
	//bcrw.SetDebugPrint(true)

	bcrw.SetTimeout(100 * time.Millisecond)
	// pipe the bcrw to a stringBuilder.
	var result strings.Builder
	go func() {
		need := len(testStr)
		gotAmt := 0
		for {
			r := 1 + rand.Intn(20)
			s := min(need-gotAmt, r)
			ch := make([]byte, s)
			n, _ := bcrw.Read(ch)
			gotAmt += n
			if n != s {
				fmt.Println("ERROR. expected =", n, s)
			}
			//fmt.Println("aread:", string(ch))
			result.Write(ch)
		}
	}()

	// writing many into a chan holding 1*16 bytes
	n, err := bcrw.Write([]byte(testStr))

	for i := 0; i < 100; i++ {
		time.Sleep(100 * time.Microsecond)
	}

	fmt.Println(n, err, result.String())
	// Output: 73 <nil> Testing, testing. This will be todays test string. How is everyone doing?

}

// go for the timeout. There is nothing at the other end of the chan in ByteChanReadWriter.
// It holds 2 and we're writing 3 so --- timeout.
func ExampleByteChunkedReadWriter_suffix3() {

	src := misc.NewByteChunkedReadWriter(2)
	src.SetTimeout(time.Millisecond)
	n, err := src.Write([]byte("1234567890ABCDEF 1234567890ABCDEF 1234567890ABCDEF "))

	fmt.Println(n, err)
	// it only writes the first two chunks of 16 bytes
	// Output:32 Timeout ByteChunkedReadWriter Write
}

// This is supposed to timeout and not read anything
func ExampleByteChunkedReadWriter_suffix4() {

	src := misc.NewByteChunkedReadWriter(2)
	src.SetTimeout(time.Millisecond)
	n, err := src.Read([]byte("1234567890ABCDEF 1234567890ABCDEF 1234567890ABCDEF "))

	fmt.Println(n, err)
	// it only writes the first two chunks of 16 bytes
	// Output:32 Timeout ByteChunkedReadWriter Read
}

// A common utility function.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
