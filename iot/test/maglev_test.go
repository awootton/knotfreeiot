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

package iot_test

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/dgryski/go-maglev"
)

// todo: clean up the workbench, put away the tools.
// also see: https://github.com/dgryski/go-jump/blob/master/jump.go
// https://github.com/dgryski/go-rendezvous/blob/master/rdv.go  which might be better. todo.
// https://github.com/dgryski/go-ketama

func TestDeltas(t *testing.T) {

	if t != nil { // it's not really a test. don't print this stuff
		return
	}

	names1 := getNames(1)
	mapped1 := getMapped(names1)
	//fmt.Println("names are ", names1)
	//fmt.Println("mapped is ", mapped1)

	names2 := getNames(2)
	mapped2 := getMapped(names2)
	//fmt.Println("names are ", names2)
	//fmt.Println("mapped is ", mapped2)

	names3 := getNames(3)
	mapped3 := getMapped(names3)
	//fmt.Println("names are ", names3)
	//fmt.Println("mapped is ", mapped3)

	fmt.Println("1 to 2 ")
	delta(mapped1, mapped2)

	fmt.Println("2 to 3 ")
	delta(mapped2, mapped3)

	for i := 4; i < 10; i++ {
		names1 := getNames(i - 1)
		mapped1 := getMapped(names1)

		names2 := getNames(i)
		mapped2 := getMapped(names2)
		fmt.Printf("%v to %v ", i-1, i)
		delta(mapped1, mapped2)
	}

}

func delta(m1, m2 []string) {
	for i := 0; i < 32; i++ {
		fmt.Print("\n")
		for j := 0; j < 32; j++ {
			s1 := m1[j+i*32]
			s2 := m2[j+i*32]
			if s1 == s2 {
				fmt.Print(".")
			} else {
				fmt.Print("#")
			}
		}
	}
	fmt.Print("\n")
}

func getNames(size int) []string {
	var names []string
	for i := 0; i < size; i++ {
		names = append(names, fmt.Sprintf("backend-%d", i))
	}
	return names
}

func getMapped(names []string) []string {
	hsize := maglev.SmallM
	hsize = 2053
	table := maglev.New(names, uint64(hsize))
	var mapped []string
	for i := 0; i < 1024; i++ {
		idx := table.Lookup(uint64(i))
		mapped = append(mapped, names[idx])
	}
	return mapped
}

func TestDistribution(t *testing.T) {

	hsize := maglev.SmallM
	//hsize = 1031

	//tablesplit := 1024 * 8

	const size = 256

	var names []string
	for i := 0; i < size; i++ {
		names = append(names, fmt.Sprintf("backend-%d", i))
	}

	//fmt.Println("names are ", names)

	table := maglev.New(names, uint64(hsize))

	// var mapped []string
	// for i := 0; i < tablesplit; i++ {
	// 	idx := table.Lookup(uint64(i))
	// 	mapped = append(mapped, names[idx])
	// }
	//fmt.Println("mapped is ", mapped)

	r := make([]int, size)
	rand.Seed(0)
	for i := 0; i < 1e6; i++ {
		iii := rand.Int63()
		//iii = iii % int64(tablesplit)
		idx := table.Lookup(uint64(iii))
		r[idx]++
	}
	fmt.Println("r len is ", len(r))

	var total int
	var max = 0
	for _, v := range r {
		total += v
		//fmt.Print(v, " ")
		if v > max {
			max = v
		}
	}

	mean := float64(total) / size
	fmt.Printf("max=%v, mean=%v, peak-to-mean=%v", max, mean, float64(max)/mean)
}
