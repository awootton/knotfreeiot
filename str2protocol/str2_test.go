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
	"encoding/hex"
	"fmt"
	"strings"
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
		str.args = [][]byte{[]byte("aa"), []byte("B"), []byte("cccccccccc")}

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

type ToJSON int

// ExampleToJSON is a test.
func ExampleToJSON() {
	cmd := Send{}
	cmd.source = []byte("sourceaddr")
	cmd.destination = []byte("destaddr")
	cmd.data = []byte("some data")
	cmd.options = make(map[string][]byte)
	cmd.options["option1"] = []byte("test")
	addr := "2001:0db8:85a3:0000:0000:8a2e:0370:7334"
	hexstr := strings.ReplaceAll(addr, ":", "")
	decoded, err := hex.DecodeString(hexstr)
	if err != nil {
		fmt.Println("wrong")
	}
	cmd.options["ip"] = decoded
	decoded, err = hex.DecodeString("FFFF00000000000000ABCDEF")
	if err != nil {
		fmt.Println("wrong2")
	}
	cmd.options["z"] = decoded
	cmd.options["option2"] = []byte("На берегу пустынных волн")

	jdata, err := (&cmd).ToJSON()
	fmt.Println(string(jdata))
	_ = err

	//  GetIPV6Option
	fmt.Println(cmd.GetIPV6Option())

	// Output: {"args":[{"a":"sourceaddr"},{"a":"destaddr"},{"a":"some data"},{"a":"z"},{"b64":"//8AAAAAAAAAq83v"},{"a":"option2"},{"utf8":"На берегу пустынных волн"},{"a":"option1"},{"a":"test"},{"a":"ip"},{"b64":"IAENuIWjAAAAAIouA3BzNA"}],"cmd":"P"}
	// [32 1 13 184 133 163 0 0 0 0 138 46 3 112 115 52]

}
