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
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"runtime"
	"strconv"
	"time"

	"golang.org/x/crypto/nacl/box"

	"github.com/awootton/knotfreeiot/packets"
	"github.com/awootton/knotfreeiot/tokens"
	"github.com/prometheus/client_golang/prometheus"
)

// Utility and controller struct and functions for LookupTable

// Executive is one instance of iot service.
// TODO: FYI: Executive and LookupTableStruct and ContactStructConfig and upstreamRouterStruct are really just one thing.

type Executive struct {
	Looker *LookupTableStruct
	Config *ContactStructConfig
	Name   string
	isGuru bool

	httpAddress string // eg 33.44.55.66:7374 or localhost:8089 or something
	tcpAddress  string
	textAddress string
	mqttAddress string

	getTime func() uint32

	Limits *ExecutiveLimits

	ClusterStats *ClusterStats // All the stats

	ClusterStatsString string // serialization of ClusterStats

	Billing BillingAccumulator

	IAmBadError error // if something happened to simply ruin us and we're quitting.

	channelToAnyAide chan packets.Interface

	ce *ClusterExecutive
}

// ClusterExecutive is a list of Executive
// used for testing.
type ClusterExecutive struct {
	Aides              []*Executive
	Gurus              []*Executive
	limits             *ExecutiveLimits
	currentGuruList    []string
	currentAddressList []string
	isTCP              bool
	currentPort        int

	PublicKeyTemp  *[32]byte //curve25519.PublicKey // temporary to this run not ed25519
	PrivateKeyTemp *[32]byte //curve25519.PrivateKey
}

// ExecutiveLimits will be how we tell if the ex is 'full'
type ExecutiveLimits struct {
	tokens.KnotFreeContactStats `json:"contactStats"` //   in out su co
}

// ExecutiveStats is fractions relative to the limits.
//
// a fraction: 1.0 is 100% maxed out. 0 is idle.
type ExecutiveStats struct {
	// four float32 :   in out su co
	tokens.KnotFreeContactStats `json:"contactStats"`
	Buffers                     float64 `json:"buf"`
	Name                        string  `json:"name"`
	HTTPAddress                 string  `json:"http"`
	TCPAddress                  string  `json:"tcp"`
	IsGuru                      bool    `json:"guru"`
	Memory                      int64   `json:"mem"`

	Limits *ExecutiveLimits `json:"limits"`
}

// ClusterStats is ExecutiveStats from everyone in the cluster.
// maybe slightly delayed
type ClusterStats struct {
	When  uint32 // unix time
	Stats []*ExecutiveStats
}

// TestLimits is for tests of autoscaling.TestLimits
// These are executive limits and not to be confused with token limits.
// irl connections limit is likely to be 10k and subscriptions 1e6
var TestLimits = ExecutiveLimits{}

func init() {
	TestLimits.Connections = 16
	TestLimits.Input = 10
	TestLimits.Output = 10
	TestLimits.Subscriptions = 64
}

