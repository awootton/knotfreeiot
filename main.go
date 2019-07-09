package main

import (
	"knotfree/knotfree"
	"os"
)

// func init() {
// 	subscriptionmgr.Qmessage = &knotfree.Qmessage
// }

func main() {

	if len(os.Args) > 1 && os.Args[1] == "client" {
		go knotfree.LightSwitch("aaaaaa")
		go knotfree.LightController("bbbbb", "aaaaaa")
	} else {
		knotfree.Server()
	}

	for {
	}

}
