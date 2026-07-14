package main

import (
	"log"
	"net/http"

	_ "net/http/pprof"

	"time"

	"github.com/awootton/knotfreeiot/iot"
	"github.com/awootton/knotfreeiot/tokens"
)

func main() {

	// iot.InitMongEnv()
	// iot.InitIotTables()

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

	log.Println("StartCluster Hello, World!")

	getTime := func() uint32 {
		return uint32(time.Now().Unix())
	}

	_ = tokens.MakeRandomPhrase(1) // force init of the gadget.

	isTCP := true
	// launch a guru
	aideCount := 1
	ce := iot.MakeSimplestCluster(getTime, isTCP, aideCount, "")

	theGuru := ce.Gurus[0]

	log.Println("StartClustertheGuru tcp", theGuru.GetTCPAddress())    // 19001
	log.Println("StartCluster theGuru http", theGuru.GetHTTPAddress()) // 19000

	// launch an aide using main.go

	// tenKstats := tokens.GetTokenTenKStatsAndPrice()
	// var mainLimits = &iot.ExecutiveLimits{}
	// mainLimits.KnotFreeContactStats = tenKstats.Stats
	// limits := mainLimits

	// token := tokens.GetImpromptuGiantToken()
	// isGuru := false
	// ce2 := iot.MakeTCPMain("aide-0", limits, token, isGuru)
	// iot.StartPublicServer(ce2) // this will heartbeat the theAide

	theAide := ce.Aides[0]
	log.Println("StartClustertheAide tcp", theAide.GetTCPAddress())    // 8384
	log.Println("StartCluster theAide http", theAide.GetHTTPAddress()) // 8080

	// init the stupid DB so we don't get a fail on the first request. this is a hack.
	_, ok := iot.GetSubscription("xOZPbNiNsA_lM_6xJEwM1C7YmVMGlDpA")
	if !ok {
		log.Println("subscription not found, xOZPbNiNsA_lM_6xJEwM1C7YmVMGlDpA during startCluster init.")
	}

	ce.WaitForActions() // force cluster status to go out, normally this is done by the k8s operator.

	// time.Sleep(1 * time.Second)

	// guruList := []string{theGuru.Name}
	// guruAddress := []string{theGuru.GetTCPAddress()}

	// err = iot.PostUpstreamNames(guruList, guruAddress, theGuru.GetHTTPAddress())
	// checkerr(err)
	// err = iot.PostUpstreamNames(guruList, guruAddress, theAide.GetHTTPAddress())
	// checkerr(err)

	token, _ := tokens.GetImpromptuGiantTokenLocal("", "")
	// what a confusing mess. monitor_pod.PublishTestTopic(token)
	_ = token

	startSomeServers := func() {
		namesList := []string{"get-unix-time", "get-unix-time_iot", "a-thermometer-demo_iot"}
		for _, name := range namesList {
			iot.StartAServer(name, "")
		}
	}
	_ = startSomeServers
	startSomeServers()

	for {
		now := getTime()
		theGuru.Heartbeat(now)
		time.Sleep(10 * time.Second)
	}
	// log.Println("the bottom of the world!")
}
