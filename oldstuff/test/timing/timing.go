// Copyright 2019 Alan Tracey Wootton

package timing

import (
	"bufio"
	"fmt"
	"knotfree/oldstuff/iot"
	"knotfree/oldstuff/types"
	"math"
	"net"
	"reflect"
	"runtime"
	"strconv"
	"sync"
	"time"
)

type floatList []float32

const ever = true

// ChanAndSubWithTCP like MeasureChanAndSub but with sockets.
// we're only sending one message, one publish, but sockets have to come up and connect.
func ChanAndSubWithTCP(chanCount, testCount int) {

	iot.ResetAllTheConnectionsMap(chanCount)

	subMgr := iot.NewPubsubManager(testCount)

	chanArray := makeChanArray(chanCount, subMgr, false)

	for i := 0; i < testCount; i++ {
		c := chanArray[i%chanCount]
		topic := "chan/test/hello7" + strconv.Itoa(i)
		topicHash := types.HashType{}
		topicHash.FromString(topic)
		subMgr.SendSubscriptionMessage(&topicHash, topic, c, nil)
	}

	ln, err := net.Listen("tcp", "localhost:6161")
	if err != nil {
		// handle error
		fmt.Println("net.Listen oops1")
		return
	}
	defer ln.Close()

	done := false // make(chan bool)
	var waitgroup sync.WaitGroup

	// we're going to wire this up using tcp:
	// obj := <-handler.wire.west
	// handler.wire.east <- obj

	go func() {
		for ever {
			conn, err := ln.Accept()
			if err != nil {
				if !done {
					fmt.Println("net.Accept oops2")
				}
				return
			}
			go func(conn net.Conn) {
				defer conn.Close()
				reader := bufio.NewReader(conn)
				err := types.SocketSetup(conn)
				if err != nil {
					fmt.Println("SocketSetup problem3", err)
					return
				}
				_ = conn.SetDeadline(time.Now().Add(20000 * time.Millisecond))
				message, err := reader.ReadString('\n')
				// find index from sender and get a connection
				index, err := strconv.ParseInt(message[:len(message)-1], 10, 32)
				if err != nil {
					fmt.Println("strconv prob ", message, err)
				}
				c := chanArray[int(index)%chanCount]
				c.SetTCPConn(conn.(*net.TCPConn))
				h := c.GetProtocolHandler()
				handler, ok := h.(*TestProtocolHandler)
				_ = ok
				if handler.index != int(index) {
					fmt.Println("index problem2")
				}
				trace("east connected to tcp read", index)
				waitgroup.Done()
				for {
					message, err := reader.ReadString('\n')
					if err != nil || len(message) < 1 {
						if !done {
							fmt.Println("ReadString bad read", string(message), err)
						}
						return
					}
					message = message[:len(message)-1]
					trace("read str pushing to east:", message, index)
					handler.wire.east <- message
				}
			}(conn)
		}
	}()

	for i := 0; i < testCount; i++ {
		c := chanArray[i%chanCount]
		waitgroup.Add(1)
		go func(c types.ConnectionIntf, index int) {
			conn, err := net.DialTimeout("tcp", "localhost:6161", 10000*time.Millisecond)
			if err != nil {
				fmt.Println("dial fail", err)
				return
			}
			defer conn.Close()
			writer := bufio.NewWriterSize(conn, 128)
			n, err := writer.WriteString(strconv.Itoa(index) + "\n")
			trace("wrote ", index, n, err)
			_ = n
			_ = err
			h := c.GetProtocolHandler()
			handler, ok := h.(*TestProtocolHandler)
			_ = ok
			if handler.index != index {
				fmt.Println("index problem")
			}
			writer.Flush()
			for {
				var obj string
				trace("waiting on west to do tcp write ", handler.index)
				select {
				case obj = <-handler.wire.west:
				case <-time.After(30 * time.Second):
					fmt.Println("waited too long for handler.wire.west")
					return
				}
				trace("got from west ", obj)
				n, err := writer.WriteString(obj)
				if err != nil {
					fmt.Println("died on write ", err)
					return
				}
				n, err = writer.WriteString("\n")
				if err != nil {
					fmt.Println("died on write2 ", err)
					return
				}
				writer.Flush()
				_ = n
				_ = err
				trace("wrote data ", obj, n, err)
			}

		}(c, i)
	}

	waitgroup.Wait()
	fmt.Println("now they are all dialed in.")
	for i := 0; i < testCount; i++ {
		c := chanArray[i%chanCount]
		// we can't publish to ourself so publish to the next guy.
		topic := "chan/test/hello7" + strconv.Itoa((i+1)%testCount)
		topicHash := types.HashType{}
		topicHash.FromString(topic)
		message := []byte("can you hear me? " + strconv.Itoa(i))
		trace("p1", i)
		subMgr.SendPublishMessage(&topicHash, c, &message)
	}
	fmt.Println("messages published.")

	var waitgroup2 sync.WaitGroup
	for i := 0; i < testCount; i++ {
		c := chanArray[i%chanCount]
		handler := c.GetProtocolHandler()
		waitgroup2.Add(1)
		go func(handler types.ProtocolHandlerIntf, index int) {
			//var h types.ProtocolHandler
			h, bad := handler.(*TestProtocolHandler)
			if !bad {
				//panic("cast should work")
				fmt.Println(reflect.TypeOf(handler), " Should be TestProtocolHandler")
			}
			{
				trace("w1", index)
				str, _ := h.Pop(time.Second)
				trace("w2", index)
				message := ("can you hear me? " + strconv.Itoa((index+testCount-1)%testCount))
				if str != message {
					fmt.Println("I got ", str, " wanted ", message)
				}
			}
			if 1 == 0-2 { // we only published once
				trace("w3", index)
				str, _ := h.Pop(time.Second)
				trace("w4", index)
				message := ("how about now " + strconv.Itoa((index+testCount-1)%testCount))
				if str != message {
					trace("I got ", str, " wanted ", message)
				}
			}
			waitgroup2.Done()
		}(handler, i)
	}
	fmt.Println("waiting...")
	waitgroup2.Wait()

	done = true
	ln.Close()
	for i := 0; i < testCount; i++ {
		c := chanArray[i%chanCount]
		c.Close()
	}
	fmt.Println("done")

}

