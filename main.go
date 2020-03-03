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

package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/awootton/knotfreeiot/iot"
	"github.com/awootton/knotfreeiot/tokens"
)

// Hint: add "127.0.0.1 knotfreeserver" to /etc/hosts
func main() {

	tokens.LoadPublicKeys()

	fmt.Println("Hello knotfreeserver")

	client := flag.Int("client", 0, "start a client test with an int of clients.")
	server := flag.Bool("server", false, "start a server.")
	//isGuru := flag.Bool("isguru", false, "")

	// means that the limits are very small
	nano := flag.Bool("nano", false, "")

	token := flag.String("token", "", " an access token for our superiors")

	flag.Parse()

	if *token == "" {
		*token = tokens.SampleSmallToken
	}

	var mainLimits = &iot.ExecutiveLimits{}
	mainLimits.Connections = 10 * 1000
	mainLimits.BytesPerSec = 10 * 1000
	mainLimits.Subscriptions = 1000 * 1000

	limits := mainLimits

	name := os.Getenv("POD_NAME")
	if len(name) == 0 {
		name = "apodnamefixme"
	}

	if *nano == true {
		limits = &iot.TestLimits
		fmt.Println("nano limits")
	}

	if *server {

		// aide1.httpAddress = ":8080"
		// aide1.tcpAddress = ":8384"
		// aide1.textAddress = ":7465"
		// aide1.mqttAddress = ":1883"

		iot.MakeTCPMain(name, limits, *token)
		for {
			time.Sleep(1000 * time.Second)
		}
	}
	if *client > 0 {

		// FIXME: put the stress tests back in here.

	} else {
		iot.MakeTCPMain(name, limits, *token)
		for {
			time.Sleep(1000 * time.Second)
		}
	}

}
