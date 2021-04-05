package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/awootton/knotfreeiot/kubectl"
)

// ## build and deploy knotfree using kubectl to current namespace.
// See deploy.sh which is how I used to do it. todo: make better
// using the go client is more professional.

// ## Start kind THIS WAY from the knotoperator dir: kind create cluster --config kind-example-config.yaml

// pre-req:
// kind create cluster --config kind-example-config.yaml
// kubectl config use-context "kind-kind"
// kubectl create ns knotspace
// kubectl config set-context --current --namespace=knotspace
// Be SURE that kubectl is pointing at the right ns. !!!!

// TODO: have config and args
// it's much faster when we don't build the docker every time.
var needtobuild = true
var startTheOperator = true

var alsoDoLibra = false         // are deprecating libra due to excessive disk usage.
var alsoStartMonitoring = false // once is enough

func main() {

	isKind := false

	kubectl.Quiet = false

	kubectl.K("pwd") // /Users/awootton/Documents/workspace/knotfreeiot/knotoperator/deploy
	nodes, err := kubectl.K8s("kubectl get no", "")
	if err != nil {
		fmt.Println("err quitting", err)
	}
	if strings.Contains(nodes, "kind-control-plane") {
		isKind = true
	}

	registry := "gcr.io/fair-theater-238820"
	if isKind {
		registry = "localhost:5000"
	}

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		if needtobuild {
			buildTheKnotFreeMain(registry)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if needtobuild && startTheOperator {
			buildTheOperator(registry)
		}
	}()

	kubectl.K("kubectl create ns knotspace")
	kubectl.K("kubectl config set-context --current --namespace=knotspace")

	kubectl.K("kubectl apply -f service_account.yaml")
	kubectl.K("kubectl apply -f role.yaml")
	kubectl.K("kubectl apply -f role_binding.yaml")
	kubectl.K("kubectl apply -f crds/app.knotfree.io_appservices_crd.yaml")
	kubectl.K("kubectl apply -f crds/app.knotfree.io_v1alpha1_appservice_cr.yaml")

	wg.Wait()

	deploymentName := "aide-"
	previousPodNames, err := kubectl.K8s("kubectl get po | grep "+deploymentName, "")
	_ = err

	hh, _ := os.UserHomeDir()
	//path2 := hh + "/atw/fair-theater-238820-firebase-adminsdk-uyr4z-63b4da8ff3.json"
	//path1 := hh + "/atw/privateKeys4.txt"
	dir := hh + "/atw"
	kubectl.K("kubectl create secret generic privatekeys4 --from-file=" + dir)
	//kubectl.K("kubectl create secret generic privatekeys4 --from-file=" + path1 + " --from-file=" + path2)

	//kubectl.K("kubectl apply -f knotfreedeploy.yaml")
	data, _ := ioutil.ReadFile("knotfreedeploy.yaml")
	sdata := strings.ReplaceAll(string(data), "gcr.io/fair-theater-238820", registry)
	kubectl.K8s("kubectl apply -f -", sdata)

	//kubectl.K("kubectl apply -f operator.yaml")
	data, _ = ioutil.ReadFile("operator.yaml")
	sdata = strings.ReplaceAll(string(data), "gcr.io/fair-theater-238820", registry)
	if startTheOperator {
		kubectl.K8s("kubectl apply -f -", sdata)
	}

	if alsoStartMonitoring {

		kubectl.K("cd ../my-kube-prometheus;kubectl create -f manifests/setup")
		kubectl.K(`until kubectl get servicemonitors --all-namespaces ; do date; sleep 1; echo ""; done`)
		kubectl.K("cd ../my-kube-prometheus;kubectl apply -f manifests/")

		kubectl.K("kubectl apply -f knotfreemonitoring.yaml")
	}
	if needtobuild && strings.Contains(previousPodNames, "No resources found") == false {
		// delete the aides
		lines := strings.Split(previousPodNames, "\n")
		for _, line := range lines {
			if len(line) < len(deploymentName) {
				continue
			}
			i := strings.Index(line, " ")
			podname := line[0:i]
			podname = strings.Trim(podname, " ")
			// eg aide-7428876776-54rws
			kubectl.K("kubectl delete po " + podname)
		}
	}

	if alsoDoLibra {
		ldir := "/Users/awootton/Documents/workspace/libra-statefulset"
		kubectl.K("cd " + ldir + "; go test -run TestApply")
	}
	kubectl.K("kubectl config set-context --current --namespace=knotspace")

	fmt.Println(time.Now())

}

func buildTheKnotFreeMain(registry string) {
	kubectl.K("cd ../../docs;bundle exec jekyll build")
	digest, _ := kubectl.K8s("docker inspect --format='{{.RepoDigests}}' "+registry+"/knotfreeserver", "")
	fmt.Println("digest of knotfreeserver 1", digest)
	kubectl.K("cd ../..;docker build -t " + registry + "/knotfreeserver .")
	digest, _ = kubectl.K8s("docker inspect --format='{{.RepoDigests}}' "+registry+"/knotfreeserver", "")
	fmt.Println("digest of knotoperator 2", digest)
	kubectl.K("docker push " + registry + "/knotfreeserver")
	digest, _ = kubectl.K8s("docker inspect --format='{{.RepoDigests}}' "+registry+"/knotfreeserver", "")
	fmt.Println("digest of knotfreeserver 3", digest)
}

func buildTheOperator(registry string) {
	digest, _ := kubectl.K8s("docker inspect --format='{{.RepoDigests}}' "+registry+"/knotoperator", "")
	fmt.Println("digest of knotoperator 1", digest)
	kubectl.K("cd ..;operator-sdk build " + registry + "/knotoperator")
	digest, _ = kubectl.K8s("docker inspect --format='{{.RepoDigests}}' "+registry+"/knotoperator", "")
	fmt.Println("digest of knotoperator 2", digest)
	kubectl.K("docker push " + registry + "/knotoperator")
	digest, _ = kubectl.K8s("docker inspect --format='{{.RepoDigests}}' "+registry+"/knotoperator", "")
	fmt.Println("digest of knotoperator 3", digest)
}
