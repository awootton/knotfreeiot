package clients

import (
	"knotfree/protocolaa"
	"knotfree/types"
	"math/rand"
	"net"
	"reflect"
	"strconv"
	"time"
)

func moreBackoff(backoff int) int {
	if backoff >= 512 { // 1024 seconds is 17 minutes
		return backoff
	}
	backoff = backoff * (200 + int(rand.Int31n(100))) / 150
	return backoff
}

// allTheClientConnections is the set of all connections here.
// since we only want the len and there's never a delete...
var allTheClientConnections = 0 // make(map[types.HashType]bool) a set

// LightSwitch -  a light switch.
// connect, send contract, subscribe.
// timeout after 20 minutes. keep trying.
// We'll spawn a thread to write every 19 min.
// add 127.0.0.1 knotfreeserver to /etc/hosts
func LightSwitch(mySubChan string, ourSwitch string) {

	time.Sleep(time.Duration(rand.Intn(60)) * time.Second)

	// randomStr := strconv.FormatInt(rand.Int63(), 16) + strconv.FormatInt(rand.Int63(), 16)
	// myKey := types.HashType{}
	// myKey.FromString(randomStr)
	allTheClientConnections++ //[myKey] = true

	connectStr := "knotfreeserver:6161"
	on := false
	_ = on
	backoff := 2
	for {
		conn, err := net.DialTimeout("tcp", connectStr, 60*time.Second)
		if err != nil {
			clientLogThing.Collect("ls sleeping  " + strconv.Itoa(backoff))
			time.Sleep(time.Duration(backoff) * time.Second)
			backoff = moreBackoff(backoff)
			continue // try to connect again
		}
		defer conn.Close()

		tcpConn := conn.(*net.TCPConn)
		err = tcpConn.SetReadBuffer(4096)
		if err != nil {
			clientLogThing.Collect("cl err " + err.Error())
			continue
		}
		err = tcpConn.SetWriteBuffer(4096)
		if err != nil {
			clientLogThing.Collect("cl err3 " + err.Error())
			continue
		}
		err = tcpConn.SetReadDeadline(time.Now().Add(20 * time.Minute))
		if err != nil {
			clientLogThing.Collect("cl err2 " + err.Error())
			continue
		}

		backoff = 2
		clientLogThing.Collect("LightSwitch dialed in")

		handler := protocolaa.NewHandler(conn.(*net.TCPConn))

		_ = handler

		sub := protocolaa.Subscribe{Msg: mySubChan}
		handler.Push(&sub)

		lastTopicReceived := "none" // there''s only one topic so this is dumb deleteme:
		for {

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
}

// LightController -  switches a light switch.
// add 127.0.0.1 knotfreeserver to /etc/hosts
func LightController(id string, target string) {

	time.Sleep(time.Duration(rand.Intn(60)) * time.Second)

	allTheClientConnections++ //[myKey] = true

	connectStr := "knotfreeserver:6161"
	on := false
	_ = on
	backoff := 2
	for { // forever
		conn, err := net.DialTimeout("tcp", connectStr, 60*time.Second)
		if err != nil {
			clientLogThing.Collect("lc sleeping  " + strconv.Itoa(backoff))
			time.Sleep(time.Duration(backoff) * time.Second)
			backoff = moreBackoff(backoff)
			continue
		}
		backoff = 2
		clientLogThing.Collect("LightCon dialed in")
		// defer func() { // we never quit
		// 	allTheClientConnections--
		defer conn.Close()
		// }()

		tcpConn := conn.(*net.TCPConn)
		err = tcpConn.SetReadBuffer(4096)
		if err != nil {
			clientLogThing.Collect("cl err3 " + err.Error())
			continue
		}
		err = tcpConn.SetWriteBuffer(4096)
		if err != nil {
			clientLogThing.Collect("cl err3 " + err.Error())
			continue
		}
		err = tcpConn.SetReadDeadline(time.Now().Add(20 * time.Minute))
		if err != nil {
			clientLogThing.Collect("cl err4 " + err.Error())
			continue
		}

		handler := protocolaa.NewHandler(conn.(*net.TCPConn))

		// Don't publish until after the light has subscribed
		var count int64
		go func() {
			for {
				st := protocolaa.SetTopic{Msg: target}
				handler.Push(&st)

				str := "hello from elsewhere" + strconv.FormatInt(count, 10)
				count++
				pu := protocolaa.Publish{Msg: str}
				handler.Push(&pu)
				time.Sleep(time.Duration(60+rand.Intn(60)) * time.Second)
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

			default:
				sss := reflect.TypeOf(got).String()
				clientLogThing.Collect("lcprob" + sss[len(sss)-6:])
			}

		}
	}
}

var aaclientRepofrter = func(seconds float32) []string {
	strlist := make([]string, 0, 5)
	size := allTheClientConnections
	strlist = append(strlist, "Client count="+strconv.Itoa(size))
	return strlist
}

var clientLogThing *types.StringEventAccumulator

func init() {
	clientLogThing = types.NewStringEventAccumulator(16)
	clientLogThing.SetQuiet(true)
	types.NewGenericEventAccumulator(aaclientRepofrter)
}
