// Copyright 2019,2020,2021,2022,2023,2024 Alan Tracey Wootton
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package iot_test

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/awootton/knotfreeiot/iot"
	"github.com/awootton/knotfreeiot/monitor_pod"
	"github.com/awootton/knotfreeiot/packets"
	"github.com/awootton/knotfreeiot/tokens"
)

func GetLocalIP() (string, string) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", ""
	}
	ipv4 := ""
	ipv6 := ""
	i := 0
	for _, address := range addrs {
		// fmt.Println("address", address)
		if ipv4 == "" {
			if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ipnet.IP.To4() != nil {
					ipv4 = ipnet.IP.String()
				}
			}
		}
		if ipnet6, ok := address.(*net.IPNet); ok && !ipnet6.IP.IsLoopback() {
			// fmt.Println(ipnet6.Mask)
			if ipnet6.IP.To16() != nil {
				if ipv6 == "" {
					ipv6 = ipnet6.IP.String()
				}
			}
		}
		i++
	}
	return ipv4, ipv6
}

func TestLookup(t *testing.T) {

	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	ip4, xxip6 := GetLocalIP()
	fmt.Println("localIP", ip4, xxip6)
	// TODO: get ipv 6 to work

	var port int
	addr := ":0"                               // ip4 + ":0"
	s, err := net.ResolveUDPAddr("udp4", addr) // this is goofy. just returns :0
	check(err)

	// sss, err := net.UDPAddrFromAddrPort(s, addr)

	ce, contacts, names := setup(t)
	_ = ce
	_ = names

	closed := make(chan interface{})
	defer close(closed)
	// let's listen on a udp port
	received := make(chan []byte)
	started := make(chan bool)
	go func() {
		// start a server
		// ln, err := net.Listen("tcp6", "[::]:1024")
		connection, err := net.ListenUDP("udp4", s)
		check(err)

		port = connection.LocalAddr().(*net.UDPAddr).Port
		addr = connection.LocalAddr().(*net.UDPAddr).AddrPort().String()
		fmt.Println("Using port:", port) // eg Using port: 62823
		fmt.Println("Using addr:", addr) // Using addr: 0.0.0.0:62823

		started <- true
		defer connection.Close()
		for {
			select {
			case <-closed:
				return
			default:
			}

			buffer := make([]byte, 1024)
			n, _, err := connection.ReadFromUDP(buffer)
			check(err)
			fmt.Println("received", string(buffer[:n]))
			received <- buffer[:n]
		}
	}()

	<-started // wait for the listener to start
	// sent it something
	{
		hostport := ip4 + ":" + fmt.Sprint(port) // ip4 ?
		s, err := net.ResolveUDPAddr("udp4", hostport)
		check(err)
		c, err := net.DialUDP("udp4", nil, s)
		check(err)
		fmt.Printf("The UDP server is %s\n", c.RemoteAddr().String())
		defer c.Close()

		data := []byte("dummy message")
		_, err = c.Write(data)
		check(err)
		fmt.Println("sent", string(data))
	}

	got := <-received

	fmt.Println("received", got)
	// makereader of got
	reader := bytes.NewReader(got)
	p, err := packets.ReadPacket(reader)

	fmt.Println("packet", p.Sig())

	_ = received

	look := packets.Lookup{}
	look.Address.FromString("contact-address-5")
	look.SetOption("debg", []byte("12345678"))
	hostport := ip4 + ":" + fmt.Sprint(port)
	look.SetOption("A", []byte(hostport))
	iot.PushPacketUpFromBottom(contacts[0], &look)

	bytes := <-received

	fmt.Println("received", bytes)

	_ = err
}