// MakeSimplestCluster is just for testing as k8s doesn't work like this.
// can't work in CI
func MakeSimplestCluster(timegetter func() uint32, isTCP bool, aideCount int, suffix string) *ClusterExecutive {

	GuruNameToConfigMap = make(map[string]*Executive)

	ce := &ClusterExecutive{}
	ce.isTCP = isTCP
	if isTCP {
		ce.currentPort = 9000
	}

	secret := tokens.GetPrivateKey("_9sh") // it's actually binary
	r := bytes.NewReader([]byte(secret))
	pub, priv, c := box.GenerateKey(r) // rand.Reader) //was ed25519.GenerateKey(rand.Reader)
	ce.PublicKeyTemp = pub
	ce.PrivateKeyTemp = priv
	_ = c

	defer ce.WaitForActions()

	ce.limits = &TestLimits

	// set up
	guru0 := NewExecutive(100, "guru0"+suffix, timegetter, true, ce)
	GuruNameToConfigMap["guru0"+suffix] = guru0
	guru0.Config.ce = ce
	ce.Gurus = append(ce.Gurus, guru0)

	if isTCP {
		guru0.httpAddress = ce.GetNextAddress()
		guru0.tcpAddress = ce.GetNextAddress()
		guru0.textAddress = ce.GetNextAddress()
		guru0.mqttAddress = ce.GetNextAddress()

		MakeTCPExecutive(guru0, guru0.tcpAddress)
		MakeHTTPExecutive(guru0, guru0.httpAddress)
	}
	ce.currentGuruList = []string{"guru0" + suffix}
	ce.currentAddressList = []string{guru0.tcpAddress}

	for i := int64(0); i < int64(aideCount); i++ {
		aide1 := NewExecutive(100, "aide"+strconv.FormatInt(i, 10)+suffix, timegetter, false, ce)
		aide1.Config.ce = ce
		ce.Aides = append(ce.Aides, aide1)
		GuruNameToConfigMap[aide1.Name] = aide1

		if isTCP {
			aide1.httpAddress = ce.GetNextAddress()
			aide1.tcpAddress = ce.GetNextAddress()
			aide1.textAddress = ce.GetNextAddress()
			aide1.mqttAddress = ce.GetNextAddress()
			MakeTCPExecutive(aide1, aide1.tcpAddress)
			MakeTextExecutive(aide1, aide1.textAddress)
			MakeHTTPExecutive(aide1, aide1.httpAddress)
			// FIXME : MakeMQTTExecutive
		}
		go aide1.DialContactToAnyAide(isTCP, ce)
	}

	go guru0.DialContactToAnyAide(isTCP, ce)

	if isTCP {
		// don't cheat: send these by http
		if len(ce.Gurus) > 0 {
			err := PostUpstreamNames(ce.currentGuruList, ce.currentAddressList, ce.Gurus[0].httpAddress)
			if err != nil {
				fmt.Println("post fail1")
			}
		}
		for _, aide := range ce.Aides {
			err := PostUpstreamNames(ce.currentGuruList, ce.currentAddressList, aide.httpAddress)
			if err != nil {
				fmt.Println("post fail2")
			}
		}
	} else {
		if len(ce.Gurus) > 0 {
			ce.Gurus[0].Looker.SetUpstreamNames(ce.currentGuruList, ce.currentGuruList)
		}
		for _, aide := range ce.Aides {
			aide.Looker.SetUpstreamNames(ce.currentGuruList, ce.currentGuruList)
		}
	}

	return ce
}

// MakeTCPMain is called by main(s) and it news a table and contacts list and starts tcp acceptors.
func MakeTCPMain(name string, limits *ExecutiveLimits, token string, isGuru bool) *ClusterExecutive {

	isTCP := true
	timegetter := func() uint32 {
		return uint32(time.Now().Unix())
	}

	ce := &ClusterExecutive{}
	ce.isTCP = isTCP

	// we should derive this from the current priv jwt ed25519 secret
	secret := tokens.GetPrivateKey("_9sh") // it's actually binary
	r := bytes.NewReader([]byte(secret))
	a, b, c := box.GenerateKey(r) // rand.Reader) //was ed25519.GenerateKey(rand.Reader)
	ce.PublicKeyTemp = a
	ce.PrivateKeyTemp = b
	_ = c

	ce.currentPort = 80

	ce.limits = limits

	aide1 := NewExecutive(1024*1024, name, timegetter, isGuru, ce)
	aide1.Limits = limits
	aide1.Config.ce = ce
	ce.Aides = append(ce.Aides, aide1)

	myip := "" //os.Getenv("MY_POD_IP")

	aide1.httpAddress = myip + ":8080"
	aide1.tcpAddress = myip + ":8384"
	aide1.textAddress = myip + ":7465"
	aide1.mqttAddress = myip + ":1883"

	MakeTCPExecutive(aide1, aide1.tcpAddress)
	MakeTextExecutive(aide1, aide1.textAddress)
	MakeHTTPExecutive(aide1, aide1.httpAddress)
	MakeMqttExecutive(aide1, aide1.mqttAddress)

	go aide1.DialContactToAnyAide(isTCP, ce)

	return ce
}

