// Copyright 2019 Alan Tracey Wootton

package main

import (
	"fmt"
	"io/ioutil"
	"knotfree/kubectl"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// use kubectl to start some pods.
// fully idempotent
// it's really only good for the one k8s deploy yaml file that we use here.
func start(c kubectl.Config) {

	// should print the list of k8s nodes running now.
	kubectl.K("kubectl get no")

	out, err := kubectl.K8s("kubectl create namespace "+c.Namespace, "")
	fmt.Println("kubectl create namespace: ", string(out), err) // ignore error

	// Set as the default namespace in kubectl
	kubectl.K("kubectl config set-context $(kubectl config current-context) --namespace=" + c.Namespace)

	// kubectl.K("cd ..;docker build -t gcr.io/fair-theater-238820/knotfreeserver:v2 .")
	// kubectl.K("docker push gcr.io/fair-theater-238820/knotfreeserver:v2")

	dat, err := ioutil.ReadFile(c.YamlFile)
	check(err)
	deployment := string(dat)
	deployment = strings.ReplaceAll(deployment, "{{NAME}}", c.DeploymentName)
	deployment = strings.ReplaceAll(deployment, "INSERT_CPU_NEEDED_HERE", c.CPU)
	deployment = strings.ReplaceAll(deployment, "INSERT_MEM_NEEDED_HERE", c.Mem)
	deployment = strings.ReplaceAll(deployment, "89345678999236962", strconv.Itoa(c.Replication))
	deployment = strings.ReplaceAll(deployment, "INSERT_COMMAND_HERE", c.Command)

	out, err = kubectl.K8s("kubectl apply -f -", deployment)
	fmt.Println(out)
	if err != nil {
		fmt.Println(deployment)
		check(err)
	}

	fmt.Println("kubectl apply: ", string(out))

	thepodnames := kubectl.GetThePodNames(c.DeploymentName)

	kubectl.K("pwd")

	copyCode := false
	//copy the code
	if copyCode {
		var waitgroup sync.WaitGroup
		for POD := range thepodnames {
			waitgroup.Add(1)
			go func(POD string) {
				kubectl.K("kubectl cp ../../knotfree " + POD + ":/go/src/")
				waitgroup.Done()
			}(POD)
		}
		waitgroup.Wait()
	}

	// start the code.
	startTheProcess := false
	if startTheProcess && false {
		var waitgroup sync.WaitGroup
		for POD := range thepodnames {
			waitgroup.Add(1)
			go func(POD string) {
				cmd := "kubectl exec " + POD + " -- bash -c \"cd src/knotfree && ./" + c.Command + " \""
				kubectl.K(cmd)
				waitgroup.Done()
			}(POD)
		}
		waitgroup.Wait()
	}
}

// eg.
// macbook-pro-atw:deploy awootton$ kk exec knotfreeserver-c58c78df4-zx9rr -- bash -c "ps -ef"
// UID          PID    PPID  C STIME TTY          TIME CMD
// root           1       0  0 06:09 ?        00:00:00 /bin/sh -c touch fffile; tail -f fffile
// root           7       1  0 06:09 ?        00:00:00 tail -f fffile
// root          18       0  0 06:10 ?        00:00:00 /bin/bash -ex ./gorunserver.sh
// root          23      18  0 06:10 ?        00:00:00 go run main.go server
// root          61      23  0 06:10 ?        00:00:00 /tmp/go-build048328676/b001/exe/main server
// root          78       0  0 06:12 ?        00:00:00 ps -ef

// doesn't do anything now
func pkillPods(c kubectl.Config) {

	thepodnames := kubectl.GetThePodNames(c.DeploymentName)
	var waitgroup sync.WaitGroup
	for POD := range thepodnames {
		waitgroup.Add(1)
		go func(POD string) {
			// kubectl exec ${POD} -- bash -c "pkill main" | true
			_, _ = kubectl.K8s("kubectl exec  "+POD+" -- bash -c \"pkill main\"", "")
			waitgroup.Done()
			time.Sleep(1 * time.Minute)
		}(POD)
	}
	waitgroup.Wait()

}

func buildTheDocker() {
	kubectl.K("cd ..;docker build -t gcr.io/fair-theater-238820/knotfreeserver:v2 .")
	kubectl.K("docker push gcr.io/fair-theater-238820/knotfreeserver:v2")
}

var count = 0

func report() {
	for now := range time.Tick(time.Minute) {
		fmt.Println(now, "--------------------------------", count)
		count++
	}
}

func main() {

	go report()

	// TODO: rtfm on flags.

	// assume 20k per socket until we fix it.

	clientconfig := kubectl.Config{
		Namespace:      "knotfree",
		DeploymentName: "client",
		Replication:    20,
		Command:        `client","2500`, //  really 5000 sockets at 20k each is 100 Mi
		Mem:            "200Mi",         // so 10k sockets max 20k per sock
		CPU:            "100m",
		YamlFile:       "knotfreedeploy.yaml",
	}
	serverconfig := kubectl.Config{
		Namespace:      "knotfree",
		DeploymentName: "knotfreeserver",
		Replication:    1,
		Command:        "server", // "gorunserver.sh"
		CPU:            "1500m",
		Mem:            "4000Mi", // 20 * 5000 = 100k socks. 100k * 20kB/sock = 2 Gi
		YamlFile:       "knotfreedeploy.yaml",
	}

	if len(os.Args) > 1 && os.Args[1] == "killmain" {
		pkillPods(clientconfig) // broken
		pkillPods(serverconfig)
		return
	}

	if len(os.Args) > 1 && os.Args[1] == "client" {
		start(clientconfig)
	} else if len(os.Args) > 1 && os.Args[1] == "server" {
		start(serverconfig)
	} else {
		fmt.Println("specify client or server")

		buildTheDocker()

		// pkillPods(serverconfig)
		// pkillPods(clientconfig)

		start(serverconfig)
		start(clientconfig)
		// we can't quit until everyone is done.
		// for {
		// 	time.Sleep(100 * time.Minute)
		// }

	}

}

func check(e error) {
	if e != nil {
		fmt.Println("PANIC because ", e)
		panic(e)
	}
}
