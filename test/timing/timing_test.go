// Copyright 2019 Alan Tracey Wootton

package timing_test

import (
	"fmt"
	"knotfree/iot"
	"knotfree/test/timing"
	"testing"
)

// ExampleChanAndSubWithTCP is
func ExampleChanAndSubWithTCP() {

	timing.ChanAndSubWithTCP(100, 100)

	// Output: now they are all dialed in.
	// messages published.
	// waiting...
	// done
}

// BenchmarkChanAndSubWithTCP
func BenchmarkChanAndSubWithTCP(b *testing.B) {
	for i := 0; i < b.N; i++ {
		timing.ChanAndSubWithTCP(100, 100)
	}
	// 100, 100   	 281,123,716 ns/op       or about 4 per sec

}

// ExampleMeasureChanAndSub is also the test.
func ExampleMeasureChanAndSub() {

	timing.MeasureChanAndSub(1000, 1000)

	// Output: waiting...
	// done
}

func BenchmarkMeasureChanAndSub(b *testing.B) {
	for i := 0; i < b.N; i++ {
		timing.MeasureChanAndSub(100*1000, 100*1000)
	}
	// 1000,1000   	           12114671 ns/op	 4028138 B/op	   77456 allocs/op
	// 2000,2000    	       25547932 ns/op	 8066258 B/op	  155893 allocs/op
	// 3000,3000    	       34957641 ns/op	11994478 B/op	  234189 allocs/op
	// 5000,5000   	      	   58040264 ns/op	20073551 B/op	  391248 allocs/op
	// 10000,10000    	       121221459 ns/op	40250574 B/op	  785004 allocs/op
	// 20000,20000    	       246207336 ns/op	81379249 B/op	 1578491 allocs/op
	// 50000,50000    	       651209920 ns/op	206616052 B/op	 3989637 allocs/op
	// 100*1000, 100*1000     1491630299 ns/op	461981512 B/op	 8102782 allocs/op
}

func BenchmarkMakeChanObjects(b *testing.B) {

	//array := make([]types.ConnectionIntf, b.N, b.N)
	for i := 0; i < b.N; i++ {
		c := iot.NewConnection(nil, nil)
		//	array[i] = &c
		iot.RememberConnection(c)
	}
	// md5:     BenchmarkMakeChanObjects-8   	 1000000	      1952 ns/op	     442 B/op	       8 allocs/op
	// highway: BenchmarkMakeChanObjects-8   	 1000000	      1484 ns/op	     554 B/op	       8 allocs/op
}

func BenchmarkMakeSubscriptions(b *testing.B) {
	// do 1000*1000 subscriptions on one channel
	for i := 0; i < b.N; i++ {
		memstat := timing.OneTestOfSubsMemory(1000*1000, 1)
		fmt.Println("bytes per subscription = ", (memstat.Bytes)/uint64(1000*1000)) // 391
	}
	// 2,238,941,732 ns/op	636443832 B/op	10038239 allocs/op
	// about 2 microseconds each or about 500,000 per sec
}

func BenchmarkMakeChannels(b *testing.B) {
	// do 1000*1000 subscriptions on one channel
	for i := 0; i < b.N; i++ {
		memstat := timing.OneTestOfSubsMemory(1, 1000*1000)
		fmt.Println("bytes per subscription = ", (memstat.Bytes)/uint64(1000*1000)) // 391
	}
	//BenchmarkMakeChannels-8   	       1	1,506,747,961 ns/op	461,052,248 B/op	 8,043,321 allocs/op
	// 1500 ns each for 666,666 per sec.
}
