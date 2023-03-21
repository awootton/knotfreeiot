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

package main

import (
	"crypto/sha256"
	"encoding/base64"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"runtime/trace"
	"syscall"
	"time"

	"github.com/awootton/knotfreeiot/iot"
	"github.com/awootton/knotfreeiot/mainhelpers"
	"github.com/awootton/knotfreeiot/tokens"
)

// Hint: add "127.0.0.1 knotfreeserver" to /etc/hosts
func main() {

	defer trace.Stop()

	// f, err := os.Create("cpu.out")
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// pprof.StartCPUProfile(f)

	// go func() {
	// 	log.Println(http.ListenAndServe("localhost:6060", nil))
	// }()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println("\r- Ctrl+C pressed in Terminal")
		runtime.GC()
		f, err := os.Create("heap.out")
		if err != nil {
			log.Fatal(err)
		}
		// pprof.StopCPUProfile()
		// pprof.WriteHeapProfile(f)
		f.Close()
		//trace.Stop()
		// pprof.StopCPUProfile()
		os.Exit(0)
	}()

	tokens.LoadPublicKeys()

	tokens.LoadPrivateKeys("~/atw/privateKeys4.txt")

	fmt.Println("Hello knotfreeserver")

	// no need to keep doing this mainhelpers.TrySomeS3Stuff()

	h := sha256.New()
	h.Write([]byte("AnonymousAnonymous"))
	hashBytes := h.Sum(nil)
	fmt.Println("Hello. sha256 of AnonymousAnonymous is " + base64.RawURLEncoding.EncodeToString(hashBytes))

	var htmp iot.HashType
	hptr := &htmp
	hptr.HashBytes([]byte("alice_vociferous_mcgrath"))
	var tmpbuf [24]byte
	hptr.GetBytes(tmpbuf[:])
	fmt.Println("Hello. fyi, standard hash of alice_vociferous_mcgrath is " + base64.RawURLEncoding.EncodeToString(tmpbuf[:]))

	isGuru := flag.Bool("isguru", false, "")

	// means that the limits are very small - for testing
	nano := flag.Bool("nano", false, "")

	token := flag.String("token", "", " an access token for our guru, if any")

	flag.Parse()

	if *token == "" {
		*token = tokens.GetImpromptuGiantToken()
	}

	tenKstats := tokens.GetTokenTenKStatsAndPrice()
	var mainLimits = &iot.ExecutiveLimits{}
	mainLimits.KnotFreeContactStats = tenKstats.Stats

	// mainLimits.Connections = 10k
	// mainLimits.Input = 10 * 1000
	// mainLimits.Output = 10 * 1000
	// mainLimits.Subscriptions = 1000 * 1000

	limits := mainLimits

	name := os.Getenv("POD_NAME")
	if len(name) == 0 {
		name = "DefaultPodName"
	}

	if *nano {
		limits = &iot.TestLimits
		fmt.Println("nano limits")
	}

	ce := iot.MakeTCPMain(name, limits, *token, *isGuru)
	mainhelpers.StartPublicServer(ce)
	for {
		time.Sleep(999999999 * time.Second)
	}
}
