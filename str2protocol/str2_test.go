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

package str2protocol

import (
	"bytes"
	"fmt"
)

type TestPacketStuff int

// ExampleTestPacketStuff is a test.
func ExampleTestPacketStuff() {

	examples := []int{0x200000 - 1, 10, 127, 128, 0x200, 0x4000 - 1, 0x4000, 0x432100}
	for _, val := range examples {

		var b bytes.Buffer // A Buffer needs no initialization.

		err := WriteVarLenInt(uint32(val), uint8(0), &b)
		if err != nil {
			fmt.Println(err)
		}
		got, err := ReadVarLenInt(&b)
		if err != nil {
			fmt.Println(err)
		}
		if got != val {
			fmt.Println(val, " vs ", got)
		}
	}
	{
		var b bytes.Buffer
		str := Str2{}
		str.cmd = 'A' // aka 65
		str.args = []Bstr{Bstr("aa"), Bstr("B"), Bstr("cccccccccc")}
		err := str.Write(&b)
		if err != nil {
			fmt.Println(err)
		}
		str2, err := ReadStr2(&b)
		if len(str2.args) != 3 {
			fmt.Println("len(str2.args) != 3")
		}
		if string(str.args[0]) != "aa" {
			fmt.Println("string(str.args[0]) != aa")
		}
		if string(str.args[1]) != "B" {
			fmt.Println("string(str.args[1]) != B")
		}
		if string(str.args[2]) != "cccccccccc" {
			fmt.Println("string(str.args[2]) != cccccccccc")
		}
	}

	fmt.Println("done")

	// Output: done

}

func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}

func bToKb(b uint64) uint64 {
	return b / 1024
}