// MeasureChanAndSub is an example
// and a test and a benchmark
// fails with less than 3,3
func MeasureChanAndSub(chanCount, testCount int) {

	iot.ResetAllTheConnectionsMap(chanCount)

	subMgr := iot.NewPubsubManager(testCount)

	chanArray := makeChanArray(chanCount, subMgr, true)

	for i := 0; i < testCount; i++ {
		c := chanArray[i%chanCount]
		topic := "chan/test/hello7" + strconv.Itoa(i)
		topicHash := types.HashType{}
		topicHash.FromString(topic)
		subMgr.SendSubscriptionMessage(&topicHash, topic, c, nil)
	}

	for i := 0; i < testCount; i++ {
		c := chanArray[i%chanCount]
		// we can't publish to ourself so publish to the next guy.
		topic := "chan/test/hello7" + strconv.Itoa((i+1)%testCount)
		topicHash := types.HashType{}
		topicHash.FromString(topic)
		message := []byte("can you hear me? " + strconv.Itoa(i))
		//fmt.Println("p1", i)
		subMgr.SendPublishMessage(&topicHash, c, &message)
	}

	var waitgroup sync.WaitGroup
	for i := 0; i < testCount; i++ {
		c := chanArray[i%chanCount]
		handler := c.GetProtocolHandler()
		waitgroup.Add(1)
		go func(handler types.ProtocolHandlerIntf, index int) {
			//var h types.ProtocolHandler
			h, bad := handler.(*TestProtocolHandler)
			if !bad {
				//panic("cast should work")
				fmt.Println(reflect.TypeOf(handler), " Should be TestProtocolHandler")
			}
			{
				//fmt.Println("w1", index)
				str, _ := h.Pop(time.Second)
				//fmt.Println("w2", index)
				message := ("can you hear me? " + strconv.Itoa((index+testCount-1)%testCount))
				if str != message {
					fmt.Println("I got ", str, " wanted ", message)
				}
			}
			{
				//fmt.Println("w3", index)
				str, _ := h.Pop(time.Second)
				//fmt.Println("w4", index)
				message := ("how about now " + strconv.Itoa((index+testCount-1)%testCount))
				if str != message {
					fmt.Println("I got ", str, " wanted ", message)
				}
			}
			waitgroup.Done()
		}(handler, i)
	}
	fmt.Println("waiting...")

	for i := 0; i < testCount; i++ {
		c := chanArray[i%chanCount]
		// we can't publish to ourself so publish to the next guy.
		// and the one before us will publish to us.
		topic := "chan/test/hello7" + strconv.Itoa((i+1)%testCount)
		topicHash := types.HashType{}
		topicHash.FromString(topic)
		message := []byte("how about now " + strconv.Itoa(i))
		//fmt.Println("p2", i)
		subMgr.SendPublishMessage(&topicHash, c, &message)
	}

	waitgroup.Wait()
	fmt.Println("done")

	for i := 0; i < testCount; i++ {
		c := chanArray[i%chanCount]
		c.Close()
	}

	amt := iot.GetAllConnectionsSize()
	if amt != 0 {
		fmt.Println("close is supposed to reclaim")
	}
}