// NewExecutive A wrapper to hold and operate
func NewExecutive(sizeEstimate int, aname string, timegetter func() uint32, isGuru bool, ce *ClusterExecutive) *Executive {

	look := NewLookupTable(sizeEstimate, aname, isGuru, timegetter)
	config := NewContactStructConfig(look)
	config.Name = aname

	ex := &Executive{}
	ex.Looker = look
	ex.Config = config
	ex.Name = aname
	ex.getTime = timegetter
	ex.Limits = &TestLimits
	ex.isGuru = isGuru
	ex.ClusterStatsString = "none-yet"
	ex.ce = ce

	if sizeEstimate > 1000 {
		ex.channelToAnyAide = make(chan packets.Interface, 10)
	} else {
		ex.channelToAnyAide = make(chan packets.Interface, 1024)
	}
	look.ex = ex

	// start with some stats - just me.
	ex.ClusterStats = &ClusterStats{}
	ex.ClusterStats.When = ex.getTime()
	ex.ClusterStats.Stats = append(ex.ClusterStats.Stats, ex.GetExecutiveStats())
	ex.ClusterStats.Stats[0].HTTPAddress = "localhost:8080"
	ex.ClusterStats.Stats[0].TCPAddress = "localhost:8384"

	return ex
}

// GetNextAddress hands out localhost addresses starting at 9000
func (ce *ClusterExecutive) GetNextAddress() string {

	if ce.currentPort == 0 {
		ce.currentPort = 9000
	}
	address := "localhost:" + strconv.FormatInt(int64(ce.currentPort), 10)
	ce.currentPort++
	return address
}

// ExpansionDesired is
type ExpansionDesired struct {
	ChangeAides int    // +1 for grow, 0 for same, -1 for shrink
	RemoveAide  string // the name of the aide to delete

	ChangeGurus int // +1 for grow, 0 for same, -1 for shrink

}

// CalcExpansionDesired is used locally in tests and used by the operator to manage the cluster.
func CalcExpansionDesired(aides []*ExecutiveStats, gurus []*ExecutiveStats) ExpansionDesired {

	result := ExpansionDesired{}

	subsTotal := 0.0
	buffersFraction := 0.0
	contactsTotal := 0.0

	for _, ex := range aides {
		c := ex.Subscriptions
		// subsTotal += float64(c) don't scale aides on subscriptions just yet
		_ = c
		buffersFraction += float64(ex.Buffers)
		con := ex.Connections
		contactsTotal += float64(con)
		// todo: check bps up and down
	}
	subsTotal /= float64(len(aides))
	buffersFraction /= float64(len(aides))
	contactsTotal /= float64(len(aides))

	max := subsTotal // pick the one closest to being 100%
	if max < buffersFraction {
		max = buffersFraction
	}
	if max < contactsTotal {
		max = contactsTotal
	}

	// if the average is > 90% then grow
	if max >= 0.9 {
		result.ChangeAides = +1
	} else if len(aides) > 1 {
		// we can only shrink if the result won't just grow again.
		// with some (10%) margin.
		tmp := max * float64(len(aides))
		tmp /= float64(len(aides) - 1)
		if tmp < 0.80 {
			result.ChangeAides = -1
			// we can shrink, which one?
			index := 0
			max := float64(0.0)
			for i, ex := range aides {
				c := ex.Subscriptions
				fract := ex.Buffers
				if c > max {
					max = c
					index = i
				}
				buffersFraction += float64(fract)
				con := ex.Connections
				contactsTotal += float64(con)
				// check bps
			}
			i := index
			result.RemoveAide = aides[i].Name
		}
	}

	// now, same routine for gurus

	subsTotal = 0.0
	buffersFraction = 0.0
	contactsTotal = 0.0
	for i, ex := range gurus {
		c := ex.Subscriptions
		subsTotal += float64(c)
		_ = c
		buffersFraction += float64(ex.Buffers)
		con := ex.Connections
		contactsTotal += float64(con)
		// check bps up and down
		_ = i
	}
	subsTotal /= float64(len(gurus))
	buffersFraction /= float64(len(gurus))
	contactsTotal /= float64(len(gurus))

	max = subsTotal
	if max < buffersFraction {
		max = buffersFraction
	}

	if max >= 0.9 {

		result.ChangeGurus = +1

	} else if len(gurus) > 1 {
		// we can only shrink if the result won't just grow again.
		// with some (10%) margin.
		tmp := max * float64(len(gurus))
		tmp /= float64(len(gurus) - 1)
		if tmp < 0.80 {

			result.ChangeGurus = -1
		}
	}
	return result
}

