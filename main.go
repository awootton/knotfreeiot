package main

import (
	"os"
	"proj1/knotfree"
)

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
