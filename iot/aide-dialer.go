// Copyright 2019,2020,2021 Alan Tracey Wootton
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

package iot

import (
	"fmt"
	"math/rand"
	"net"
	"sync/atomic"
	"time"

	"github.com/awootton/knotfreeiot/packets"
	"github.com/awootton/knotfreeiot/tokens"
)

// DialContactToAnyAide is a utility to wait until we have a reference to
// an aide address and then get a tcp conn and keep it up and retry and keep it up forever.
// In test there is a ClusterExecutive struct that has references to all the names and addresses
// In k8s there is an operator that is periodically sending
func (ex *Executive) DialContactToAnyAide(isTCP bool, ce *ClusterExecutive) {

	// count := 0
	if isTCP {
		ex.dialAideAndServe()
		// for {
		// 	if ex.ClusterStats != nil {
		// 		if len(ex.ClusterStats.Stats) > 0 {
		// 			aides := make([]*ExecutiveStats, 0)
		// 			for _, stat := range ex.ClusterStats.Stats {
		// 				if !stat.IsGuru {
		// 					aides = append(aides, stat)
		// 				}
		// 			}
		// 			if len(aides) != 0 {
		// 				aide := aides[rand.Intn(len(aides))]
		// 				if len(aide.TCPAddress) > 4 {
		// 					if !strings.HasPrefix(aide.TCPAddress, ":") {
		// 						// we have a tcp address, dial it.
		// 						err := ex.dialAideAndServe(aide.TCPAddress, ce)
		// 						if err != nil {
		// 							fmt.Println("DialContactToAnyAide dialAideAndServe returned")
		// 						} //else {
		// 						// there's always an error or else we'd still be in dialAideAndServe
		// 						//}
		// 					}
		// 				}
		// 			}
		// 		}
		// 	}
		// 	time.Sleep(1000 * time.Millisecond)
		// 	count++
		// 	if (count % 100) == 0 {
		// 		fmt.Println("ex.Looker.contactToAnyAide is having problems")
		// 	}
		// } // for
	} else { // isTCP == false
		for { // not the ClusterStats technique. for unit test only.
			if len(ce.Aides) > 0 {
				aide := ce.Aides[rand.Intn(len(ce.Aides))]
				//  because we're in test
				// with no tcp
				token := tokens.GetImpromptuGiantToken()

				contact := &ContactStruct{}
				AddContactStruct(contact, contact, aide.Config)
				contact.SetExpires(contact.contactExpires + 60*60*24*365*10) // in 10 years

				// define a reader and a writer
				contact.realWriter = &DevNull{} // we don't subscribe or care what they say

				connect := packets.Connect{}
				connect.SetOption("token", []byte(token)) //SampleSmallToken))
				err := PushPacketUpFromBottom(contact, &connect)
				if err != nil {
					fmt.Println("connect problems test dial conn ", err)
					continue
				}

				for p := range ex.channelToAnyAide {
					// fmt.Println(" got channelToAnyAide aide ", p)
					err := PushPacketUpFromBottom(contact, p)
					if err != nil {
						fmt.Println("err PushPacketUpFromBottom ", err)
					}
				}
			} else {
				fmt.Println("no aides in cluster fail")
				panic("no aides in cluster fail")
			}
		}
	}
}

// return index, name, address of a random aide
func getTheIndex(ex *Executive) (int, string, string) {
	index := -1

	ex.statsmu.Lock()
	defer ex.statsmu.Unlock()

	// get a new one
	// just the aides

	ilen := len(ex.ClusterStats.Stats)
	if ilen == 0 {
		time.Sleep(100 * time.Millisecond)
		return -1, "", ""
	}
	randindex := rand.Intn(len(ex.ClusterStats.Stats))

	for i := 0; i < len(ex.ClusterStats.Stats); i++ {
		index = (i + randindex) % len(ex.ClusterStats.Stats)
		if !ex.ClusterStats.Stats[index].IsGuru {
			name := ex.ClusterStats.Stats[index].Name
			address := ex.ClusterStats.Stats[index].TCPAddress
			fmt.Println("getTheIndex returns", index, name, address)
			return index, name, address
		}
	}

	time.Sleep(100 * time.Millisecond)
	return -1, "", ""
}

// dialAideAndServe wants to maintain a connection to some aide so that we can
// pop packets off the channelToAnyAide and send them to the aide.
// The problem is that the aide might go away and we need get another. // ce *ClusterExecutive??
// how can we be sure to not call this twice?
var dialAideAndServeInvoked atomic.Int64

