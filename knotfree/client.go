package knotfree

import (
	"encoding/json"
	"net"
	"strconv"
	"time"
)

var scale = (time.Second * 10) // time.Minute // time.Second // time.Minute

var clientLogThing *StringEventAccumulator

func init() {
	clientLogThing = NewStringEventAccumulator(16)
	clientLogThing.quiet = true
}

// func writeOn(conn net.Conn, on bool) error {

// 	err := WriteProtocolA(conn, "p"+`{}`)
// 	if err != nil {
// 		return err
// 	}
// 	return nil
// }

func writePublish(conn net.Conn, realTopicName string, message string) error {

	pp := PublishProtocolA{}
	pp.T = realTopicName
	pp.M = message
	bytes, err := json.Marshal(pp)
	if err != nil {
		return err
	}
	err = WriteProtocolA(conn, "p"+string(bytes))
	if err != nil {
		return err
	}
	return nil
}

func writeSubscribe(conn net.Conn, id string) error {

	err := WriteProtocolA(conn, "s"+id)
	if err != nil {
		return err
	}
	return nil
}

// LightSwitch -  a light switch.
// connect, send contract, subscribe.
// timeout after 20 minutes. keep trying.
// We'll spawn a thread to write every 19 min.
// usually "localhost:8080"
func LightSwitch(id string) {

	connectStr := "localhost:8080"
	on := false
	_ = on
	for {
		//fmt.Println("dialing")
		conn, err := net.DialTimeout("tcp", connectStr, scale*10)
		if err != nil {
			time.Sleep(10 * time.Second)
			continue
		}
		defer conn.Close()
		clientLogThing.Collect("LightSwitch dialed in")

		err = writeSubscribe(conn, id)
		if err != nil {
			break
		}

		// go func() {
		// 	//fmt.Println("start write loop")
		// 	for {
		// 		time.Sleep(19 * time.Minute)
		// 		err := write(conn, on)
		// 		if err != nil {
		// 			//fmt.Println("client write err " + err.Error())
		// 			conn.Close()
		// 			break
		// 		}
		// 	}
		// }()

		//fmt.Println("Connected " + id)

		bytes := make([]byte, 256)
		for {
			err = conn.SetReadDeadline(time.Now().Add(20 * scale))
			str, err := ReadProtocolA(conn, bytes)
			clientLogThing.Collect("LightSw pkt") // + str)
			clientLogThing.Sum("LightSw r bytes", len(str))
			if err != nil {
				clientLogThing.Collect("LightSw read err " + err.Error())
				conn.Close()
				break
			}

			//time.Sleep(1000 * time.Millisecond)

		}
	}
}

var count int64

// LightController -  switches a light switch.
func LightController(id string, target string) {

	connectStr := "localhost:8080"
	on := false
	_ = on
	for {
		//fmt.Println("dialing")
		conn, err := net.DialTimeout("tcp", connectStr, scale*10)
		if err != nil {
			time.Sleep(10 * time.Second)
			continue
		}
		clientLogThing.Collect("LightCon dialed in")
		defer conn.Close()

		go func() {
			//fmt.Println("start write loop")
			for {
				time.Sleep(5 * scale) // rand.Intn(15)

				str := "hello from elsewhere" + strconv.FormatInt(count, 10)
				clientLogThing.Sum("LightCo w bytes", len(str))
				err := writePublish(conn, target, str)
				count++
				if err != nil {
					clientLogThing.Collect("LightCon w err " + err.Error())
					conn.Close()
					break
				}
			}
		}()

		//fmt.Println("Connected " + id)

		bytes := make([]byte, 256)
		for {
			err = conn.SetReadDeadline(time.Now().Add(35 * scale))
			n, err := conn.Read(bytes) // blocks
			clientLogThing.Collect("LightCo pkt:" + string(bytes))
			clientLogThing.Sum("LightCo r bytes", n)
			if err != nil {
				clientLogThing.Collect("LightCon r err " + err.Error())
				conn.Close()
				break
			}
		}
	}
}
