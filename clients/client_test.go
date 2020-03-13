package clients

import (
	"bufio"
	"crypto/rand"
	"net"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/awootton/knotfreeiot/tokens"
)

var wg = sync.WaitGroup{}

func startSockets(n int) {

	// be sure to
	// kubectl port-forward service/knotfreeaide 7465:7465

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			servAddr := "knotfree.io:7465"
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
				for {
					var b [1]byte
					_, _ = rand.Read(b[:])
					topic := "test_topic_" + strconv.FormatInt(int64(15&b[0]), 10)
					str = "P " + topic + " ,,,, hello_from_here"
					_, err = conn.Write([]byte(str + "\n"))
					if err != nil {
						println("Write to server failed:", err.Error())
						return // os.Exit(1)
					}
					time.Sleep(60 * time.Second)
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

func TestGrowGurus(t *testing.T) {

	if os.Getenv("KNOT_KUNG_FOO") == "atw" {
		startSockets(10)

		wg.Wait()
	}
}
