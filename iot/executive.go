// Copyright 2019,2020 Alan Tracey Wootton
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
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

// Utility and controller struct and functions for LookupTable

// Executive is one instance of iot service.
// TODO: Executive and LookupTableStruct and ContactStructConfig are really just one thing.
type Executive struct {
	Looker *LookupTableStruct
	Config *ContactStructConfig
	Name   string
	isGuru bool

	httpAddress string // eg 33.44.55.66:7374 or localhost:8089 or something
	tcpAddress  string // eg 33.44.55.66:7374 or localhost:8089 or something
	textAddress string // eg 33.44.55.66:7374 or localhost:8089 or something
	mqttAddress string // eg 33.44.55.66:7374 or localhost:8089 or something

	getTime func() uint32

	Limits *ExecutiveLimits

	IAmBadError error // if something happened to simply ruin us and we're quitting.
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
}

// ExecutiveLimits will be how we tell if the ex is 'full'
type ExecutiveLimits struct {
	Connections   int `json:"con"`
	BytesPerSec   int `json:"bps"`
	Subscriptions int `json:"sub"`
}

// ExecutiveStats is including limits sometimes
type ExecutiveStats struct {
	Connections   float64 `json:"con"`
	Subscriptions float64 `json:"sub"`
	Buffers       float64 `json:"buf"`
	BytesPerSec   float64 `json:"bps"`
	Name          string  `json:"name"`
	HTTPAddress   string  `json:"http"`
	TCPAddress    string  `json:"tcp"`

	Limits *ExecutiveLimits `json:"limits"`
}

// TestLimits is for testsvar TestLimits = ExecutiveLimits{16, 10, 64}
// eg 16 contects is 100% or 10 bytes per sec is 100% or 64 contacts is 100%
var TestLimits = ExecutiveLimits{16, 10, 64}

// GetNewContact add a contect to the least used of the aides
func (ce *ClusterExecutive) GetNewContact(factory ContactFactory) ContactInterface {
	min := 1 << 30
	var smallestAide *Executive
	for _, aide := range ce.Aides {
		cons, fract := aide.Looker.GetAllSubsCount()
		if cons < min {
			min = cons
			smallestAide = aide
		}
		_ = fract
	}
	if smallestAide == nil {
		return nil // fixme return error
	}
	//fmt.Println("smallest aide is ", smallestAide.Name)
	cc := factory(smallestAide.Config)
	return cc
}

// AttachContact add a contect to the least used of the aides
// it's for an existing contact that's reconnecting.
func (ce *ClusterExecutive) AttachContact(cc ContactInterface, attacher ContactAttach) {
	max := -1
	var smallestAide *Executive
	for _, aide := range ce.Aides {
		cons, fract := aide.Looker.GetAllSubsCount()
		if cons > max {
			max = cons
			smallestAide = aide
		}
		_ = fract
	}
	if smallestAide == nil {
		fmt.Println("fix mesmallestAide fatal ")
		return // fixme return error
	}
	attacher(cc, smallestAide.Config)
}