func getMemstatsSubscribe() {
	types.DoStartEventCollectorReporting = false

	spacing := 80000 // 40000

	var statsPerChanSetting []floatList

	for chanCount := 1; chanCount < 500000; chanCount += spacing {

		var byteCountDeltas []float32
		var prevMemstat *Memstat

		for subscriptionCount := 1; subscriptionCount < 500000; subscriptionCount += spacing {

			memstat := OneTestOfSubsMemory(subscriptionCount, chanCount)

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
	const results = "[[368.8352 367.0268 483.7704 226.9258 328.4918 663.955] [597.3554 307.9092 365.294 227.2264 304.9664 451.6732] [597.3216 595.943 365.3756 227.1292 305.0394 453.104] [597.3726 595.8972 652.546 227.7404 305.3434 451.4576] [597.3674 595.8972 652.3802 516.1012 305.0198 452.6458] [597.3174 595.9472 653.3434 515.2368 592.5818 450.8216] [597.3226 595.9432 652.7964 515.6604 592.6182 739.258]]"
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

// OneTestOfSubsMemory is probably slow
func OneTestOfSubsMemory(subscriptionCount int, chanCount int) Memstat {

	types.DoStartEventCollectorReporting = false

	iot.ResetAllTheConnectionsMap(chanCount)

	subMgr := iot.NewPubsubManager(subscriptionCount)

	chanArray := makeChanArray(chanCount, subMgr, false)

	runtime.GC()
	runtime.GC()

	stat := Memstat{}
	memStatsBefore := PrintMemUsage(nil, &stat)

	for i := 0; i < subscriptionCount; i++ {
		c := chanArray[i%chanCount]
		topic := "chan" + strconv.Itoa(i)
		topicHash := types.HashType{}
		topicHash.FromString(topic)
		subMgr.SendSubscriptionMessage(&topicHash, topic, c, nil)
	}

	runtime.GC()
	runtime.GC()

	//fmt.Print("subscriptionCount = ", subscriptionCount)
	//fmt.Print("\tchanCount = ", chanCount)
	PrintMemUsage(memStatsBefore, &stat)

	// add subscriptions.
	return stat
}

func makeChanArray(amt int, subscribeMgr types.SubscriptionsIntf, wireEndsTogether bool) []types.ConnectionIntf {
	array := make([]types.ConnectionIntf, amt, amt)

	for i := 0; i < amt; i++ {
		c := iot.NewConnection(nil, subscribeMgr)
		array[i] = c
		iot.RememberConnection(c)

		handler := NewTestProtocolHandler(i)
		c.SetProtocolHandler(handler)

		if wireEndsTogether {
			handlerStruct, _ := handler.(*TestProtocolHandler)
			// make an 'echo' for a client like this:
			go func(handler *TestProtocolHandler) {
				for {
					obj := <-handler.wire.west
					handler.wire.east <- obj
				}
			}(handlerStruct)
		}

	}
	return array
}

// Memstat for local use
type Memstat struct {
	Bytes   uint64 // how many bytes allocated
	Objects uint64 // how many objects allocated
	NumGC   uint32 // how many GC
}

// PrintMemUsage outputs the current, total and OS memory being used. As well as the number
// of garage collection cycles completed.
func PrintMemUsage(before *runtime.MemStats, stat *Memstat) *runtime.MemStats {

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

// const doTrace = false

// func trace(a ...interface{}) {
// 	if doTrace {
// 		fmt.Println(a...)
// 	}
// }
