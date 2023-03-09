// Copyright 2020 Alan Tracey Wootton
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
	"math"

	"github.com/awootton/knotfreeiot/tokens"
)

/***  Billing flow.

On Connnect (in Push(..)) the Contact remembers the token; which contains a unique JWTID.
The JWTID will be our billing id.

The connect send a subscribe to the JWTID (see contacts expectToken) and also
sends "statsmax" with the subscribe and statsmax is a KnotFreeContactStats from the token.

In subscribe.go we notice the statsmax KnotFreeContactStats and construct a BillingAccumulator
and add it to the watcher with the name of "bill". There is a getter for that. watchedItem.GetBilling()

In the Heartbeat of a Contact we periodically (30 then 60 sec) construct a Stats
and we Push it as a Send with option "add-stats".
When the publish.go gets the stats message we do a GetBilling() (aka GetOption("bill"))
 and we have special case for that and we BillingAccumulator.Add(stats)

Later, in the heartbeat of lookup, we check if it's a billing topic and if we're 'over' our maximums.
If over then we construct a send with the option "error" as the message. We send it *down* like a
normal publish. WriteDownstream(..)

Then, in all the implementations of WriteDownstream check for error and make a disconnect packet if needed.
See HasError()

The subscribe packets, as they pass the contact going upwards, are all going to need to carry the alias
of the jwtid from the token. See the Push(...) in Contacts.go. It will become watchedTopic.jwtid in subscribe.go.

Then, during the lookup-table Heartbeat (heartBeatCallBack) we periodically (30 then 300 sec) construct a
StatsWithTime and Send it to the odd contact Looker.contactToAnyAide where it will meet the topic with the BillingAccumulator
and that will get incremented as with the contacts. As with the contacts the looker will service the accumulator topic
and send down our special error Send{}

*/

// there's a tokens.KnotFreeContactStats which has the maximum allowable values
// there's iot.StatsWithTime which reports usage.

// how to accumulate:
// we only want to go back about an hour. about.
// we'll use 10 min buckets and add 4 of them.

const bucketSpanTime = 10 * 60 // 10 minutes

// BillingAccumulator is terse
type BillingAccumulator struct {
	//
	a [4]StatsWithTime
	i int // this will be always mod len(a) and will increment.

	max tokens.KnotFreeContactStats

	name string
}

// StatsWithTime
type StatsWithTime struct {
	tokens.KnotFreeContactStats
	Start uint32 `json:"st"`
	Used  bool   `json:"u"`
}

// Stats
type Stats struct {
	tokens.KnotFreeContactStats
}

// AddUsage accumulates the stats into the BillingAccumulator
func (ba *BillingAccumulator) AddUsage(stats *tokens.KnotFreeContactStats, now uint32, deltat int) {

	c := &ba.a[ba.i]
	if !c.Used {
		// first time. init with some time. 60 min free
		for i := 0; i < len(ba.a); i++ {
			ba.a[i].Start = now - uint32(bucketSpanTime*(len(ba.a)-i))
		}
		c.Start = now - uint32(deltat)
		c.Used = false
	}
	c.Used = true

	c.Connections += stats.Connections
	c.Input += stats.Input
	c.Output += stats.Output
	c.Subscriptions += stats.Subscriptions

	if (c.Start + bucketSpanTime) < now {

		previousTime := c.Start + bucketSpanTime
		ba.i = (ba.i + 1) % len(ba.a)
		next := &ba.a[ba.i]

		//fmt.Println("rolling forward")
		BucketClear(next)

		c = next
		c.Start = previousTime
		c.Used = true
		c.Connections = 0
		c.Input = 0
		c.Output = 0
		c.Subscriptions = 0
	}

	if ba.max.Subscriptions == 1 && stats.Subscriptions != 0 { // the test in billing_test
		subs := ba.GetSubscriptions(now)
		fmt.Println("Subscriptions now", subs, ba.name)
	}

	//fmt.Println("added", stats.Subscriptions)
	//fmt.Println("Subscriptions now", ba.GetSubscriptions(now), ba.name)
	//fmt.Println("doing an add")
}

// AreUnderMax returns if the stats are under the limits and, if not true,
// returns a message about it.
func (ba *BillingAccumulator) AreUnderMax(now uint32) (bool, string) {
	weGood := true
	why := ""
	current := tokens.KnotFreeContactStats{}
	ba.GetStats(now, &current)

	if current.Connections > (ba.max.Connections + .1) {
		weGood = false
		why += fmt.Sprintf(" BILLING ERROR %v connections > %v", current.Connections, ba.max.Connections)
	}
	if current.Input > (ba.max.Input + .1*100) {
		weGood = false
		why += fmt.Sprintf(" BILLING ERROR %v bytes in > %v/s", current.Input, ba.max.Input)
	}
	if current.Output > (ba.max.Output + .1) {
		weGood = false
		why += fmt.Sprintf(" BILLING ERROR %v bytes out > %v/s", current.Output, ba.max.Output)
	}
	if current.Subscriptions > (ba.max.Subscriptions + .1) {
		weGood = false
		why += fmt.Sprintf(" BILLING ERROR %v subscriptions > %v", current.Subscriptions, ba.max.Subscriptions)
	}

	return weGood, why
}

