package iot2_test

import (
	"fmt"
	"knotfree/iot2"
)

// ExampleGetFirstWord is a test. I do with I didn't have to make GetFirstWord public just to write this test.
func ExampleGetFirstWord() {

	a, b := iot2.GetFirstWord("aa bb cc")
	if a != "aa" || b != "bb cc" {
		fmt.Println("oops")
	}
	a, b = iot2.GetFirstWord(b)
	if a != "bb" || b != "cc" {
		fmt.Println("oops2")
	}

	a, b = iot2.GetFirstWord("")
	if a != "" || b != "" {
		fmt.Println("oops3")
	}

	a, b = iot2.GetFirstWord("aaa")
	if a != "aaa" || b != "" {
		fmt.Println("oops4")
	}

	a, b = iot2.GetFirstWord("aaa  bbb  ")
	if a != "aaa" || b != "bbb" {
		fmt.Println("oops5")
	}

	a, b = iot2.GetFirstWord("  aaa      bbb cc dd     ")
	if a != "aaa" || b != "bbb cc dd" {
		fmt.Println("oops6")
	}

	a, b = iot2.GetFirstWord(" \" bbb ccc \"  \" ddd eee \" ")
	if a != "bbb ccc" || b != "\" ddd eee \"" {
		fmt.Println("oops7")
	}

	fmt.Println("done")

	// Output: done

}
