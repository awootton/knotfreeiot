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
	"syscall"
	"time"

	"net/http"
	_ "net/http/pprof"

	"github.com/awootton/knotfreeiot/iot"
	"github.com/awootton/knotfreeiot/tokens"
	"golang.org/x/sys/unix"
)

// when running?
// kk exec podname -- wget -o - http://localhost:6080/debug/pprof/heap > heap.profile

// Hint: add "127.0.0.1 knotfreeserver" to /etc/hosts
func main() {

	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	if isRunningUnderRosetta() {
		log.Println("Running under Rosetta  !!!! FAILURE !!!! MAJOR PROBLEM !!!! FIXME:.")
	} else {
		log.Println("Not running under Rosetta")
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		log.Println("\r- Ctrl+C pressed in Terminal. why? why? why? ")
		runtime.GC()

		// show memory stats
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		log.Printf("Alloc = %v MiB", bToMb(m.Alloc))
		log.Printf("\tTotalAlloc = %v MiB", bToMb(m.TotalAlloc))
		log.Printf("\tSys = %v MiB", bToMb(m.Sys))
		log.Printf("\tNumGC = %v\n", m.NumGC)
		log.Printf("\tHeapSys = %v MiB", bToMb(m.HeapSys))

		// where to put this?
		// useless. It's binary. pprof.WriteHeapProfile(os.Stdout)

		// we're crashing because of  Warning  Unhealthy  51s   kubelet
		//  Readiness probe failed: Get "http://10.244.37.232:8085/healthz": dial tcp 10.244.37.232:8085: conn
		// but why?

		// I'm not sure if this is helping and it's hard to read
		// // 3. Allocate a buffer large enough for all goroutines
		// buf := make([]byte, 1024*1024)
		// n := runtime.Stack(buf, true)

		// // 4. Dump the state to standard error (or your logging system)
		// os.Stderr.Write(buf[:n])

		os.Exit(0)
	}()

	tokens.LoadPublicKeys()

	tokens.LoadPrivateKeys("~/atw/privateKeys4.txt")

	log.Println("Hello knotfreeserver")

	// no need to keep doing this mainhelpers.TrySomeS3Stuff()

	h := sha256.New()
	h.Write([]byte("AnonymousAnonymous"))
	hashBytes := h.Sum(nil)
	log.Println("Hello. sha256 of AnonymousAnonymous is " + base64.RawURLEncoding.EncodeToString(hashBytes))

	var htmp iot.HashType
	hptr := &htmp
	hptr.HashBytes([]byte("alice_vociferous_mcgrath"))
	var tmpbuf [24]byte
	hptr.GetBytes(tmpbuf[:])
	log.Println("Hello. fyi, standard hash of alice_vociferous_mcgrath is " + base64.RawURLEncoding.EncodeToString(tmpbuf[:]))

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
		log.Println("nano limits")
	}

	ce := iot.MakeTCPMain(name, limits, *token, *isGuru)
	iot.StartPublicServer(ce)
	for {
		time.Sleep(999999999 * time.Second)
	}
}

func bToMb(u uint64) any {
	return fmt.Sprintf("%.2f", float64(u)/1024/1024)
}

func isRunningUnderRosetta() bool {
	// sysctl.proc_translated returns 1 if running under Rosetta, 0 otherwise
	val, err := unix.SysctlUint32("sysctl.proc_translated")
	if err != nil {
		return false
	}
	return val == 1
}
