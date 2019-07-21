// Copyright 2019 Alan Tracey Wootton

package main

import (
	"fmt"
	"knotfree/iot"
	"knotfree/types"
	"math"
	"runtime"
	"strconv"
)

func main() {

	fmt.Println("hello measurements ")

	for i := 0; i < 10; i++ {
		c := iot.NewConnection(nil, nil)
		//	array[i] = &c
		iot.RememberConnection(&c)
	}

	getMemstatsSubscribe()

}

type floatList []float32

func getMemstatsSubscribe() {
	types.DoStartEventCollectorReporting = false

	spacing := 80000 // 40000

	var statsPerChanSetting []floatList

	for chanCount := 1; chanCount < 500000; chanCount += spacing {

		var byteCountDeltas []float32
		var prevMemstat *memstat

		for subscriptionCount := 1; subscriptionCount < 500000; subscriptionCount += spacing {

			memstat := oneTestOfSubsMemory(subscriptionCount, chanCount)

			//fmt.Println("memstat.Bytes", memstat.Bytes)

			if prevMemstat != nil {
				fmt.Println("bytes per subscription = ", (memstat.Bytes-prevMemstat.Bytes)/uint64(spacing))
				byteCountDeltas = append(byteCountDeltas, float32(memstat.Bytes-prevMemstat.Bytes)/float32(spacing))
			}

			//fmt.Print("subscriptionCount = ", subscriptionCount)
			//fmt.Print("\tchanCount = ", chanCount)

			// bytes, err := json.Marshal(memstat)
			// _ = err
			// fmt.Println(string(bytes))
			prevMemstat = &memstat
		}
		avg, std := findAverage(byteCountDeltas)
		fmt.Println("Average bytes per subscription, std ", avg, std)
		// 662 when chanCount = 1000

		statsPerChanSetting = append(statsPerChanSetting, byteCountDeltas)
	}
	fmt.Println("all the stats ", statsPerChanSetting)
}

func findAverage(nums []float32) (float32, float32) {
	sum := float32(0)
	for _, num := range nums {
		sum += num
	}
	avg := sum / float32(len(nums))
	stddev := float64(0)
	for _, num := range nums {
		stddev += math.Sqrt(float64((num - avg) * (num - avg)))
	}
	// actually std dev squared
	stddev = stddev / float64(len(nums))

	sum = float32(0)
	// reject values greater than 1 sigma
	// get new list and new avg
	var newlist []float32
	for _, num := range nums {
		x := math.Sqrt(float64((num - avg) * (num - avg)))
		if x < stddev {
			sum += num
			newlist = append(newlist, num)
		}
	}
	if len(newlist)*2 < len(nums) {
		fmt.Println("BAD DATA - more than half is outside  sigma ")
	}
	fmt.Println("N is ", len(newlist), " of ", len(nums))
	avg = sum / float32(len(newlist))

	stddev = float64(0)
	sum = float32(0)
	for _, num := range newlist {
		sum += num
		stddev += math.Sqrt(float64((num - avg) * (num - avg)))
	}
	avg = sum / float32(len(newlist))
	stddev = stddev / float64(len(newlist))
	return avg, float32(stddev)
}

func oneTestOfSubsMemory(subscriptionCount int,
	chanCount int) memstat {

	types.DoStartEventCollectorReporting = false

	iot.ResetAllTheConnectionsMap()

	subMgr := iot.NewPubsubManager()

	chanArray := makeChanArray(chanCount, subMgr)

	runtime.GC()
	runtime.GC()
	runtime.GC()
	runtime.GC()
	runtime.GC()
	runtime.GC()
	runtime.GC()
	runtime.GC()

	stat := memstat{}
	memStatsBefore := PrintMemUsage(nil, &stat)

	for i := 0; i < subscriptionCount; i++ {
		c := chanArray[i%chanCount]
		topic := "chan" + strconv.Itoa(i)
		topicHash := types.HashType{}
		topicHash.FromString(topic)
		subMgr.SendSubscriptionMessage(&topicHash, topic, c)
	}

	runtime.GC()
	runtime.GC()
	runtime.GC()
	runtime.GC()
	runtime.GC()
	runtime.GC()
	runtime.GC()
	runtime.GC()

	//fmt.Print("subscriptionCount = ", subscriptionCount)
	//fmt.Print("\tchanCount = ", chanCount)
	PrintMemUsage(memStatsBefore, &stat)

	// add subscriptions.
	return stat
}

func makeChanArray(amt int, subscribeMgr types.SubscriptionsIntf) []types.ConnectionIntf {
	array := make([]types.ConnectionIntf, amt, amt)
	for i := 0; i < amt; i++ {
		c := iot.NewConnection(nil, subscribeMgr)
		array[i] = &c
		iot.RememberConnection(&c)

	}
	return array
}

type memstat struct {
	Bytes   uint64 // how many bytes allocated
	Objects uint64 // how many objects allocated
	NumGC   uint32 // how many GC
}

// PrintMemUsage outputs the current, total and OS memory being used. As well as the number
// of garage collection cycles completed.
func PrintMemUsage(before *runtime.MemStats, stat *memstat) *runtime.MemStats {

	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	// For info on each, see: https://golang.org/pkg/runtime/#MemStats
	//fmt.Printf("Alloc = %v MiB", bToMb(m.Alloc))
	//fmt.Printf("\tTotalAlloc = %v MiB", bToMb(m.TotalAlloc))
	//fmt.Printf("\tSys = %v MiB", bToMb(m.Sys))
	//fmt.Printf("\tNumGC = %v\n", m.NumGC)

	// fmt.Printf("Alloc = %v MiB", bToMb(m.Alloc))
	// fmt.Printf("\tSys = %v MiB", bToMb(m.Sys))
	// fmt.Printf("\tDeltaObj = %v objects", m.Mallocs-m.Frees)
	// fmt.Printf("\tHeapObj = %v objects", m.HeapObjects)
	// fmt.Println()
	if before != nil {

		//	fmt.Printf("Bytes = %v KiB", bToKb(m.Alloc-before.Alloc))
		stat.Bytes = m.Alloc - before.Alloc
		//fmt.Printf("\tBytes = %v KiB", bToKb(stat.Bytes))

		stat.Objects = (m.Mallocs - m.Frees) - (before.Mallocs - before.Frees)
		//fmt.Printf("\tObjects = %v ", stat.Objects)

		stat.NumGC = m.NumGC - before.NumGC
		//fmt.Printf("\tNumGC = %v\n", m.NumGC-before.NumGC)

	}

	return &m
}

func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}

func bToKb(b uint64) uint64 {
	return b / 1024
}
