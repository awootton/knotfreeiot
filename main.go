package main

import (
	"fmt"
	"knotfree/knotfree"
	"os"
	"strconv"
)

// func init() {
// 	subscriptionmgr.Qmessage = &knotfree.Qmessage
// }

func main() {

	if len(os.Args) > 1 && os.Args[1] == "client" {
		fmt.Println("Starting clients = " + strconv.Itoa(200))
		for i := 0; i < 100; i++ {
			istr := strconv.Itoa(i)
			go knotfree.LightSwitch("aaaaaa" + istr)
			go knotfree.LightController("bbbbb"+istr, "aaaaaa"+istr)
		}

	} else {
		knotfree.Server()
	}

	for {
	}

}
