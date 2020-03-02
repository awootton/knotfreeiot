package clients

import (
	"bufio"
	"crypto/rand"
	"net"
	"os"
	"strconv"
	"sync"
	"testing"

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
			servAddr := "localhost:7465"
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

			str := "C " + tokens.SampleSmallToken
			_, err = conn.Write([]byte(str))
			if err != nil {
				println("Write to server failed:", err.Error())
				os.Exit(1)
			}
			var b [1]byte
			_, _ = rand.Read(b[:])

			for i := 0; i < 10; i++ {
				str = "S test_topic_" + strconv.FormatInt(int64(127&b[0]), 10)
				_, err = conn.Write([]byte(str))
				if err != nil {
					println("Write to server failed:", err.Error())
					os.Exit(1)
				}
			}

			for {
				lineReader := bufio.NewReader(conn)
				str, err := lineReader.ReadString('\n')
				if err != nil {
					goto top
					//return
				}
				println("client got:", str, err.Error())
			}
		}(i)
	}

}

func TestGrowGurus(t *testing.T) {

	if os.Getenv("KUBE_EDITOR") == "atom --wait" {
		startSockets(25)

		wg.Wait()
	}
}
