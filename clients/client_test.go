package clients

import (
	"bufio"
	"crypto/rand"
	"fmt"
	"net"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/awootton/knotfreeiot/packets"
	"github.com/awootton/knotfreeiot/tokens"
)

var wg = sync.WaitGroup{}

func TestGrowGurus(t *testing.T) {

	if os.Getenv("KNOT_KUNG_FOO") == "atw" {
		//
		//	startSockets(10)

		startSockets8384(10)

		wg.Wait()
	}
}

func startSockets8384(n int) {

	// be sure to
	// kubectl port-forward service/knotfreeaide 8384:8384

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			servAddr := "knotfree.net:8384"
			tcpAddr, err := net.ResolveTCPAddr("tcp", servAddr)
			if err != nil {
				println("ResolveTCPAddr failed:", err.Error())
				return // os.Exit(1)
			}
		top:
			println("Dialing ", i)
			conn, err := net.DialTCP("tcp", nil, tcpAddr)
			if err != nil {
				println("Dial failed:", err.Error())
				return //os.Exit(1)
			}

			go func() {
				//str := "C token " + tokens.SampleSmallToken
				connect := &packets.Connect{}
				connect.SetOption("token", []byte(tokens.SampleSmallToken))
				err = connect.Write(conn)
				if err != nil {
					println("Write to server failed:", err.Error())
					return //os.Exit(1)
				}
				for i := 0; i < 10; i++ {
					var b [1]byte
					_, _ = rand.Read(b[:])
					topic := "test_topic_" + strconv.FormatInt(int64(15&b[0]), 10)
					//str = "S " + topic
					//_, err = conn.Write([]byte(str + "\n"))
					sub := &packets.Subscribe{}
					sub.Address.FromString(topic)
					sub.Write(conn)
					if err != nil {
						println("Write to server failed:", err.Error())
						return // os.Exit(1)
					}
				}
				count := 0
				for {
					var b [1]byte
					_, _ = rand.Read(b[:])
					topic := "test_topic_" + strconv.FormatInt(int64(15&b[0]), 10)
					//str = "P " + topic + " ,,,, hello_from_here" + strconv.FormatInt(count, 10)
					//str = fmt.Sprintf("P %v ,,,, hello_from_here%v ", topic, count)
					//_, err = conn.Write([]byte(str + "\n"))

					str := fmt.Sprintf("your message here %v", count)

					pub := &packets.Send{}
					pub.Address.FromString(topic)
					pub.Payload = []byte(str)
					pub.Write(conn)

					if err != nil {
						println("Write to server failed:", err.Error())
						return // os.Exit(1)
					}
					time.Sleep(60 * time.Second)
					count++
				}

			}()

			for {
				//	str, err := lineReader.ReadString('\n')
				p, err := packets.ReadPacket(conn)
				if err != nil {
					println("client err:", err.Error())
					goto top
					//return
				}
				println("client got:", p.String())
			}

		}(i)
	}

}

func startSockets(n int) {

	// be sure to
	// kubectl port-forward service/knotfreeaide 7465:7465

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			servAddr := "knotfree.net:7465"
			tcpAddr, err := net.ResolveTCPAddr("tcp", servAddr)
			if err != nil {
				println("ResolveTCPAddr failed:", err.Error())
				return // os.Exit(1)
			}
		top:
			println("Dialing ", i)
			conn, err := net.DialTCP("tcp", nil, tcpAddr)
			if err != nil {
				println("Dial failed:", err.Error())
				return //os.Exit(1)
			}

			go func() {
				str := "C token " + tokens.SampleSmallToken
				_, err = conn.Write([]byte(str + "\n"))
				if err != nil {
					println("Write to server failed:", err.Error())
					return //os.Exit(1)
				}
				for i := 0; i < 10; i++ {
					var b [1]byte
					_, _ = rand.Read(b[:])
					topic := "test_topic_" + strconv.FormatInt(int64(15&b[0]), 10)
					str = "S " + topic
					_, err = conn.Write([]byte(str + "\n"))
					if err != nil {
						println("Write to server failed:", err.Error())
						return // os.Exit(1)
					}
				}
				count := 0
				for {
					var b [1]byte
					_, _ = rand.Read(b[:])
					topic := "test_topic_" + strconv.FormatInt(int64(15&b[0]), 10)
					str = fmt.Sprintf("P %v ,,,, hello_from_here%v ", topic, count)
					_, err = conn.Write([]byte(str + "\n"))
					if err != nil {
						println("Write to server failed:", err.Error())
						return // os.Exit(1)
					}
					time.Sleep(60 * time.Second)
					count++
				}

			}()
			lineReader := bufio.NewReader(conn)
			for {
				str, err := lineReader.ReadString('\n')
				if err != nil {
					println("client err:", err.Error())
					goto top
					//return
				}
				println("client got:", str)
			}
		}(i)
	}

}
