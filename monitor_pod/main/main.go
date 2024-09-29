package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/awootton/knotfreeiot/kubectl"
	"github.com/awootton/knotfreeiot/monitor_pod"
	"github.com/awootton/knotfreeiot/tokens"
)

func main() {

	target_cluster := os.Getenv("TARGET_CLUSTER")
	fmt.Println("target_cluster", target_cluster)

	token := os.Getenv("TOKEN")
	fmt.Println("token", token)

	fmt.Println("version 3")

	argsWithoutProg := os.Args[1:]

	if len(argsWithoutProg) > 0 && argsWithoutProg[0] == "deploy" {

		Deploy(target_cluster)
		return
	}

	{
		c := monitor_pod.ThingContext{}
		c.Topic = "get-unix-time"
		c.CommandMap = make(map[string]monitor_pod.Command)
		c.Index = 0
		c.Token = token
		c.LogMeVerbose = os.Getenv("TARGET_CLUSTER") == "knotfree.com" // aka localhost

		c.Host = os.Getenv("TARGET_CLUSTER") + ":8384" // + ":8384"

		fmt.Println("monitor main c.Host", c.Host)

		monitor_pod.ServeGetTime(token, &c)
	}
	{
		c := monitor_pod.ThingContext{}
		c.Topic = "get-unix-time_iot"
		c.CommandMap = make(map[string]monitor_pod.Command)
		c.Index = 0
		c.Token = token
		c.LogMeVerbose = os.Getenv("TARGET_CLUSTER") == "knotfree.com" // aka localhost

		c.Host = os.Getenv("TARGET_CLUSTER") + ":8384" // + ":8384"

		fmt.Println("monitor main c.Host", c.Host)

		monitor_pod.ServeGetTime(token, &c)
	}
	{
		c := monitor_pod.ThingContext{}
		c.Topic = "a-thermometer-demo_iot"
		c.CommandMap = make(map[string]monitor_pod.Command)
		c.Index = 0
		c.Token = token
		c.LogMeVerbose = os.Getenv("TARGET_CLUSTER") == "knotfree.com" // aka localhost

		c.Host = os.Getenv("TARGET_CLUSTER") + ":8384" // + ":8384"

		fmt.Println("monitor main c.Host", c.Host)

		monitor_pod.ServeGetTime(token, &c)
	}
	{
		c := monitor_pod.ThingContext{}
		c.Topic = "backyard-temp-9gmf97inj5e_xyz"
		c.CommandMap = make(map[string]monitor_pod.Command)
		c.Index = 0
		c.Token = token
		c.LogMeVerbose = os.Getenv("TARGET_CLUSTER") == "knotfree.com" // aka localhost

		c.Host = os.Getenv("TARGET_CLUSTER") + ":8384" // + ":8384"

		fmt.Println("monitor main c.Host", c.Host)

		monitor_pod.ServeGetTime(token, &c)
	}

	// monitor_pod.PublishTestTopic(token)

	for {
		fmt.Println("in monitor_pod")
		time.Sleep(600 * time.Second)
	}
}

func Deploy(TARGET_CLUSTER string) {

	tokens.LoadPrivateKeys("~/atw/privateKeys4.txt")
	TOKEN := tokens.GetImpromptuGiantToken() // 256k connections is GiantX32

	kubectl.K("cd ../../;docker build -f DockerfileMonitor -t  gcr.io/fair-theater-238820/monitor_pod .")
	kubectl.K("cd ../../;docker push gcr.io/fair-theater-238820/monitor_pod")

	data, _ := os.ReadFile("../../monitor_pod/deploy.yaml")
	sdata := strings.ReplaceAll(string(data), "__TARGET_CLUSTER__", TARGET_CLUSTER)
	sdata = strings.ReplaceAll(sdata, "__TOKEN__", TOKEN)
	err := os.WriteFile("dummy.yaml", []byte(sdata), 0644)
	if err != nil {
		fmt.Println("fail flail 888")
	}
	kubectl.K("kubectl apply -f dummy.yaml")
}

// Copyright 2022,2023 Alan Tracey Wootton
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
