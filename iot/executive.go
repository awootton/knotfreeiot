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
	"strconv"
)

// Utility and controller struct and functions for LookupTable

// Executive is one instance of server
type Executive struct {
	Looker *LookupTableStruct
	Config *ContactStructConfig
	Name   string

	getTime func() uint32

	Limits *ExecutiveLimits
}

// ClusterExecutive is a list of Executive
// helpful for testing.
type ClusterExecutive struct {
	Aides           []*Executive
	Gurus           []*Executive
	limits          *ExecutiveLimits
	currentGuruList []string
}

// ExecutiveLimits will be how we tell if the ex is 'full'
type ExecutiveLimits struct {
	connections   int
	buffers       float32 // 1.0 is fill
	bytesPerSec   int
	subscriptions int
}

// TestLimits is for tests
var TestLimits = ExecutiveLimits{16, 0.5, 10, 75}

// Operate where we pretend to be an Operator and resize the cluster.
func (ce *ClusterExecutive) Operate() {

	needsNewAide := false
	for _, ex := range ce.Aides {
		c, fract := ex.GetSubsCount()
		if c > ce.limits.connections {
			needsNewAide = true
		}
		if fract > ce.limits.buffers {
			needsNewAide = true
		}
		// check bps and subscriptions
		// TODO calc % for each category and return largest and then grow on >0.85 and shrink at <0.75
	}

	if needsNewAide {
		anaide := ce.Aides[0]
		n := strconv.FormatInt(int64(len(ce.Aides)), 10)
		aide1 := NewExecutive(100, "aide"+n, anaide.getTime)
		ce.Aides = append(ce.Aides, aide1)
		aide1.Looker.NameResolver = anaide.Looker.NameResolver
		aide1.Looker.SetUpstreamNames(ce.currentGuruList)
	}

	needsNewGuru := false
	for _, ex := range ce.Gurus {
		c, fract := ex.GetSubsCount()
		if c > ce.limits.connections {
			needsNewGuru = true
		}
		if fract > ce.limits.buffers {
			needsNewGuru = true
		}
		// check bps and subscriptions
		// TODO calc % for each category and return largest and then grow on >0.85 and shrink at <0.75
	}

	if needsNewGuru && false {
		sample := ce.Gurus[0]
		n := strconv.FormatInt(int64(len(ce.Gurus)), 10)
		newName := "guru" + n
		n1 := NewExecutive(100, newName, sample.getTime)
		ce.Gurus = append(ce.Gurus, n1)
		GuruNameToConfigMap[newName] = n1
		n1.Looker.NameResolver = sample.Looker.NameResolver
		ce.currentGuruList = append(ce.currentGuruList, newName)

		for _, aide := range ce.Aides {
			aide.Looker.SetUpstreamNames(ce.currentGuruList)
		}
	}

}

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
		return // fixme return error
	}
	attacher(cc, smallestAide.Config)
}

// MakeSimplestCluster is
func MakeSimplestCluster(timegetter func() uint32, nameResolver GuruNameResolver) *ClusterExecutive {

	ce := ClusterExecutive{}

	ce.limits = &TestLimits

	// set up
	guru0 := NewExecutive(100, "guru0", timegetter)
	GuruNameToConfigMap["guru0"] = guru0

	ce.Gurus = append(ce.Gurus, guru0)

	aide1 := NewExecutive(100, "aide1", timegetter)
	ce.Aides = append(ce.Aides, aide1)
	aide1.Looker.NameResolver = nameResolver

	ce.currentGuruList = []string{"guru0"}
	aide1.Looker.SetUpstreamNames(ce.currentGuruList)

	return &ce
}

// NewExecutive A wrapper to hold and operate
func NewExecutive(sizeEstimate int, aname string, timegetter func() uint32) *Executive {

	look0 := NewLookupTable(sizeEstimate)
	config0 := NewContactStructConfig(look0)
	config0.Name = aname

	e := Executive{}
	e.Looker = look0
	e.Config = config0
	e.Name = aname + "_ex"
	e.getTime = timegetter
	e.Limits = &TestLimits
	return &e

}

// GetSubsCount returns a count of how many names it's remembering.
// it also returns a fraction of buffer usage where 0.0 is empty and 1.0 is full.
func (ex *Executive) GetSubsCount() (int, float32) {
	subscriptions, queuefraction := ex.Looker.GetAllSubsCount()
	return subscriptions, queuefraction
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
