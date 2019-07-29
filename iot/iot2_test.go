package iot

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
