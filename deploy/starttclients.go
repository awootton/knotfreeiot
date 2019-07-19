package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

// I'm doing kubernetes cluster namespace configuration using the
// command line technique. There is also a technique using go-client.

func check(e error) {
	if e != nil {
		fmt.Println("PANIC because ", e)
		panic(e)
	}
}

// K8s runs a kubectl command like "kubectl get nodes"
func K8s(command string, input string) (string, error) {
	fmt.Println(">" + command)
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

	out, err := cmd.CombinedOutput()
	return string(out), err
}

// K - a shorter version of K8s
func K(command string) {
	fmt.Println(">" + command)
	out, err := K8s(command, "")
	if err != nil {
		fmt.Println(">ERROR:", err, out)
	}
	fmt.Println(">", out)
}

//type DeployClients int

type config struct {
	myNamespace    string //= "knotfree"
	deploymentName string
	repl           int
	command        string
	cpu            string
	mem            string
}

func getThePodNames(c config) map[string]bool {
	// wait for them to come up
	// remember their names.
	thepodnames := make(map[string]bool)
	for {
		pods, err := K8s("kubectl get po | grep "+c.deploymentName, "")
		if err != nil {
			fmt.Println("kubectl get po err ", err)
			time.Sleep(100 * time.Millisecond)
			continue // back to get po
		}
		fmt.Println(pods)
		// eg. deploymentName-7428876776-54rws   0/1       Pending   0          10s
		allGood := true
		lines := strings.Split(pods, "\n")
		for _, line := range lines {
			if len(line) < len(c.deploymentName) {
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
	}
	// pods all up
	fmt.Println(thepodnames)
	return thepodnames
}

// ExampleDeployClients
func startk8s(c config) {

	K("kubectl get no")

	out, err := K8s("kubectl create namespace "+c.myNamespace, "")
	fmt.Println("kubectl create namespace: ", string(out), err)

	// Set as the default namespace in kubectl
	K("kubectl config set-context $(kubectl config current-context) --namespace=" + c.myNamespace)

	dat, err := ioutil.ReadFile("knotfreedeploy.yaml")
	check(err)
	deployment := string(dat)
	deployment = strings.ReplaceAll(deployment, "{{NAME}}", c.deploymentName)
	deployment = strings.ReplaceAll(deployment, "INSERT_CPU_NEEDED_HERE", c.cpu)
	deployment = strings.ReplaceAll(deployment, "INSERT_MEM_NEEDED_HERE", c.mem)
	deployment = strings.ReplaceAll(deployment, "89345678999236962", strconv.Itoa(c.repl))

	out, err = K8s("kubectl apply -f -", deployment)
	fmt.Println(out)
	if err != nil {
		fmt.Println(deployment)
		check(err)
	}

	fmt.Println("kubectl apply: ", string(out))

	thepodnames := getThePodNames(c)

	//copy the code
	var waitgroup sync.WaitGroup
	for POD := range thepodnames {
		waitgroup.Add(1)
		go func(POD string) {
			K("kubectl cp ../../knotfree " + POD + ":/go/src/")
			waitgroup.Done()
		}(POD)
	}
	waitgroup.Wait()

	// start the code.
	for POD := range thepodnames {
		waitgroup.Add(1)
		go func(POD string) {
			K("kubectl exec " + POD + " -- bash -c \"cd src/knotfree && ./" + c.command + " \"")
			fmt.Println("run command returned")
			waitgroup.Done()
		}(POD)
	}
	waitgroup.Wait()
}

// macbook-pro-atw:deploy awootton$ kk exec knotfreeserver-c58c78df4-zx9rr -- bash -c "ps -ef"
// \UID          PID    PPID  C STIME TTY          TIME CMD
// root           1       0  0 06:09 ?        00:00:00 /bin/sh -c touch fffile; tail -f fffile
// root           7       1  0 06:09 ?        00:00:00 tail -f fffile
// root          18       0  0 06:10 ?        00:00:00 /bin/bash -ex ./gorunserver.sh
// root          23      18  0 06:10 ?        00:00:00 go run main.go server
// root          61      23  0 06:10 ?        00:00:00 /tmp/go-build048328676/b001/exe/main server
// root          78       0  0 06:12 ?        00:00:00 ps -ef

func pkillPods(c config) {

	thepodnames := getThePodNames(c)
	var waitgroup sync.WaitGroup
	for POD := range thepodnames {
		waitgroup.Add(1)
		go func(POD string) {
			// kubectl exec ${POD} -- bash -c "pkill main" | true
			_, _ = K8s("kubectl exec  "+POD+" -- bash -c \"pkill main\"", "")
			waitgroup.Done()
		}(POD)
	}
	waitgroup.Wait()

}

func main() {

	if len(os.Args) > 1 && os.Args[1] == "killmain" {
		// do a pkill on all the pods
		return
	}

	clientconfig := config{
		myNamespace:    "knotfree",
		deploymentName: "client",
		repl:           8,
		command:        "gorunclient.sh",
		cpu:            "100m",
		mem:            "700Mi",
	}
	serverconfig := config{
		myNamespace:    "knotfree",
		deploymentName: "knotfreeserver",
		repl:           1,
		command:        "gorunserver.sh",
		cpu:            "400m",
		mem:            "2048Mi",
	}

	if len(os.Args) > 1 && os.Args[1] == "client" {
		startk8s(clientconfig)
	} else if len(os.Args) > 1 && os.Args[1] == "server" {
		startk8s(serverconfig)
	} else {
		fmt.Println("specify client or server")

		//pkillPods(clientconfig)
		//pkillPods(serverconfig)

		startk8s(clientconfig)
		//startk8s(serverconfig)

	}

}
