// Copyright 2019 Alan Tracey Wootton

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"knotfreeiot/kubectl"
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
	Type           string // eg type NodePort
}

// use kubectl to start some pods.
// fully idempotent
// it's really only good for the one k8s deploy yaml file that we use here.
func start(c Config) {

	// should print the list of k8s nodes running now.
	kubectl.K("kubectl get no")

	out, err := kubectl.K8s("kubectl create namespace "+c.Namespace, "")
	fmt.Println("kubectl create namespace: ", string(out), err) // ignore error

	// Set as the default namespace in kubectl
	kubectl.K("kubectl config set-context $(kubectl config current-context) --namespace=" + c.Namespace)

	// kubectl.K("cd ..;docker build -t gcr.io/fair-theater-238820/knotfreeserver:v2 .")
	// kubectl.K("docker push gcr.io/fair-theater-238820/knotfreeserver:v2")

	// Behold: the home-rolled template engine.
	dat, err := ioutil.ReadFile(c.YamlFile)
	check(err)
	deployment := string(dat)
	deployment = strings.ReplaceAll(deployment, "{{NAME}}", c.DeploymentName)
	deployment = strings.ReplaceAll(deployment, "INSERT_CPU_NEEDED_HERE", c.CPU)
	deployment = strings.ReplaceAll(deployment, "INSERT_MEM_NEEDED_HERE", c.Mem)
	deployment = strings.ReplaceAll(deployment, "89345678999236962", strconv.Itoa(c.Replication))
	deployment = strings.ReplaceAll(deployment, "INSERT_COMMAND_HERE", c.Command)
	deployment = strings.ReplaceAll(deployment, "#type: {{SERVICE_TYPE}}", c.Type)

	// ioutil.WriteFile("dummy.yaml", []byte(deployment), 0644)

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

func pkillPods(c Config) {

	kubectl.K("kubectl delete deploy " + c.DeploymentName)

	// thepodnames := kubectl.GetThePodNames(c.DeploymentName)
	// var waitgroup sync.WaitGroup
	// for POD := range thepodnames {
	// 	waitgroup.Add(1)
	// 	go func(POD string) {
	// 		// kubectl exec ${POD} -- bash -c "pkill main" | true
	// 		_, _ = kubectl.K8s("kubectl exec  "+POD+" -- bash -c \"pkill main\"", "")
	// 		waitgroup.Done()
	// 		time.Sleep(1 * time.Minute)
	// 	}(POD)
	// }
	// waitgroup.Wait()

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

	clientconfig := Config{
		Namespace:      "knotfree",
		DeploymentName: "client",
		Replication:    20,
		Command:        `-client=2500`, //  really 5000 sockets at 20k each is 100 Mi
		Mem:            "800Mi",        // so 10k sockets max 20k per sock
		CPU:            "250m",
		YamlFile:       "knotfreedeploy.yaml",
		Type:           "type: NodePort",
	}
	serverconfig := Config{
		Namespace:      "knotfree",
		DeploymentName: "knotfreeserver",
		Replication:    1,
		Command:        `-server`,
		CPU:            "1500m",
		Mem:            "4000Mi", // 20 * 5000 = 100k socks. 100k * 20kB/sock = 2 Gi
		YamlFile:       "knotfreedeploy.yaml",
		Type:           "type: LoadBalancer",
	}

	// ["-client=10","-server","-str"","-aa"]

	combinedconfig := Config{
		Namespace:      "knotfree2",
		DeploymentName: "knotfreeserver",
		Replication:    2,
		Command:        `-server","-client=6000","-str","-aa`,
		CPU:            "250m",
		Mem:            "800Mi", // 20 * 5000 = 100k socks. 100k * 20kB/sock = 2 Gi
		YamlFile:       "knotfreedeploy.yaml",
		Type:           "type: NodePort",
	}

	// do flags.

	if len(os.Args) > 1 && os.Args[1] == "killmain" {
		pkillPods(clientconfig) // broken
		pkillPods(serverconfig)
		return
	}

	if len(os.Args) > 1 && os.Args[1] == "combined" {

		pkillPods(combinedconfig)

		buildTheDocker()

		start(combinedconfig)

	} else if len(os.Args) > 1 && os.Args[1] == "client" {
		start(clientconfig)
	} else if len(os.Args) > 1 && os.Args[1] == "server" {
		start(serverconfig)
	} else {
		fmt.Println("specify client or server or we'll do both")

		buildTheDocker()

		pkillPods(serverconfig)
		pkillPods(clientconfig)

		start(serverconfig)
		start(clientconfig)

	}

}

func check(e error) {
	if e != nil {
		fmt.Println("PANIC because ", e)
		panic(e)
	}
}
