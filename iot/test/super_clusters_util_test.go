// Copyright 2019,2021 Alan Tracey Wootton
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

package iot_test

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/awootton/knotfreeiot/iot"
	"github.com/awootton/knotfreeiot/tokens"
)

func StartClusterOfClusters(timeGetter func() uint32) [][]*iot.ClusterExecutive {

	tokens.LoadPublicKeys()

	//gurus := 2
	aidesCount := 2
	superClusterWidth := 2
	superClusterHeight := 2

	var clusterExecs [][]*iot.ClusterExecutive

	// starting from the bottom up we make two clusters of 2x2 aides and gurus and put them in
	for h := 0; h < superClusterHeight; h++ {

		width := superClusterWidth - h
		if width <= 0 {
			width = 1
		}
		var arow []*iot.ClusterExecutive
		for w := 0; w < width; w++ {
			suffix := "_" + strconv.Itoa(w) + "_" + strconv.Itoa(h)
			isTCP := false
			ce := iot.MakeSimplestCluster(timeGetter, isTCP, aidesCount, suffix)
			arow = append(arow, ce)
		}
		clusterExecs = append(clusterExecs, arow)
	}
	// now we have to wire up the gurus of the lower levels to the aides of upper levels
	// TODO:
	for h := 0; h < superClusterHeight-1; h++ {
		row := clusterExecs[h]
		rowAbove := clusterExecs[h+1]
		var gurusInRow []*iot.Executive
		for _, ce := range row {
			for _, exe := range ce.Gurus {
				gurusInRow = append(gurusInRow, exe)
			}
		}
		var aidesInRowAbove []*iot.Executive
		for _, ce := range rowAbove {
			for _, exe := range ce.Aides {
				aidesInRowAbove = append(aidesInRowAbove, exe)
			}
		}
		for i, guru := range gurusInRow {
			if i >= len(aidesInRowAbove) {
				i = len(aidesInRowAbove) - 1
			}
			aide := aidesInRowAbove[i]
			fmt.Println("connecting " + guru.Name + " to " + aide.Name)

			iot.ConnectGuruToSuperAide(guru, aide)
		}
	}

	return clusterExecs
}

func getLeafAideCount(clusters [][]*iot.ClusterExecutive) int {

	bottomRow := clusters[0]
	count := 0
	for i := 0; i < len(bottomRow); i++ {
		ce := bottomRow[i]
		count += len(ce.Aides)
	}
	return count
}

func getAnAide(clusters [][]*iot.ClusterExecutive, index int) *iot.Executive {

	count := getLeafAideCount(clusters)
	if index > count {
		index = count - 1
	}

	bottomRow := clusters[0]
	offset := 0
	i := 0
	var ce *iot.ClusterExecutive
	for true {
		ce = bottomRow[i]
		nextOffset := offset + len(ce.Aides)
		if nextOffset > index {
			break
		}
		i++
		offset += nextOffset
	}
	i = index - offset

	aide := ce.Aides[i]

	return aide
}

//eg. 	IterateAndWait(t, func() bool { return true }, "test")

//eg.	
//	got := ""
// 	IterateAndWait(t, func() bool {
// 		got = contact9.(*testContact).getResultAsString()
// 		return got != "no message received"
// 	}, "timed out waiting for can you hear me now")

func IterateAndWait(t *testing.T, fn func() bool, message string) {
	for i := 0; i < 1000; i++ {
		if fn() {
			return
		}
		time.Sleep(1 * time.Second)
	}
	t.Error(message)
}
