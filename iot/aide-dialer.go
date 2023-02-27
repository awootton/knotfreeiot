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
	"strings"
	"sync"
	"time"

	"github.com/awootton/knotfreeiot/packets"
	"github.com/awootton/knotfreeiot/tokens"
)

// DialContactToAnyAide is a utility to wait until we have a reference to
// an aide address and then get a tcp conn and keep it up and retry and keep it up forever.
// In test there is a ClusterExecutive struct that has references to all the names and addresses
// In k8s there is an operator that is periodically sending
func (ex *Executive) DialContactToAnyAide(isTCP bool, ce *ClusterExecutive) {

	defer fmt.Println("** escape from infinite loop **")
	count := 0
	if isTCP {
		for {
			if ex.ClusterStats != nil {
				if len(ex.ClusterStats.Stats) > 0 {
					aides := make([]*ExecutiveStats, 0)
					for _, stat := range ex.ClusterStats.Stats {
						if !stat.IsGuru {
							aides = append(aides, stat)
						}
					}
					if len(aides) != 0 {
						aide := aides[rand.Intn(len(aides))]
						if len(aide.TCPAddress) > 4 {
							if !strings.HasPrefix(aide.TCPAddress, ":") {
								// we have a tcp address, dial it.
								err := ex.dialAideAndServe(aide.TCPAddress, ce)
								if err != nil {
									fmt.Println("DialContactToAnyAide dialAideAndServe returned")
								} //else {
								// there's always an error or else we'd still be in dialAideAndServe
								//}
							}
						}
					}
				}
			}
			time.Sleep(1000 * time.Millisecond)
			count++
			if (count % 100) == 0 {
				fmt.Println("ex.Looker.contactToAnyAide is having problems")
			}
		} // for
	} else { // isTCP == false
		for { // not the ClusterStats technique. for unit test only.
			if len(ce.Aides) > 0 {
				aide := ce.Aides[rand.Intn(len(ce.Aides))]
				//   because we're in test
				// with no tcp
				token := tokens.Test32xToken //GetImpromptuGiantToken()

				contact := &ContactStruct{}
				AddContactStruct(contact, contact, aide.Config)
				contact.SetExpires(contact.contactExpires + 60*60*24*365*10) // in 10 years

				// define a reader and a writer
				contact.realWriter = &DevNull{} // we don't  subscribe or care what they say

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

func (ex *Executive) dialAideAndServe(address string, ce *ClusterExecutive) error {

	fmt.Println("top of dialAideAndServe ", address, " of ", ex.Name)

	// todo: tell prometheius we're dialing
	conn, err := net.DialTimeout("tcp", address, time.Duration(uint64(2*time.Second)))
	if err != nil {
		fmt.Println("dialAideAndServe 2 fail", address, err)
		TCPNameResolverFail2.Inc()
		return nil
	}
	// have conn.(*net.TCPConn)

	TCPNameResolverConnected.Inc()

	conn.(*net.TCPConn).SetNoDelay(true)
	conn.(*net.TCPConn).SetWriteBuffer(4096)

	var founderr error

	go func() {
		for founderr == nil {
			p, err := packets.ReadPacket(conn)
			if err != nil {
				founderr = err
				return
			}
			_ = p // drop it on the floor
		}
	}()

	var mux sync.Mutex // needed

	connect := &packets.Connect{}
	connect.SetOption("token", []byte(tokens.GetImpromptuGiantToken()))
	mux.Lock()
	err = connect.Write(conn)
	mux.Unlock()
	if err != nil {
		fmt.Println("a write c fail", conn, err)
		conn.Close()
		founderr = err
		return err
	}

	go func() {
		for founderr == nil {
			time.Sleep(300 * time.Second)
			p := &packets.Ping{}
			ex.channelToAnyAide <- p
			// mux.Lock()
			// err = p.Write(conn)
			// mux.Unlock()
			// if err != nil {
			// 	fmt.Println("a write ping fail", conn, err)
			// 	conn.Close()
			// 	founderr = err
			//}
		}
	}()

	for p := range ex.channelToAnyAide {
		err := p.Write(conn)
		// fmt.Println("channelToAnyAide pushing to aide ", p)
		if err != nil || founderr != nil {
			if err == nil {
				err = founderr
			}
			if founderr == nil {
				founderr = err
			}
			fmt.Println("err L pushing to aide ", err)
			conn.Close()
		}
	}
	return nil
}
