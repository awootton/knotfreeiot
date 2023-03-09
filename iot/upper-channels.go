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
	"sync"

	"github.com/awootton/knotfreeiot/packets"
	"github.com/dgryski/go-maglev"
)

// this file is a bit of a grab bag.

// DevNull has it's uses.
type DevNull struct {
}

func (null *DevNull) Read(b []byte) (int, error) {
	return len(b), nil
}

func (null *DevNull) Write(b []byte) (int, error) {
	return len(b), nil
}

type ByteChan struct {
	TheChan chan []byte
}

// this is a packet in bytes
func (bc *ByteChan) Write(b []byte) (int, error) {
	bc.TheChan <- b
	// fmt.Println(" ByteChan has ", string(b))
	return len(b), nil
}

// upperChannel represents the 'upper' version of a contact.
// Unlike their numerous lower bretheren they have buffers.

type upperChannel struct {
	name    string
	address string // as in tcp ip:port
	up      chan packets.Interface
	down    chan packets.Interface
	running bool
	ex      *Executive
}

// upstreamRouterStruct is maybe virtual in the future
// it's really just a sub part of LookupTableStruct
type upstreamRouterStruct struct {
	//
	channels       []*upperChannel
	maglev         *maglev.Table
	previousmaglev *maglev.Table
	name2channel   map[string]*upperChannel
	mux            sync.Mutex
}

// getUpperChannel returns which upperChannel to handle i
func (router *upstreamRouterStruct) getUpperChannel(h uint64) *upperChannel {
	router.mux.Lock()
	defer router.mux.Unlock()
	index := router.maglev.Lookup(h)
	if index >= len(router.channels) {
		fmt.Println("oops oops oops oops oops oops ")
	}
	c := router.channels[index]
	return c
}

// SetUpstreamNames is called by a cluster exec of some kind when changing the guru count.
// We will update upstreamRouterStruct
// names are like:  guru-0f3bca46d414d506ecce3de9762df6c3
// addresses are like: 10.244.0.149:8384
func (me *LookupTableStruct) SetUpstreamNames(names []string, addresses []string) {

	// don't block on anything.
	router := me.upstreamRouter
	router.mux.Lock()
	defer router.mux.Unlock()

	if len(names) != len(addresses) {
		fmt.Println("error len(names) != len(addresses) panic")
		return
	}

	if len(names) == len(router.channels) {
		for i, c := range router.channels {
			if names[i] != c.name {
				return // no changes
			}
		}
	}
	// maybe some more verifications?

	if me.isGuru {
		me.setGuruUpstreamNames(names)
		return
	}

	// we're an aide.
	oldContacts := router.channels

	router.channels = make([]*upperChannel, len(names))
	theNamesThisTime := make(map[string]string, len(names))

	// iterate the names passed in.
	// constuct a new list. populate it with existing upc
	// when possible

	for i, name := range names {
		address := addresses[i]
		theNamesThisTime[name] = address

		upc, found := router.name2channel[name]
		if found && upc.running {
			router.channels[i] = upc
		} else {
			fmt.Println("starting upper router from ", me.ex.Name, " to ", name)
			upc = &upperChannel{}
			upc.name = name
			upc.address = address
			upc.up = make(chan packets.Interface, 1280)
			upc.down = make(chan packets.Interface, 128)
			upc.ex = me.ex
			router.channels[i] = upc
			go upc.dialGuru()
		}
	}
	// lose the stale ones
	if len(oldContacts) != 0 {
		fmt.Println("forgetting old channels ", oldContacts, " vs ", theNamesThisTime)
	}

	for _, upc := range oldContacts {
		_, found := theNamesThisTime[upc.name]
		if !found {
			upc.running = false
			fmt.Println("forgetting upper router ", upc.name)
			close(upc.up)
			close(upc.down)
			delete(router.name2channel, upc.name)
		}
	}

	router.previousmaglev = router.maglev
	maglevsize := maglev.SmallM
	if DEBUG {
		maglevsize = 97
	}
	router.maglev = maglev.New(names, uint64(maglevsize))
	// order subscriptions to be forwarded to the new UpContact.

	// iterate all the subscriptions and push up (again) the ones that have been remapped.
	// iterate all subscriptions and delete the ones that don't map here anymore.
	// note that this will need to push up to the guru through the conn
	// just defined and it can BLOCK until the conn completes.
	// we must watch those buffers and not block here.
	go func() {
		command := callBackCommand{}
		command.callback = reSubscribeRemappedTopics

		for _, bucket := range me.allTheSubscriptions {
			command.wg.Add(1)
			bucket.incoming <- &command
		}
		command.wg.Wait()
	}()

}

// setGuruUpstreamNames because the guru needs to know also.
// recalc the maglev. reveal all the subs and delete the ones we wouldn't have.
// todo: guruDeleteRemappedAndGoneTopics on a case by case basis?
func (me *LookupTableStruct) setGuruUpstreamNames(names []string) {

	// only called from above and the mux is locked.
	router := me.upstreamRouter

	router.previousmaglev = router.maglev
	maglevsize := maglev.SmallM
	if DEBUG {
		maglevsize = 97
	}
	router.maglev = maglev.New(names, uint64(maglevsize))

	myindex := -1
	for i, n := range names {
		if n == me.myname {
			myindex = i
		}
	}

	// iterate all subscriptions and delete the ones that don't map here anymore.

	go func() {
		command := callBackCommand{}
		command.callback = guruDeleteRemappedAndGoneTopics
		command.index = myindex

		for _, bucket := range me.allTheSubscriptions {
			command.wg.Add(1)
			bucket.incoming <- &command
		}
		command.wg.Wait()
	}()
}
