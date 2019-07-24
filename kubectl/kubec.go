// Copyright 2019 Alan Tracey Wootton

package kubectl

import (
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"
)

// Config is for setting up a typical deployment
type Config struct {
	Namespace      string
	DeploymentName string
	Replication    int
	Command        string
	CPU            string
	Mem            string
	YamlFile       string
}

var quiet = false

// I'm doing kubernetes cluster namespace configuration using the
// command line technique. There is also a technique using go-client.

// K8s runs a kubectl command like "kubectl get nodes"
// except it's really just another shell gadget.
// returns what the command outputs and not before it's done.
// and that's annoying because I kinds like watching Docker build.
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

func check(e error) {
	if e != nil {
		fmt.Println("PANIC because ", e)
		panic(e)
	}
}