// Operate where we pretend to be an Operator and resize the cluster.
// This is really only for test. Only works in non-tcp mode
// Does not call heartbeat or advance the time.
func (ce *ClusterExecutive) Operate() {

	aides := make([]*ExecutiveStats, len(ce.Aides))
	gurus := make([]*ExecutiveStats, len(ce.Gurus))

	for i, n := range ce.Aides {
		aides[i] = n.GetExecutiveStats()
	}
	for i, n := range ce.Gurus {
		gurus[i] = n.GetExecutiveStats()
	}

	expansion := CalcExpansionDesired(aides, gurus)

	// if the average is > 90% then grow
	if expansion.ChangeAides > 0 {

		_ = CalcExpansionDesired(aides, gurus)

		anaide := ce.Aides[0]
		n := strconv.FormatInt(int64(len(ce.Aides)), 10)
		aide1 := NewExecutive(100, "aide"+n, anaide.getTime, false, ce)
		ce.Aides = append(ce.Aides, aide1)
		aide1.Looker.SetUpstreamNames(ce.currentGuruList, ce.currentGuruList)
		for _, ex := range ce.Gurus {
			ex.Looker.FlushMarkerAndWait()
		}
		for _, ex := range ce.Aides {
			ex.Looker.FlushMarkerAndWait()
		}
	} else if expansion.ChangeAides < 0 {
		if true {
			// we can shrink, which one?
			index := 0
			for i, ex := range aides {
				if ex.Name == expansion.RemoveAide {
					index = i
				}
			}

			i := index
			minion := ce.Aides[i]
			contactList := minion.Config.GetContactsListCopy()
			// copy out the list of contacts.
			// minion.Config.AccessContactsList(func(config *ContactStructConfig, listOfCi *list.List) {
			// 	l := listOfCi
			// 	e := l.Front()
			// 	for ; e != nil; e = e.Next() {
			// 		cc := e.Value.(ContactInterface)
			// 		contactList = append(contactList, cc)
			// 	}
			// })

			ce.Aides[i] = ce.Aides[len(ce.Aides)-1] // Copy last element to index i.
			ce.Aides[len(ce.Aides)-1] = nil         // Erase last element (write zero value).
			ce.Aides = ce.Aides[:len(ce.Aides)-1]   // shorten list

			// close them all
			for _, cc := range contactList {
				cc.Close(errors.New("routine maintainance a1"))
			}
			//for _, cc := range minion.Looker.upstreamRouter.contacts {
			//	cc.Close(errors.New("routine maintainance a2"))
			//}
			for _, ex := range ce.Gurus {
				ex.Looker.FlushMarkerAndWait()
			}
			for _, ex := range ce.Aides {
				ex.Looker.FlushMarkerAndWait()
			}
		}
	}

	// now, same routine for gurus
	if expansion.ChangeGurus > 0 {
		sample := ce.Gurus[0]
		n := strconv.FormatInt(int64(len(ce.Gurus)), 10)
		newName := "guru" + n
		n1 := NewExecutive(100, newName, sample.getTime, true, ce)
		ce.Gurus = append(ce.Gurus, n1)
		GuruNameToConfigMap[newName] = n1 // for test
		ce.currentGuruList = append(ce.currentGuruList, newName)

		for _, ex := range ce.Gurus {
			ex.Looker.SetUpstreamNames(ce.currentGuruList, ce.currentGuruList)
		}
		for _, aide := range ce.Aides {
			aide.Looker.SetUpstreamNames(ce.currentGuruList, ce.currentGuruList)
		}
		for _, ex := range ce.Gurus {
			ex.Looker.FlushMarkerAndWait()
		}
		for _, ex := range ce.Aides {
			ex.Looker.FlushMarkerAndWait()
		}
	} else if expansion.ChangeGurus < 0 {
		// we can only shrink if the result won't just grow again.
		// with some (10%) margin.
		if true {
			// we can shrink, which one?
			index := 0
			index = len(ce.Gurus) - 1 // always the last one with gurus
			i := index
			minion := ce.Gurus[i]

			ce.Gurus[i] = ce.Gurus[len(ce.Gurus)-1] // Copy last element to index i.
			ce.Gurus[len(ce.Gurus)-1] = nil         // Erase last element (write zero value).
			ce.Gurus = ce.Gurus[:len(ce.Gurus)-1]   // shorten list
			ce.currentGuruList = ce.currentGuruList[0:index]

			contactList := minion.Config.GetContactsListCopy()

			for _, ex := range ce.Gurus {
				ex.Looker.SetUpstreamNames(ce.currentGuruList, ce.currentGuruList)
			}
			for _, aide := range ce.Aides {
				aide.Looker.SetUpstreamNames(ce.currentGuruList, ce.currentGuruList)
			}
			// we need to wait?
			for _, cc := range contactList {
				cc.Close(errors.New("routine maintainance g1"))
			}
			// for _, cc := range minion.Looker.upstreamRouter.contacts {
			// 	cc.Close(errors.New("routine maintainance g2"))
			// }

			for _, ex := range ce.Gurus {
				ex.Looker.FlushMarkerAndWait()
			}
			for _, ex := range ce.Aides {
				ex.Looker.FlushMarkerAndWait()
			}

		}
	}
}