const lookback = 4

// GetInput - we sum up some prevous buckets and divide.
// do we need to sync with Add? No, because all access to this goes through a q in lookup.
func (ba *BillingAccumulator) GetInput(now uint32) float64 {

	statsResult := &tokens.KnotFreeContactStats{}
	ba.GetStats(now, statsResult)
	return statsResult.Input

	// vals := float64(0)
	// times := float64(0)
	// tmpnow := now
	// for i := 0; i < lookback; i++ { // walk backwards
	// 	index := (ba.i - i + len(ba.a)) % len(ba.a)
	// 	c := &ba.a[index]
	// 	if !c.Used {
	// 		break
	// 	}
	// 	vals += c.Input
	// 	buckettime := tmpnow - c.Start
	// 	tmpnow -= buckettime
	// 	times += float64(buckettime)
	// }
	// if times == 0 {
	// 	return 0
	// }
	// f := vals / times
	// return f
}

// GetConnections is
func (ba *BillingAccumulator) GetConnections(now uint32) float64 {
	statsResult := &tokens.KnotFreeContactStats{}
	ba.GetStats(now, statsResult)
	return statsResult.Connections

	// vals := float64(0)
	// times := float64(0)
	// tmpnow := now
	// for i := 0; i < lookback; i++ {
	// 	index := (ba.i - i + len(ba.a)) % len(ba.a)
	// 	c := &ba.a[index]
	// 	if !c.Used {
	// 		break
	// 	}
	// 	vals += c.Connections
	// 	buckettime := tmpnow - c.Start
	// 	tmpnow -= buckettime
	// 	times += float64(buckettime)
	// }
	// if times == 0 {
	// 	return 0
	// }
	// f := vals / times
	// return f
}

// GetOutput is
func (ba *BillingAccumulator) GetOutput(now uint32) float64 {

	statsResult := &tokens.KnotFreeContactStats{}
	ba.GetStats(now, statsResult)
	return statsResult.Output

	// vals := float64(0)
	// times := float64(0)
	// tmpnow := now
	// for i := 0; i < lookback; i++ {
	// 	index := (ba.i - i + len(ba.a)) % len(ba.a)
	// 	c := &ba.a[index]
	// 	if !c.Used {
	// 		break
	// 	}
	// 	vals += c.Output
	// 	buckettime := tmpnow - c.Start
	// 	tmpnow -= buckettime
	// 	times += float64(buckettime)
	// }
	// if times == 0 {
	// 	return 0
	// }
	// f := vals / times
	// return f
}

// GetSubscriptions is
func (ba *BillingAccumulator) GetSubscriptions(now uint32) float64 {

	statsResult := &tokens.KnotFreeContactStats{}
	ba.GetStats(now, statsResult)
	return statsResult.Subscriptions

	// vals := float64(0)
	// times := float64(0)
	// tmpnow := now
	// for i := 0; i < lookback; i++ {
	// 	index := (ba.i - i + len(ba.a)) % len(ba.a)
	// 	c := &ba.a[index]
	// 	if !c.Used {
	// 		break
	// 	}
	// 	vals += c.Subscriptions
	// 	buckettime := tmpnow - c.Start
	// 	tmpnow -= buckettime
	// 	times += float64(buckettime)
	// }
	// if times == 0 {
	// 	return 0
	// }
	// f := vals / times
	// return f
}

// GetStats calcs them all at once into dest.
// dest should be zeroed before calling.
func (ba *BillingAccumulator) GetStats(now uint32, dest *tokens.KnotFreeContactStats) {

	times := float64(0)
	tmpnow := now
	for i := 0; i < lookback; i++ {
		index := (ba.i - i + len(ba.a)) % len(ba.a)
		c := &ba.a[index]
		if !c.Used {
			break
		}
		dest.Connections += c.Connections
		dest.Input += c.Input
		dest.Output += c.Output
		dest.Subscriptions += c.Subscriptions
		buckettime := tmpnow - c.Start
		tmpnow -= buckettime
		times += float64(buckettime)
	}
	if times == 0 {
		return
	}
	dest.Connections /= times
	dest.Input /= times
	dest.Output /= times
	dest.Subscriptions /= times

	dest.Connections = math.Floor(dest.Connections*100) / 100
	dest.Input = math.Floor(dest.Input*100) / 100
	dest.Output = math.Floor(dest.Output*100) / 100
	dest.Subscriptions = math.Floor(dest.Subscriptions*100) / 100
}

// BucketCopy is
func BucketCopy(src *tokens.KnotFreeContactStats, dest *tokens.KnotFreeContactStats) {
	dest.Connections = src.Connections
	dest.Input = src.Input
	dest.Output = src.Output
	dest.Subscriptions = src.Subscriptions
}

// BucketClear is
func BucketClear(dest *StatsWithTime) {
	dest.Connections = 0
	dest.Input = 0
	dest.Output = 0
	dest.Subscriptions = 0
}

// BucketSubtract is
func BucketSubtract(src *StatsWithTime, dest *StatsWithTime) {
	dest.Connections -= src.Connections
	dest.Input -= src.Input
	dest.Output -= src.Output
	dest.Subscriptions -= src.Subscriptions
}
