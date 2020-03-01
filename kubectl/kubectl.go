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

// Package kubectl has some utilty functions that will issue kubectl commands.
// It is assumed that kubectl in your envrionment, typically a workstation, is setup
// and configured to access a running k8s cluster.

package kubectl

import (
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"
)

var quiet = false

// For doing kubernetes cluster namespace configuration using the
// command line technique. There is also a better technique using go-client.

// K8s here runs a kubectl command like "kubectl get nodes"
// except it's really just another shell gadget.
// Returns what the command outputs and not before it's done.
// and that's annoying because I kinda like watching Docker build.
func K8s(command string, input string) (string, error) {
	if !quiet {
		fmt.Println(">" + command)
	}
	cmd := exec.Command("bash", "-c", command)
	if input != "" {
		stdin, err := cmd.StdinPipe()
		if err != nil {
			check(err)
		}
		go func() {
			defer stdin.Close()
			n, err := io.WriteString(stdin, input)
			fmt.Println("io.WriteString> ", n, " ", err)
		}()
	}

	var timer *time.Timer
	timer = time.AfterFunc(120*time.Second, func() {
		timer.Stop()
		cmd.Process.Kill()
	})

	out, err := cmd.CombinedOutput()
	return string(out), err
}

// K - a shorter version of K8s
func K(command string) {
	if !quiet {
		fmt.Println(">" + command)
	}
	out, err := K8s(command, "")
	if err != nil {
		fmt.Println(">ERROR:", err, out)
	}
	if !quiet {
		fmt.Println("", out)
	}
}

// GetThePodNames Waits for the pods to be up and then returns them.
// this smells like crap in bash.
func GetThePodNames(deploymentName string) map[string]bool {

	// remember their names.
	thepodnames := make(map[string]bool)
	count := 1
	for { // wait for them to come up
		quiet = true
		pods, err := K8s("kubectl get po | grep "+deploymentName, "")
		quiet = false
		if err != nil {
			fmt.Println("kubectl get po err ", err)
			time.Sleep(2000 * time.Millisecond)
			count++
			if count > 10 { // 20 sec
				return thepodnames
			}
			continue // back to get po
		}
		// fmt.Println(pods)
		// eg. deploymentName-7428876776-54rws   0/1       Pending   0          10s
		allGood := true
		lines := strings.Split(pods, "\n")
		for _, line := range lines {
			if len(line) < len(deploymentName) {
				continue
			}
			i := strings.Index(line, " ")
			podname := line[0:i]
			podname = strings.Trim(podname, " ")
			// eg deploymentName-7428876776-54rws
			thepodnames[podname] = true
			//fmt.Println(podname)
			tmp := line[len(podname)+1:]
			//fmt.Println(tmp)
			if strings.Contains(tmp, "0/1") {
				allGood = false
			}
			if !strings.Contains(tmp, "Running") {
				allGood = false
			}
		}
		if allGood {
			break
		}
		time.Sleep(2000 * time.Millisecond)
		count++
		if count > 500 { // 1000 sec
			fmt.Println("Pods not all up. Timed out. \n", thepodnames)
			return thepodnames
		}
	}
	// pods all up
	fmt.Println(thepodnames)
	return thepodnames
}

// TODO: needs example

func check(e error) {
	if e != nil {
		fmt.Println("PANIC because ", e)
		panic(e)
	}
}