// GetSubsCount returns a count of how many names it's remembering.
// it also returns a fraction of buffer usage where 0.0 is empty and 1.0 is full.
func (ex *Executive) GetSubsCount() (int, float64) {
	subscriptions, queuefraction := ex.Looker.GetAllSubsCount()
	return subscriptions, queuefraction
}

// GetExecutiveStats is fractions relative to the limits.
func (ex *Executive) GetExecutiveStats() *ExecutiveStats {

	now := ex.getTime()

	// call the looker to get the dirt on subscriptions and the buffers
	subscriptions, queuefraction := ex.Looker.GetAllSubsCount()
	_ = subscriptions

	// // scan the contacts for byte rates
	// contactCount := ex.Config.Len()
	// inputs := int64(0)
	// outputs := int64(0)
	// times := int64(0)

	// fixme: have stats call billingAccumulator on heartbeat.
	// ex.Config.AccessContactsList(func(config *ContactStructConfig, listOfCi *list.List) {
	// 	e := listOfCi.Front()
	// 	for ; e != nil; e = e.Next() {
	// 		cc, ok := e.Value.(ContactInterface)
	// 		if !ok {
	// 			fmt.Println("not a ci?")
	// 		}
	// 		in, out, dt := cc.GetRates(now)
	// 		inputs += int64(in)
	// 		outputs += int64(out)
	// 		times += int64(dt)
	// 	}
	// })
	// if times <= 0 {
	// 	times = 1
	// }
	stats := &ExecutiveStats{}
	//statsex := tokens.KnotFreeContactStats{}
	ex.Billing.GetStats(now, &stats.KnotFreeContactStats)

	stats.IsGuru = ex.isGuru

	runtime.GC()
	var gstats runtime.MemStats
	runtime.ReadMemStats(&gstats)
	stats.Memory = int64(gstats.HeapAlloc)

	stats.Input = stats.Input / ex.Limits.Input
	stats.Output = stats.Output / ex.Limits.Output
	stats.Connections = stats.Connections / ex.Limits.Connections
	stats.Subscriptions = stats.Subscriptions / ex.Limits.Subscriptions

	stats.Buffers = queuefraction

	stats.Limits = ex.Limits
	stats.Name = ex.Name
	stats.TCPAddress = ex.GetTCPAddress()
	stats.HTTPAddress = ex.GetHTTPAddress()
	return stats
}

// Heartbeat one per 10 sec.
func (ex *Executive) Heartbeat(now uint32) {

	connectionsTotal.Set(float64(ex.Config.Len()))
	subscriptions, queuefraction := ex.Looker.GetAllSubsCount()
	topicsTotal.Set(float64(subscriptions))
	qFullness.Set(float64(queuefraction))

	ex.Looker.Heartbeat(now)

	timer := prometheus.NewTimer(heartbeatContactsDuration)
	defer timer.ObserveDuration()

	contactList := ex.Config.GetContactsListCopy()

	for _, ci := range contactList {
		ci.Heartbeat(now)
	}
}

// Heartbeat everyone when testing
func (ce *ClusterExecutive) Heartbeat(now uint32) {

	for _, ex := range ce.Aides {
		ex.Heartbeat(now)
	}
	for _, ex := range ce.Gurus {
		ex.Heartbeat(now)
	}
}