func setup(t *testing.T) (*iot.ClusterExecutive, []iot.ContactInterface, []string) {

	tokens.LoadPublicKeys()

	atoken := tokens.GetTest32xToken()
	atokenStruct := tokens.ParseTokenNoVerify(atoken)

	got := ""
	want := ""

	localtime := uint32(time.Now().Unix())
	getTime := func() uint32 {
		return localtime
	}
	var ce *iot.ClusterExecutive

	_ = captureStdout(func() {
		isTCP := true
		aideCount := 1
		ce = iot.MakeSimplestCluster(getTime, isTCP, aideCount, "")
		globalClusterExec = ce

		ce.WaitForActions()
	})

	contacts := make([]iot.ContactInterface, 10)
	names := make([]string, 10)
	for i := 0; i < 10; i++ {

		contact1 := makeTestContact(ce.Aides[0].Config, "")
		contacts[i] = contact1
		connect := packets.Connect{}
		connect.SetOption("token", atoken)
		iot.PushPacketUpFromBottom(contact1, &connect)

		// subscribe
		subs := packets.Subscribe{}
		names[i] = fmt.Sprint("contact-address-", i)
		subs.Address.FromString(names[i])
		// subs.SetOption("debg", []byte("12345678"))
		iot.PushPacketUpFromBottom(contact1, &subs)
		// fmt.Println("contact1 subscribed contact", contact1.GetKey().Sig())
		// fmt.Println("contact1 subscribed    subs", subs.Address.Sig())
	}

	ce.WaitForActions()

	// they should all have subacks now
	for i := 0; i < 10; i++ {
		contact1 := contacts[i]
		name := fmt.Sprint("contact-address-", i)
		var h iot.HashType
		h.HashBytes([]byte(name))
		hashedb64 := h.String()
		got, _ = contact1.(*testContact).popResultAsString() // the suback
		got = strings.Replace(got, atokenStruct.JWTID, "xxxx", 1)
		want = "[S,=" + hashedb64 + ",jwtid,xxxx,pub2self,0]" //"no message received"
		if got != want {
			t.Errorf("got %v, want %v", got, want)
		}
	}

	return ce, contacts, names
}

// These (below) could go to another file.

func TestSubDomain(t *testing.T) {

	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	tokens.LoadPublicKeys()

	atoken := tokens.GetTest32xToken()
	atokenStruct := tokens.ParseTokenNoVerify(atoken)

	got := ""
	want := ""
	ok := true
	var err error
	localtime := uint32(time.Now().Unix())
	getTime := func() uint32 {
		return localtime
	}
	var ce *iot.ClusterExecutive

	_ = captureStdout(func() {
		isTCP := true
		aideCount := 1
		ce = iot.MakeSimplestCluster(getTime, isTCP, aideCount, "")
		globalClusterExec = ce

		ce.WaitForActions()
	})
	contact0 := makeTestContact(ce.Aides[0].Config, "")
	if contact0.IsClosed() {
		fmt.Println("contact1 closed")
	}
	contact0.DoClose(nil)
	if contact0.IsClosed() {
		fmt.Println("contact1 closed")
	}
	contact0.DoClose(nil)
	if contact0.IsClosed() {
		fmt.Println("contact1 closed")
	}

	contact1 := makeTestContact(ce.Aides[0].Config, "")
	connect := packets.Connect{}
	connect.SetOption("token", atoken)
	iot.PushPacketUpFromBottom(contact1, &connect)

	// subscribe
	subs := packets.Subscribe{}
	subs.Address.FromString("contact1 address")
	subs.SetOption("debg", []byte("12345678"))
	iot.PushPacketUpFromBottom(contact1, &subs)

	fmt.Println("contact1 subscribed contact", contact1.GetKey().Sig())
	fmt.Println("contact1 subscribed    subs", subs.Address.Sig())
	ce.WaitForActions()

	got, _ = contact1.(*testContact).popResultAsString() // the suback
	got = strings.Replace(got, atokenStruct.JWTID, "xxxx", 1)
	want = "[S,=ygRnE97Kfx0usxBqx5cygy4enA1eojeR,debg,12345678,jwtid,xxxx,pub2self,0]" //"no message received"
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	// start a monitor
	c := monitor_pod.ThingContext{}
	c.Topic = "get-unix-time"
	c.CommandMap = make(map[string]monitor_pod.Command)
	c.Index = 0
	c.Token = string(atoken)
	c.LogMeVerbose = true
	c.Host = ce.Aides[0].GetTCPAddress() //"localhost:8384"
	monitor_pod.ServeGetTime(string(atoken), &c)

	//time.Sleep(10 * time.Second) // every sleep is a failure

	ce.WaitForActions()

	sendmessage := packets.Send{}
	sendmessage.Address.FromString("get-unix-time")
	sendmessage.Source.FromString("contact1 address")
	sendmessage.Payload = []byte("get pubk")
	sendmessage.SetOption("debg", []byte("12345678"))

	iot.PushPacketUpFromBottom(contact1, &sendmessage)

	ce.WaitForActions()

	got, _ = contact1.(*testContact).popResultAsString() // the reply
	want = "[P,=ygRnE97Kfx0usxBqx5cygy4enA1eojeR,=xOZPbNiNsA_lM_6xJEwM1C7YmVMGlDpA,bht-Ka3j7GKuMFOablMlQnABnBvBeugvSf4CdFV3LXs,debg,12345678]"
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	client := http.Client{Timeout: 5 * time.Second}
	host := "127.0.0.1:8085"

	resp, err := client.Get("http://" + host + "/get/pubk?debg=12345678") // eg serves get-unix-time by hack
	if err != nil {
		fmt.Println("aide0 get-unix-time err", err)
		t.Errorf("got error %v", err)
		return
	}
	if resp.StatusCode != 200 {
		fmt.Println("get pubk not 200", resp.StatusCode)
	}
	defer resp.Body.Close()

	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	got = buf.String()
	fmt.Println("aide0 get pubk", got)

	want = "bht-Ka3j7GKuMFOablMlQnABnBvBeugvSf4CdFV3LXs"
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if false { // again, longer.
		client = http.Client{Timeout: 5 * time.Second}

		host = "get-unix-time.knotfree.com:8085"

		resp, err := client.Get("http://" + host + "/help") // eg serves get-unix-time by hack
		if err != nil {
			fmt.Println("aide0 get-unix-time err", err)
			t.Errorf("got error %v", err)
			return
		}
		if resp.StatusCode != 200 {
			fmt.Println("get pubk not 200", resp.StatusCode)
		}
		defer resp.Body.Close()

		buf := new(bytes.Buffer)
		buf.ReadFrom(resp.Body)
		got = buf.String()
		fmt.Println("aide0 help", got)

		tmp := len(got)
		if tmp < 500 {
			t.Errorf("got %v, want %v", got, want)
		}
	}

	_ = err
	_ = ok
	_ = want
	_ = got
	_ = atokenStruct

}

