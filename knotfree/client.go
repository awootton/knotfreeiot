package knotfree

import (
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"time"
)

var scale = time.Second // time.Minute

func writeOn(conn net.Conn, on bool) error {

	err := WriteProtocolA(conn, "p"+`{}`)
	if err != nil {
		return err
	}

	// n, err := conn.Write([]byte("Client" + time.Now().String()))
	// if err != nil {
	// 	fmt.Println("client write err " + err.Error())
	// 	return err
	// }
	// _ = n
	return nil
}

func writePublish(conn net.Conn, realChannelName string, message string) error {

	pp := PublishProtocolA{}
	pp.C = realChannelName
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
		fmt.Println("LightSwitch dialed in")

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
			fmt.Println("LightSwitch got:" + str)
			if err != nil {
				fmt.Println("LightSwitch read err " + err.Error())
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
		fmt.Println("LightController dialed in")
		defer conn.Close()

		go func() {
			//fmt.Println("start write loop")
			for {
				time.Sleep(5 * scale) // rand.Intn(15)
				//	err := write(conn, on)
				err := writePublish(conn, target, "hello from elsewhere"+strconv.FormatInt(count, 10))
				count++
				if err != nil {
					fmt.Println("LightController write err " + err.Error())
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
			fmt.Println("LightController got:" + string(bytes))
			if err != nil {
				fmt.Println("LightController read err " + err.Error())
				conn.Close()
				break
			}
			_ = n
			//time.Sleep(1000 * time.Millisecond)

		}
	}
}