// GetLowerContactsCount is how many tcp sessions do we have going on at the bottom
// func (ex *Executive) GetLowerContactsCount() int {
// 	return ex.Config.list.Len()
// }

// GuruNameToConfigMap for ease of unit test.
var GuruNameToConfigMap map[string]*Executive

func init() {
	GuruNameToConfigMap = make(map[string]*Executive)
}

// GetSubsCount returns count of all the subscriptions in all the lookup tables.
// this is really only good for test.
func (ce *ClusterExecutive) GetSubsCount() int {
	count := 0
	for _, ex := range ce.Aides {
		c, _ := ex.GetSubsCount()
		count += c
	}
	for _, ex := range ce.Gurus {
		c, _ := ex.GetSubsCount()
		count += c
	}
	return count
}

// GetHTTPAddress is a getter
func (ex *Executive) GetHTTPAddress() string {
	return ex.httpAddress
}

// GetTCPAddress is a getter
func (ex *Executive) GetTCPAddress() string {
	return ex.tcpAddress
}

// GetTextAddress is a getter
func (ex *Executive) GetTextAddress() string {
	return ex.textAddress
}

// GetMQTTAddress is a getter
func (ex *Executive) GetMQTTAddress() string {
	return ex.mqttAddress
}

// DeepCopyInto the slow way
func (in *ExecutiveStats) DeepCopyInto(out *ExecutiveStats) {
	if in == nil {
	}
	jbytes, err := json.Marshal(in)
	_ = err
	json.Unmarshal(jbytes, out)
}

// DeepCopy is an atwgenerated deepcopy function, copying the receiver, creating a new AppService.
func (in *ExecutiveStats) DeepCopy() *ExecutiveStats {
	if in == nil {
		return nil
	}
	out := new(ExecutiveStats)
	in.DeepCopyInto(out)
	return out
}

// WaitForActions is a utility for unit tests.
// we must wait for things to happen during tests
// we pretend to get service from the operator.
func (ce *ClusterExecutive) WaitForActions() {

	nodes := make([]*Executive, 0, len(ce.Aides)+len(ce.Gurus))
	for _, ex := range ce.Aides {
		ex.WaitForActions()
		nodes = append(nodes, ex)
	}
	for _, ex := range ce.Gurus {
		ex.WaitForActions()
		nodes = append(nodes, ex)
	}

	stats := make([]*ExecutiveStats, 0, len(nodes))
	// are we tcp mode?
	if ce.isTCP {

		when := uint32(0)
		for _, ex := range nodes {
			stat, err := GetServerStats(ex.GetHTTPAddress())
			if err == nil {
				stats = append(stats, stat)
			} else {
				fmt.Println("GetServerStats fail", err)
			}
			when = ex.getTime()
		}
		clusterStats := ClusterStats{}
		clusterStats.When = when
		clusterStats.Stats = stats

		for _, ex := range nodes {
			PostClusterStats(&clusterStats, ex.GetHTTPAddress())
		}
	} else {
		/// get them directly
		when := uint32(0)
		for _, ex := range nodes {
			stat := ex.GetExecutiveStats()
			stats = append(stats, stat)
			when = ex.getTime()
		}
		clusterStats := ClusterStats{}
		clusterStats.When = when
		clusterStats.Stats = stats

		for _, ex := range nodes {
			ex.ClusterStats = &clusterStats
		}
	}
	time.Sleep(1 * time.Millisecond)
	for _, ex := range nodes {
		ex.WaitForActions()
	}
	time.Sleep(1 * time.Millisecond)
	for _, ex := range nodes {
		ex.WaitForActions()
	}
	time.Sleep(1 * time.Millisecond)
}

// WaitForActions needs to be properly implemented.
// The we inject tracer packets with wait groups into q's
// and then wait for that.
func (ex *Executive) WaitForActions() {

	if ex != nil {
		ex.Looker.FlushMarkerAndWait()
		// can we flush the lower and upper contacts too ?
		// TODO:
	}
}

func SpecialPrint(p *packets.PacketCommon, fn func()) {
	val, ok := p.GetOption("debg")
	if len(string(val)) > 0 {
		fmt.Println("SpecialPrint ", string(val))
	}
	if ok && (string(val) == "[12345678]" || string(val) == "12345678") {
		fn()
	}
}