func (ex *Executive) dialAideAndServe() {

	// TODO: do this with channels and not these racey vars
	index := -1

	name := ""
	count := 0
	address := ""

	fmt.Println("top of dialAideAndServe ONE TOP ONCE", ex.Name)

	startReader := make(chan bool)

	dialAideAndServeInvoked.Add(1)

	var conn net.Conn = nil

	go func() {
		for { // forever
			for index == -1 {
				index, name, address = getTheIndex(ex)
			}
			var tmp int64
			dialAideAndServeInvoked.Store(tmp)
			fmt.Println("top of dialAideAndServe from ", ex.Name, " to ", name, address, tmp, index)

			// todo: tell prometheius we're dialing
			var err error
			conn, err = net.DialTimeout("tcp", address, time.Duration(uint64(2*time.Second)))
			if err != nil {
				if conn != nil {
					conn.Close() // really?
				}
				index = -1
				count++
				if (count % 1000) == 0 {
					fmt.Println("dialAideAndServe 2 error", address, err)
				}
				TCPNameResolverFail2.Inc()
				time.Sleep(100 * time.Millisecond) // try hard. There's a q filling up.
				continue                           // back to top
			}

			startReader <- true

			add := conn.(*net.TCPConn).LocalAddr().String()
			fmt.Println("dialAideAndServe tcp ", add)

			TCPNameResolverConnected.Inc()

			conn.(*net.TCPConn).SetNoDelay(true)
			conn.(*net.TCPConn).SetWriteBuffer(4096)

			connect := &packets.Connect{}
			connect.SetOption("token", []byte(tokens.GetImpromptuGiantToken()))
			connect.SetOption("comment", []byte("dialAideAndServe"+ex.Name))
			err = connect.Write(conn)
			if err != nil {
				fmt.Println("dialAideAndServe connect error", conn, err)
				conn.Close()
				index = -1
				time.Sleep(100 * time.Millisecond)
				continue // back to top
			}

			fmt.Println("dialAideAndServe connected, waiting to write")

			for { // pop packets off the channelToAnyAide and send them to the aide.
				if index == -1 {
					fmt.Println("dialAideAndServe index -1")
					if conn != nil {
						conn.Close()
					}
					break // from pop-packets, back to top of connect
				}
				oops := false

				ex.statsmu.Lock()
				exists := false
				for i := 0; i < len(ex.ClusterStats.Stats); i++ {
					if ex.ClusterStats.Stats[i].Name == name {
						exists = true
					}
				}
				if !exists {
					fmt.Println("dialAideAndServe name change", ex.ClusterStats.Stats[index].Name, name, index)
					oops = true
				}
				ex.statsmu.Unlock()

				if oops {
					index = -1
					conn.Close()
					break // from pop-packets, back to top of connect
				}
				if conn == nil {
					fmt.Println("ERROR dialAideAndServe conn nil")
					break // from pop-packets, back to top of connect
				}
				p := <-ex.channelToAnyAide
				err := p.Write(conn)
				if err != nil {
					fmt.Println("dialAideAndServe write error", conn, err)
					if conn != nil {
						conn.Close()
					}
					index = -1
					conn = nil
					break // from pop-packets, back to top of connect
				}
			}
		}
	}()

	go func() { // keep it alive timeout is 20 min
		for {
			time.Sleep(15 * time.Minute)
			p := &packets.Ping{}
			// do we care if this blocks?
			if len(ex.channelToAnyAide)*4 >= cap(ex.channelToAnyAide)*3 {
				fmt.Println("dialAideAndServe channel full error")
				time.Sleep(1 * time.Millisecond)
			}
			ex.channelToAnyAide <- p
		}
	}()

	// we don't care about the packets we're reading.
	// but we care if the connection goes away.
	// we're going to read packets and drop them on the floor.
	// if something goes wrong we'll close the connection and that's tge signal to start over.
	go func() {
		count := 0
		for {
			// wait for the connection to be established
			<-startReader
			for {
				// for !wasStarted {
				// 	time.Sleep(100 * time.Millisecond)
				// }
				if conn == nil {
					index = -1
					fmt.Println("ERROR dialAideAndServe packets.ReadPacket no conn ")
					break
				}
				p, err := packets.ReadPacket(conn)
				if err != nil {
					// if err.Error() == "EOF" { // this is what i'm seeing.
					// }
					//if wasStarted {
					fmt.Println("dialAideAndServe packets.ReadPacket error ", err, count, address)
					conn.Close()
					conn = nil
					index = -1
					//}
					time.Sleep(100 * time.Millisecond)
					break
				}
				_ = p // drop it on the floor
			}
		}
	}()

}
