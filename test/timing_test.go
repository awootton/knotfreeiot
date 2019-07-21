// Copyright 2019 Alan Tracey Wootton

package main_test

import (
	"knotfree/iot"
	"testing"
)

func BenchmarkMakeChanObjects(b *testing.B) {

	//array := make([]types.ConnectionIntf, b.N, b.N)
	for i := 0; i < b.N; i++ {
		c := iot.NewConnection(nil, nil)
		//	array[i] = &c
		iot.RememberConnection(&c)
	}
	// md5:     BenchmarkMakeChanObjects-8   	 1000000	      1952 ns/op	     442 B/op	       8 allocs/op
	// highway: BenchmarkMakeChanObjects-8   	 1000000	      1484 ns/op	     554 B/op	       8 allocs/op
}