// MakeSimplestCluster is just for testing as k8s doesn't work like this.
func MakeSimplestCluster(timegetter func() uint32, nameResolver GuruNameResolver, isTCP bool, aideCount int) *ClusterExecutive {

	ce := &ClusterExecutive{}
	ce.isTCP = isTCP
	if isTCP {
		ce.currentPort = 9000
	}

	ce.limits = &TestLimits

	// set up
	guru0 := NewExecutive(100, "guru0", timegetter, true)
	GuruNameToConfigMap["guru0"] = guru0
	guru0.Config.ce = ce
	ce.Gurus = append(ce.Gurus, guru0)
	guru0.Looker.NameResolver = nameResolver

	if isTCP {
		guru0.httpAddress = ce.GetNextAddress()
		guru0.tcpAddress = ce.GetNextAddress()
		guru0.textAddress = ce.GetNextAddress()
		guru0.mqttAddress = ce.GetNextAddress()

		MakeTCPExecutive(guru0, guru0.tcpAddress)
		MakeHTTPExecutive(guru0, guru0.httpAddress)
	}
	ce.currentGuruList = []string{"guru0"}
	ce.currentAddressList = []string{guru0.tcpAddress}

	for i := int64(0); i < int64(aideCount); i++ {
		aide1 := NewExecutive(100, "aide"+strconv.FormatInt(i, 10), timegetter, false)
		aide1.Config.ce = ce
		ce.Aides = append(ce.Aides, aide1)
		aide1.Looker.NameResolver = nameResolver

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
	}

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
		// guru0.Looker.SetUpstreamNames(ce.currentGuruList, ce.currentAddressList)
		// for _, aide := range ce.Aides {
		// 	aide.Looker.SetUpstreamNames(ce.currentGuruList, ce.currentAddressList)
		// }

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
func MakeTCPMain(name string, limits *ExecutiveLimits, token string) *ClusterExecutive {

	isTCP := true
	timegetter := func() uint32 {
		return uint32(time.Now().Unix())
	}
	nameResolver := TCPNameResolver

	ce := &ClusterExecutive{}
	ce.isTCP = isTCP

	ce.currentPort = 80

	ce.limits = limits

	aide1 := NewExecutive(100, name, timegetter, false)
	aide1.Limits = limits
	aide1.Config.ce = ce
	ce.Aides = append(ce.Aides, aide1)
	aide1.Looker.NameResolver = nameResolver

	myip := os.Getenv("MY_POD_IP")

	aide1.httpAddress = myip + ":8080"
	aide1.tcpAddress = myip + ":8384"
	aide1.textAddress = myip + ":7465"
	aide1.mqttAddress = myip + ":1883"

	MakeTCPExecutive(aide1, aide1.tcpAddress)
	MakeTextExecutive(aide1, aide1.textAddress)
	MakeHTTPExecutive(aide1, aide1.httpAddress)
	MakeMqttExecutive(aide1, aide1.mqttAddress)

	return ce
}

// NewExecutive A wrapper to hold and operate
func NewExecutive(sizeEstimate int, aname string, timegetter func() uint32, isGuru bool) *Executive {

	look0 := NewLookupTable(sizeEstimate, aname, isGuru)
	config0 := NewContactStructConfig(look0)
	config0.Name = aname

	e := Executive{}
	e.Looker = look0
	e.Config = config0
	e.Name = aname
	e.getTime = timegetter
	e.Limits = &TestLimits
	e.isGuru = isGuru
	return &e

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
			max := 0.0
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
		anaide := ce.Aides[0]
		n := strconv.FormatInt(int64(len(ce.Aides)), 10)
		aide1 := NewExecutive(100, "aide"+n, anaide.getTime, false)
		ce.Aides = append(ce.Aides, aide1)
		aide1.Looker.NameResolver = anaide.Looker.NameResolver
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
			//	subsTotal /= float64(len(ce.Aides))
			//	subsTotal /= float64(ce.limits.subscriptions)
			minion.Config.listlock.Lock()
			contactList := make([]ContactInterface, 0, minion.Config.list.Len())
			ce.Aides[i] = ce.Aides[len(ce.Aides)-1] // Copy last element to index i.
			ce.Aides[len(ce.Aides)-1] = nil         // Erase last element (write zero value).
			ce.Aides = ce.Aides[:len(ce.Aides)-1]   // shorten list
			l := minion.Config.GetContactsList()
			e := l.Front()
			for ; e != nil; e = e.Next() {
				cc := e.Value.(ContactInterface)
				contactList = append(contactList, cc)
			}
			minion.Config.listlock.Unlock()
			for _, cc := range contactList {
				cc.Close(errors.New("routine maintainance a1"))
			}
			for _, cc := range minion.Looker.upstreamRouter.contacts {
				cc.Close(errors.New("routine maintainance a2"))
			}
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
		n1 := NewExecutive(100, newName, sample.getTime, true)
		ce.Gurus = append(ce.Gurus, n1)
		GuruNameToConfigMap[newName] = n1 // for test
		n1.Looker.NameResolver = sample.Looker.NameResolver
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
			minion.Config.listlock.Lock()
			contactList := make([]ContactInterface, 0)
			ce.Gurus[i] = ce.Gurus[len(ce.Gurus)-1] // Copy last element to index i.
			ce.Gurus[len(ce.Gurus)-1] = nil         // Erase last element (write zero value).
			ce.Gurus = ce.Gurus[:len(ce.Gurus)-1]   // shorten list
			ce.currentGuruList = ce.currentGuruList[0:index]

			l := minion.Config.GetContactsList()
			e := l.Front()
			for ; e != nil; e = e.Next() {
				cc := e.Value.(ContactInterface)
				contactList = append(contactList, cc)
			}
			minion.Config.listlock.Unlock()
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
			for _, cc := range minion.Looker.upstreamRouter.contacts {
				cc.Close(errors.New("routine maintainance g2"))
			}

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

// GetExecutiveStats is
func (ex *Executive) GetExecutiveStats() *ExecutiveStats {

	stats := &ExecutiveStats{}
	subscriptions, queuefraction := ex.Looker.GetAllSubsCount()
	stats.Buffers = queuefraction
	stats.Subscriptions = float64(subscriptions) / float64(ex.Limits.Subscriptions)
	stats.Connections = float64(ex.GetLowerContactsCount()) / float64(ex.Limits.Connections)
	stats.BytesPerSec = float64(1) / float64(ex.Limits.BytesPerSec)
	stats.Limits = ex.Limits
	stats.Name = ex.Name
	stats.TCPAddress = ex.GetTCPAddress()
	stats.HTTPAddress = ex.GetHTTPAddress()
	return stats
}

// GetLowerContactsCount is how many tcp sessions do we have going on at the bottom
func (ex *Executive) GetLowerContactsCount() int {
	return ex.Config.list.Len()
}

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
