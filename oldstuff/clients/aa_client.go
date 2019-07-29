// Copyright 2019 Alan Tracey Wootton

package clients

import (
	"knotfreeiot/oldstuff/protocolaa"
	"knotfreeiot/oldstuff/types"
	"math/rand"
	"net"
	"reflect"
	"strconv"
	"sync/atomic"
	"time"
)

const testport = "knotfreeserver:6162"

var maxBackoff = 30 * 60 // is seconds

func moreBackoff(backoff int) int {
	if backoff >= maxBackoff {
		return backoff
	}
	backoff = backoff * (200 + int(rand.Int31n(100))) / 150
	return backoff
}

// ExpectedConnections so we can act differnetly during debugging
var ExpectedConnections = 0

// allTheClientConnections is the set of all connections here.
// since we only want the len and there's never a delete...
var allTheClientConnections = int32(0) // make(map[types.HashType]bool) a set

// LightSwitch -  a light switch.
// connect, send contract, subscribe.
// timeout after 20 minutes. keep trying.
// We'll spawn a thread to write every 19 min.
// add 127.0.0.1 knotfreeserver to /etc/hosts
func LightSwitch(mySubChan string, ourSwitch string) {

	if ExpectedConnections > 10 { // 60 sec * 30 = 1800 sec = 30 min
		time.Sleep(time.Duration(rand.Intn(60)) * time.Second * 30)
	}
	if ExpectedConnections == 1 {
		clientLogThing.SetQuiet(false)
	}

	atomic.AddInt32(&allTheClientConnections, 1)

	connectStr := testport
	on := false
	_ = on
	backoff := 2
	for {
		atomic.AddInt32(&allTheClientConnections, -1)
		conn, err := net.DialTimeout("tcp", connectStr, 60*time.Second)
		atomic.AddInt32(&allTheClientConnections, +1)
		if err != nil {
			clientLogThing.Collect("LightSwitch sleeping  " + strconv.Itoa(backoff))
			atomic.AddInt32(&allTheClientConnections, -1)
			time.Sleep(time.Duration(backoff) * time.Second)
			atomic.AddInt32(&allTheClientConnections, +1)
			backoff = moreBackoff(backoff)
			continue // try to connect again
		}
		defer conn.Close()

		if types.SocketSetup(conn) != nil {
			continue // try again
		}

		backoff = 2
		clientLogThing.Collect("LightSwitch dialed in")

		handler := protocolaa.NewHandler(conn.(*net.TCPConn))

		_ = handler

		sub := protocolaa.Subscribe{Msg: mySubChan}
		handler.Push(&sub)

		lastTopicReceived := "none" // there''s only one topic so this is dumb deleteme s
		for {
			// all error of any kind must propogate to Pop()
			// so they can be known
			got, err := handler.Pop(15 * time.Minute) // blocks
			if err != nil {
				clientLogThing.Collect("LightSw read err " + err.Error())
				conn.Close()
				break // and reconnect
			}
			switch got.(type) {
			case *protocolaa.SetTopic:
				lastTopicReceived = got.(*protocolaa.SetTopic).Msg
				continue
			case *protocolaa.Publish:
				what := got.(*protocolaa.Publish).Msg
				clientLogThing.Collect("LightSw received:" + what)
				// echo it back
				handler.Push(&protocolaa.SetTopic{Msg: ourSwitch})
				handler.Push(&protocolaa.Publish{Msg: what})

			default:
				// nothing
			}

		}
		_ = lastTopicReceived
	}
	//fmt.Println("NEVER SUPPOSED TO HAPPEN!")
}

// LightController -  switches a light switch.
// add 127.0.0.1 knotfreeserver to /etc/hosts
func LightController(id string, target string) {

	if ExpectedConnections > 10 {
		time.Sleep(time.Duration(rand.Intn(60)) * time.Second * 30)
	}

	atomic.AddInt32(&allTheClientConnections, 1)

	connectStr := testport
	on := false
	_ = on
	backoff := 2
	for { // forever
		atomic.AddInt32(&allTheClientConnections, -1)
		conn, err := net.DialTimeout("tcp", connectStr, 60*time.Second)
		atomic.AddInt32(&allTheClientConnections, 1)
		if err != nil {
			clientLogThing.Collect("LightCon sleeping  " + strconv.Itoa(backoff))
			atomic.AddInt32(&allTheClientConnections, -1)
			time.Sleep(time.Duration(backoff) * time.Second)
			atomic.AddInt32(&allTheClientConnections, +1)
			backoff = moreBackoff(backoff)
			continue
		}
		backoff = 2
		clientLogThing.Collect("LightCon dialed in")
		defer conn.Close() // never happens

		if types.SocketSetup(conn) != nil {
			continue // try again
		}

		handler := protocolaa.NewHandler(conn.(*net.TCPConn))
		sub := protocolaa.Subscribe{Msg: id}
		handler.Push(&sub)

		//	protocolaa.WriteStr(conn, "s"+sub.Msg)

		// Don't publish until after the light has subscribed
		var count int64
		expecting := "some hello message or something"
		when := time.Now()
		go func() {
			for {
				st := protocolaa.SetTopic{Msg: target}
				handler.Push(&st)
				expecting = "hello from elsewhere" + strconv.FormatInt(count, 10)
				count++
				pu := protocolaa.Publish{Msg: expecting}
				when = time.Now()
				handler.Push(&pu)
				if ExpectedConnections > 10 {
					time.Sleep(time.Duration(60+rand.Intn(60)) * time.Second * 10)
				} else {
					time.Sleep(10 * time.Second)
				}
			}
		}()

		for {
			got, err := handler.Pop(15 * time.Minute) // blocks
			if err != nil {
				clientLogThing.Collect("LightCon read err " + err.Error())
				conn.Close()
				break // and reconnect
			}
			switch got.(type) {
			case *protocolaa.SetTopic:
				continue
			case *protocolaa.Publish:
				what := got.(*protocolaa.Publish).Msg
				clientLogThing.Collect("LightCon received:" + what)
				if what != expecting {
					clientLogThing.Collect("customer not happy")
				}
				duration := time.Now().Sub(when)
				if duration > time.Second*10 {
					clientLogThing.Collect("customer bored")
				} else if duration < time.Millisecond*100 {
					clientLogThing.Collect("happy joy")
				} else {
					clientLogThing.Collect("ok")
				}

			default:
				sss := reflect.TypeOf(got).String()
				clientLogThing.Collect("lcprob" + sss[len(sss)-6:])
			}

		}
	}
	//fmt.Println("NEVER SUPPOSED TO HAPPEN!")
}

var aaclientReporter = func(seconds float32) []string {
	strlist := make([]string, 0, 5)
	size := allTheClientConnections
	strlist = append(strlist, "Client count="+strconv.FormatInt(int64(size), 10))
	return strlist
}

var clientLogThing *types.StringEventAccumulator

func init() {
	clientLogThing = types.NewStringEventAccumulator(16)
	clientLogThing.SetQuiet(true)
	types.NewGenericEventAccumulator(aaclientReporter)
}
