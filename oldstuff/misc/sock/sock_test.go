package main

import (
	"testing"
)

type SockSpeedTest int

// ExampleSockSpeedTest is
func ExampleSockSpeedTest() {

	ChanAndSubWithTCP3(16, 16)

	// Output:

}

func BenchmarkSockSpeedTest(b *testing.B) {

	size := 3000
	for i := 0; i < b.N; i++ {
		ChanAndSubWithTCP3(size, size)
	}

	// size := 100	BenchmarkSockSpeedTest-8   	      10	 103111000 ns/op	  431668 B/op	    8054 allocs/op
	// 1000     	BenchmarkSockSpeedTest-8   	       2	 652111000 ns/op	 4233688 B/op	   79302 allocs/op
	// 2000 		BenchmarkSockSpeedTest-8   	       1	1080111000 ns/op	 8648352 B/op	  158990 allocs/op    1080 ms
	// 3000         BenchmarkSockSpeedTest-8   	       1	1721553205 ns/op	12778200 B/op	  238255 allocs/op
	// 4000			BenchmarkSockSpeedTest-8   	       1	2278658441 ns/op	17188752 B/op	  317325 allocs/op

	// so about .7 sec for 1000 connections or 1400 per sec
	// and about 3.8 MiB per 1000 or 4.4 or 4.1 or 4.4
	// or ~ 4k per socket

	// change buffers to 1024
	// 2000 BenchmarkSockSpeedTest-8   	       1	1033598530 ns/op	 8632112 B/op	  158946 allocs/op
	// 3000 BenchmarkSockSpeedTest-8   	       1	1815182324 ns/op	12767688 B/op	  238148 allocs/op
	// about 4.1 k per socket
}
