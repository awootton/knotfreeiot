package appservice

import (
	"fmt"
	"testing"
)

func TestGetStats(t *testing.T) {

	es := GetServerStats("aide-59794f445c-24hsk", "unknown:8080")

	fmt.Println("stats", es)

}
