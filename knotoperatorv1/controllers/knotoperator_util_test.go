package controllers

import (
	"fmt"
	"testing"
)

func TestGetStats(t *testing.T) {

	es, _ := getServerStats("aide-c9b9b5c49-g6shw", "unknown:8080")

	fmt.Println("stats", es)

}
