package clients

import (
	"knotfree/iot"
	"knotfree/protocolaa"
	"math/rand"
	"net"
	"strconv"
	"time"
)

// func writePublish(conn net.Conn, realTopicName string, message string) error {

// 	// err := writeProtocolAaStr(conn, "t"+realTopicName)
// 	// if err != nil {
// 	// 	return err
// 	// }
// 	// err = writeProtocolAaStr(conn, "p"+message)
// 	// if err != nil {
// 	// 	return err
// 	// }
// 	return nil
// }

// func writeSubscribe(conn net.Conn, id string) error {

// 	// err := writeProtocolAaStr(conn, "s"+id)
// 	// if err != nil {
// 	// 	return err
// 	// }
// 	return nil
// }

func moreBackoff(backoff int) int {
	if backoff >= 512 { // 1024 seconds is 17 minutes
		return backoff
	}
	backoff = backoff * (200 + int(rand.Int31n(100))) / 150
	return backoff
}

// LightSwitch -  a light switch.
// connect, send contract, subscribe.
// timeout after 20 minutes. keep trying.
// We'll spawn a thread to write every 19 min.
// add 127.0.0.1 knotfreeserver to /etc/hosts
func LightSwitch(mySubChan string, ourSwitch string) {

	connectStr := "knotfreeserver:6161"
	on := false
	_ = on
	backoff := 2
	for {
		//fmt.Println("dialing")
		conn, err := net.DialTimeout("tcp", connectStr, 60*time.Second)
		if err != nil {
			clientLogThing.Collect("sleeping " + strconv.Itoa(backoff))
			time.Sleep(time.Duration(backoff) * time.Second)
			backoff = moreBackoff(backoff)
			continue // try to connect again
		}
		defer conn.Close()
		backoff = 2
		clientLogThing.Collect("LightSwitch dialed in")

		handler := protocolaa.NewHandler(conn.(*net.TCPConn))

		_ = handler

		sub := protocolaa.Subscribe{Msg: mySubChan}
		handler.Push(&sub)

		//lastTopicReceived := "none" // there''s only one topic so this is dumb deleteme:
		for {

			got, err := handler.Pop() // blocks
			if err != nil {
				clientLogThing.Collect("LightSw read err " + err.Error())
				conn.Close()
				break // and reconnect
			}
			switch got.(type) {
			case *protocolaa.SetTopic:
				//lastTopicReceived = got.(*protocolaa.SetTopic).Msg
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
	}
}

var count int64

// LightController -  switches a light switch.
// add 127.0.0.1 knotfreeserver to /etc/hosts
func LightController(id string, target string) {

	connectStr := "knotfreeserver:6161"
	on := false
	_ = on
	backoff := 2
	for {
		//fmt.Println("dialing")
		conn, err := net.DialTimeout("tcp", connectStr, 60*time.Second)
		if err != nil {
			time.Sleep(time.Duration(backoff) * time.Second)
			backoff = moreBackoff(backoff)
			continue
		}
		backoff = 2
		clientLogThing.Collect("LightCon dialed in")
		defer conn.Close()

		handler := protocolaa.NewHandler(conn.(*net.TCPConn))

		// Don't publish until after the light has subscribed
		time.Sleep(1 * time.Second)

		st := protocolaa.SetTopic{Msg: target}
		handler.Push(&st)

		str := "hello from elsewhere" + strconv.FormatInt(count, 10)
		pu := protocolaa.Publish{Msg: str}
		handler.Push(&pu)

		for {
			got, err := handler.Pop() // blocks
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
				// nothing
			}

		}
	}
}

var clientLogThing *iot.StringEventAccumulator

func init() {
	clientLogThing = iot.NewStringEventAccumulator(16)
}
