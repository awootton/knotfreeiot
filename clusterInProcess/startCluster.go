package main

import (
	"fmt"
	"log"
	"net/http"

	_ "net/http/pprof"

	"time"

	"github.com/awootton/knotfreeiot/iot"
	"github.com/awootton/knotfreeiot/tokens"
)

func main() {

	// var err error

	// f, err := os.Create("cpu.out")
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// pprof.StartCPUProfile(f)

	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	tokens.LoadPublicKeys()
	tokens.LoadPrivateKeys("~/atw/privateKeys4.txt")

	fmt.Println("Hello, World!")

	getTime := func() uint32 {
		return uint32(time.Now().Unix())
	}

	isTCP := true
	// launch a guru
	aideCount := 1
	ce := iot.MakeSimplestCluster(getTime, isTCP, aideCount, "")

	theGuru := ce.Gurus[0]

	fmt.Println("theGuru tcp", theGuru.GetTCPAddress())   // 9001
	fmt.Println("theGuru http", theGuru.GetHTTPAddress()) // 9000

	// launch an aide using main.go

	tenKstats := tokens.GetTokenTenKStatsAndPrice()
	var mainLimits = &iot.ExecutiveLimits{}
	mainLimits.KnotFreeContactStats = tenKstats.Stats
	// limits := mainLimits

	// token := tokens.GetImpromptuGiantToken()
	// isGuru := false
	// ce2 := iot.MakeTCPMain("aide-0", limits, token, isGuru)
	// iot.StartPublicServer(ce2) // this will heartbeat the theAide

	theAide := ce.Aides[0]
	fmt.Println("theAide tcp", theAide.GetTCPAddress())   // 8384
	fmt.Println("theAide http", theAide.GetHTTPAddress()) // 8080

	time.Sleep(1 * time.Second)

	// guruList := []string{theGuru.Name}
	// guruAddress := []string{theGuru.GetTCPAddress()}

	// err = iot.PostUpstreamNames(guruList, guruAddress, theGuru.GetHTTPAddress())
	// checkerr(err)
	// err = iot.PostUpstreamNames(guruList, guruAddress, theAide.GetHTTPAddress())
	// checkerr(err)

	for {
		now := getTime()
		theGuru.Heartbeat(now)
		time.Sleep(10 * time.Second)
	}
	// fmt.Println("the bottom of the world!")
}

// func checkerr(err error) {
// 	if err != nil {
// 		fmt.Println("error", err)
// 	}
// }
