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
	"fmt"
)

// ExampleGetFirstWord is a test. I do with I didn't have to make GetFirstWord public just to write this test.
func ExampleGetFirstWord() {

	a, b := GetFirstWord("aa bb cc")
	if a != "aa" || b != "bb cc" {
		fmt.Println("oops")
	}
	a, b = GetFirstWord(b)
	if a != "bb" || b != "cc" {
		fmt.Println("oops2")
	}

	a, b = GetFirstWord("")
	if a != "" || b != "" {
		fmt.Println("oops3")
	}

	a, b = GetFirstWord("aaa")
	if a != "aaa" || b != "" {
		fmt.Println("oops4")
	}

	a, b = GetFirstWord("aaa  bbb  ")
	if a != "aaa" || b != "bbb" {
		fmt.Println("oops5")
	}

	a, b = GetFirstWord("  aaa      bbb cc dd     ")
	if a != "aaa" || b != "bbb cc dd" {
		fmt.Println("oops6")
	}

	a, b = GetFirstWord(" \" bbb ccc \"  \" ddd eee \" ")
	if a != "bbb ccc" || b != "\" ddd eee \"" {
		fmt.Println("oops7")
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
