package appservice

import (
	"fmt"
	"testing"
)

func TestGetStats(t *testing.T) {

	es, _ := GetServerStats("aide-c9b9b5c49-g6shw", "unknown:8080")

	fmt.Println("stats", es)

}
