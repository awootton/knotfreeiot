package iot

import (
	"log"
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
// In k8s there is an operator that is periodically sending. Who dials from guru to aide?
func (ex *Executive) DialContactToAnyAide(isTCP bool, ce *ClusterExecutive, comment string) {

	// count := 0
	if isTCP {
		ex.dialAideAndServe("dialAideAndServe DialContactToAnyAide Executive initial call " + comment)
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
		// 						err := ex.dial Aide AndServe(aide.TCPAddress, ce)
		// 						if err != nil {
		// 							log.Println("DialContactToAnyAide dial AideAndServe returned")
		// 						} //else {
		// 						// there's always an error or else we'd still be in dial AideAndServe
		// 						//}
		// 					}
		// 				}
		// 			}
		// 		}
		// 	}
		// 	time.Sleep(1000 * time.Millisecond)
		// 	count++
		// 	if (count % 100) == 0 {
		// 		log.Println("ex.Looker.contactToAnyAide is having problems")
		// 	}
		// } // for
	} else { // isTCP == false
		log.Println("DON'T use non TCP mode anymore. Delete this code. ")
		for { // not the ClusterStats technique. for unit test only.
			// if len(ce.Aides) > 0 {
			// 	aide := ce.Aides[rand.Intn(len(ce.Aides))]
			// 	//  because we're in test
			// 	// with no tcp
			// 	token := tokens.GetImpromptuGiantToken()

			// 	contact := &ContactStruct{}
			// 	AddContactStruct(contact, contact, aide.Config)
			// 	contact.SetExpires(contact.contactExpires + 60*60*24*365*10) // in 10 years

			// 	// define a reader and a writer
			// 	contact.realWriter = &DevNull{} // we don't subscribe or care what they say

			// 	connect := packets.Connect{}
			// 	connect.SetOption("token", []byte(token))
			// 	err := PushPacketUpFromBottom(contact, &connect)
			// 	if err != nil {
			// 		log.Println("connect problems test dial conn ", err)
			// 		continue
			// 	}
			// 	// this is dead code
			// 	log.Println("starting for range channelToAnyAide", ex.Name)
			// 	for p := range ex.channelToAnyAide {
			// 		log.Println("got channelToAnyAide aide ", ex.Name)
			// 		err := PushPacketUpFromBottom(contact, p)
			// 		if err != nil {
			// 			log.Println("err PushPacketUpFromBottom ", err)
			// 		}
			// 	}
			// 	log.Println("ending for range channelToAnyAide", ex.Name)
			// } else
			{
				log.Println("no aides in cluster fail")
				panic("no aides in cluster fail")
			}
		}
	}
}

// return index, name, address of a random aide
// it depends on the ClusterStats being up to date. and can loop in the caller !
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
			log.Println("getTheIndex returns", index, name, address)
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

func (ex *Executive) dialAideAndServe(comment string) {

	// TODO: do this with channels and not these racey vars
	index := -1

	name := ""
	count := 0
	address := ""

	log.Println("top of dialAideAndServe ONE TOP ONCE", ex.Name, comment)

	startReader := make(chan bool)

	dialAideAndServeInvoked.Add(1)

	var conn net.Conn = nil

	go func() {
		for { // forever
			icount := 0
			for index == -1 {
				index, name, address = getTheIndex(ex)
				icount++
				if icount > 10 {
					log.Println("dialAideAndServe 1 error waiting for clusterStats too long", ex.Name, index, name, address)
					time.Sleep(1000 * time.Millisecond)
				}
			}
			var tmp int64
			dialAideAndServeInvoked.Store(tmp)
			log.Println("top of dialAideAndServe from ", ex.Name, " to ", name, address, tmp, index)

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
					log.Println("dialAideAndServe 2 error", address, err)
				}
				TCPNameResolverFail2.Inc()
				log.Println("dialAideAndServe top of dial timeout (2sec) ", ex.Name, " to ", name, address, tmp, index)

				time.Sleep(100 * time.Millisecond) // try hard. There's a q filling up.
				continue                           // back to top
			}

			startReader <- true

			add := conn.(*net.TCPConn).LocalAddr().String()
			add2 := conn.(*net.TCPConn).RemoteAddr().String()
			helpful_comment := "dialAideAndServe " + ex.Name + " to " + name + " " + address + " from " + add + " to " + add2
			log.Println("dialAideAndServe connected to tcp ", add, " to ", add2, " from ", ex.Name, " to ", name, address, index)

			TCPNameResolverConnected.Inc()

			conn.(*net.TCPConn).SetNoDelay(true)
			// this seems small? It's not like we have a large number or any-aide connection channels.

			var write_buffer_size int = 4096
			write_buffer_size = 1024 * 64 * 2 // 128k
			conn.(*net.TCPConn).SetWriteBuffer(write_buffer_size)

			connect := &packets.Connect{}
			connect.SetOption("token", []byte(tokens.GetImpromptuGiantToken()))
			connect.SetOption("helpful_comment", []byte(helpful_comment))
			err = connect.Write(conn)
			if err != nil {
				log.Println("dialAideAndServe connect error", conn, err)
				conn.Close()
				index = -1
				time.Sleep(100 * time.Millisecond)
				continue // back to top
			}

			log.Println("dialAideAndServe connected, waiting to write", ex.Name)

			// we pop them off the huge channelToAnyAide and write them to this socket.
			// they reappear in other machine and turn back into packets to get processed
			// by the contact over there. Does that contact have a big enough TCP buffer?

			for { // pop packets off the channelToAnyAide and send them to the aide.
				if index == -1 {
					log.Println("dialAideAndServe index -1")
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
					log.Println("dialAideAndServe name change", ex.ClusterStats.Stats[index].Name, name, index)
					oops = true
				}
				ex.statsmu.Unlock()

				if oops {
					index = -1
					log.Println("dialAideAndServe have index -1", name, index)

					conn.Close()
					break // from pop-packets, back to top of connect
				}
				if conn == nil {
					log.Println("ERROR dialAideAndServe conn nil")
					break // from pop-packets, back to top of connect
				}
				// log.Println("dialAideAndServe waiting channelToAnyAide", ex.Name)
				p := <-ex.channelToAnyAide
				if serviceDebugSession1 {
					log.Println("dialAideAndServe got from channelToAnyAide and writing", p.Sig(), ex.Name)
				}
				err := p.Write(conn)
				if err != nil {
					log.Println("dialAideAndServe write error", conn, err)
					if conn != nil {
						conn.Close()
					}
					index = -1
					conn = nil
					break // from pop-packets, back to top of connect
				}
			}
			log.Println("dialAideAndServe pop then write loop exiting", ex.Name)
		}
	}()

	go func() { // keep it alive timeout is 20 min
		for {
			time.Sleep(15 * time.Minute)
			p := &packets.Ping{}
			// do we care if this blocks?
			if len(ex.channelToAnyAide)*4 >= cap(ex.channelToAnyAide)*3 {
				log.Println("dialAideAndServe channel full error")
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
					log.Println("ERROR dialAideAndServe packets.ReadPacket no conn ")
					break
				}
				p, err := packets.ReadPacket(conn)
				if err != nil {
					// if err.Error() == "EOF" { // this is what i'm seeing.
					// }
					//if wasStarted {
					log.Println("dialAideAndServe packets.ReadPacket error ", err, count, address)
					if conn != nil {
						conn.Close()
					}
					conn = nil
					index = -1
					//}
					time.Sleep(100 * time.Millisecond)
					break
				}
				_ = p // drop it on the floor
			}
			log.Println("dialAideAndServe reader loop exiting")
		}
	}()

}

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