func TestSomeApis(t *testing.T) {

	tokens.LoadPublicKeys()

	atoken := tokens.GetTest32xToken()
	atokenStruct := tokens.ParseTokenNoVerify(atoken)

	got := ""
	want := ""
	ok := true
	var err error
	localtime := starttime
	getTime := func() uint32 {
		return localtime
	}
	isTCP := true
	aideCount := 2
	ce := iot.MakeSimplestCluster(getTime, isTCP, aideCount, "")
	globalClusterExec = ce

	ce.WaitForActions()

	theGuru := ce.Gurus[0]
	aide0 := ce.Aides[0]
	aide1 := ce.Aides[1]

	addr0 := aide0.GetHTTPAddress()
	fmt.Println("aide0 tcp", addr0) // 8384

	client := http.Client{Timeout: 1 * time.Second}
	resp, err := client.Get("http://" + addr0 + "/api2/getstats")
	if err != nil {
		fmt.Println("aide0 getstats err", err)
		t.Errorf("got error %v", err)
		return

	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		fmt.Println("getstats not 200")
	}
	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	got = buf.String()
	fmt.Println("aide0 stats", got)

	client = http.Client{Timeout: 1 * time.Second}
	resp, err = client.Get("http://localhost:8085/api1/getallstats")
	if err != nil {
		fmt.Println("aide0 getallstats err", err)
		t.Errorf("got error %v", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		fmt.Println("clusterstats not 200", resp.StatusCode)
	}
	buf = new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	got = buf.String()
	fmt.Println("aide0 clusterstats", got)

	client = http.Client{Timeout: 1 * time.Second}
	resp, err = client.Get("http://localhost:8085/api1/getGiantPassword")
	if err != nil {
		fmt.Println("aide0 getGiantPassword err", err)
		t.Errorf("got error %v", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		fmt.Println("getGiantPassword not 200", resp.StatusCode)
	}
	buf = new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	got = buf.String()
	fmt.Println("aide0 getGiantPassword", got)

	contact := getNewContactFromSlackestAide(ce, string(atoken))
	fmt.Println("contact", contact)

	_ = aide0
	_ = aide1
	_ = theGuru
	_ = err
	_ = ok
	_ = want
	_ = got
	_ = atokenStruct

}

func captureStdout(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func check(e error) {
	if e != nil {
		fmt.Println("ERROR because ", e)
	}
}
