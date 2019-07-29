// Copyright 2019 Alan Tracey Wootton

package main

import (
	"fmt"
	"knotfreeiot/oldstuff/test/timing"
)

func main() {

	fmt.Println("hello measurements ")

	timing.ChanAndSubWithTCP(1000, 1000)

	//timing.MeasureChanAndSub(4, 4)

	// for i := 0; i < 10; i++ {
	// 	c := iot.NewConnection(nil, nil)
	// 	//	array[i] = &c
	// 	iot.RememberConnection(&c)
	// }

	//getMemstatsSubscribe()

	//memstat := timing.OneTestOfSubsMemory(1000*1000, 1)
	//fmt.Println("bytes per subscription = ", (memstat.Bytes)/uint64(1000*1000)) // 391

}
