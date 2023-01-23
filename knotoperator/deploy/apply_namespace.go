package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/awootton/knotfreeiot/kubectl"
	"github.com/awootton/knotfreeiot/tokens"
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
var startTheOperator = false // this is now ../knotoperatorv1/make deploy

var alsoDoLibra = false        // are deprecating libra due to excessive disk usage.
var alsoStartMonitoring = true // once is enough //this is broken from being too old

var buildReactAndCopy = true // todo: mount the react static files instead of baking them in the docker.
// TODO: better. redirect to s3 bucket with the files in it.? does this work?

var TARGET_CLUSTER = "knotfree.io" // for monitor pod

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
		registry = "localhost:5000" // btw. isKind is broken
	}

	kubectl.K("kubectl create ns knotspace")
	kubectl.K("kubectl config set-context --current --namespace=knotspace")

	{
		tokens.LoadPrivateKeys("~/atw/privateKeys4.txt")
		TOKEN := tokens.GetImpromptuGiantToken() // 256k connections is GiantX32

		// wtf? hangs kubectl.K("cd ../../monitor_pod;go mod tidy")
		kubectl.K("cd ../../monitor_pod;docker build -t  gcr.io/fair-theater-238820/monitor_pod .")
		kubectl.K("cd ../../monitor_pod;docker push gcr.io/fair-theater-238820/monitor_pod")

		data, _ := ioutil.ReadFile("../../monitor_pod/deploy.yaml")
		sdata := strings.ReplaceAll(string(data), "__TARGET_CLUSTER__", TARGET_CLUSTER)
		sdata = strings.ReplaceAll(sdata, "__TOKEN__", TOKEN)
		err := ioutil.WriteFile("dummy.yaml", []byte(sdata), 0644)
		if err != nil {
			fmt.Println("fail flail 888")
		}
		kubectl.K("kubectl apply -f dummy.yaml")
	}

	var wg sync.WaitGroup

	// wg.Add(1)
	// go func() {
	// 	defer wg.Done()
	// 	if needtobuild && startTheOperator {
	// 		buildTheOperator(registry)
	// 	}
	//}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if needtobuild {
			buildTheKnotFreeMain(registry)
		}
	}()

	wg.Wait()

	deploymentName := "aide-"
	previousPodNames, err := kubectl.K8s("kubectl get po | grep "+deploymentName, "")
	_ = err

	deploymentName2 := "guru-"
	previousPodNames2, err := kubectl.K8s("kubectl get po | grep "+deploymentName2, "")
	_ = err

	deploymentName3 := "monitor-"
	previousPodNames3, err := kubectl.K8s("kubectl get po | grep "+deploymentName3, "")
	_ = err

	// deploymentName3 := "knotoperator-"
	// previousPodNames3, err := kubectl.K8s("kubectl get po | grep "+deploymentName3, "")
	// _ = err

	// previousPodNames += previousPodNames2
	// previousPodNames += previousPodNames3

	hh, _ := os.UserHomeDir()
	//path2 := hh + "/atw/fair-theater-238820-firebase-adminsdk-uyr4z-63b4da8ff3.json"
	//path1 := hh + "/atw/privateKeys4.txt"
	dir := hh + "/atw"
	kubectl.K("kubectl create secret generic privatekeys4 --from-file=" + dir)

	//kubectl.K("kubectl apply -f knotfreedeploy.yaml")
	data, _ := ioutil.ReadFile("knotfreedeploy.yaml")
	sdata := strings.ReplaceAll(string(data), "gcr.io/fair-theater-238820", registry)
	kubectl.K8s("kubectl apply -f -", sdata)

	//kubectl.K("kubectl apply -f operator.yaml")
	// data, _ = ioutil.ReadFile("operator.yaml")
	// sdata = strings.ReplaceAll(string(data), "gcr.io/fair-theater-238820", registry)
	// if startTheOperator {
	// 	kubectl.K8s("kubectl apply -f -", sdata)
	// }

	if alsoStartMonitoring { //this is broken from being too old

		// kubectl.K("cd ../my-kube-prometheus;kubectl create -f manifests/setup")
		// kubectl.K(`until kubectl get servicemonitors --all-namespaces ; do date; sleep 1; echo ""; done`)
		// kubectl.K("cd ../my-kube-prometheus;kubectl apply -f manifests/")

		// kubectl.K("kubectl apply -f knotfreemonitoring.yaml")
	}
	if needtobuild && strings.Contains(previousPodNames, "No resources found") == false {
		// delete the aides, and others
		{
			lines := strings.Split(previousPodNames2, "\n")
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
		{
			lines := strings.Split(previousPodNames3, "\n")
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

		// if false {
		// 	lines := strings.Split(previousPodNames3, "\n")
		// 	for _, line := range lines {
		// 		if len(line) < len(deploymentName) {
		// 			continue
		// 		}
		// 		i := strings.Index(line, " ")
		// 		podname := line[0:i]
		// 		podname = strings.Trim(podname, " ")
		// 		// eg aide-7428876776-54rws
		// 		kubectl.K("kubectl delete po " + podname)
		// 	}
		// }
		//time.Sleep(time.Second * 20)
		{ // these are the aides. do them last
			// so it's less likeely it will beroken.
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
		// sometimes the aides thrash and can't contact the guru.
		// TODO: check for that and fix it.
	}

	if alsoDoLibra {
		ldir := "/Users/awootton/Documents/workspace/libra-statefulset"
		kubectl.K("cd " + ldir + "; go test -run TestApply")
	}
	kubectl.K("kubectl config set-context --current --namespace=knotspace")

	fmt.Println("apply_namespace finished apply_namespace finished apply_namespace finished apply_namespace finished ")
	fmt.Println(time.Now())
	fmt.Println("apply_namespace finished apply_namespace finished apply_namespace finished apply_namespace finished ")
}

func buildTheKnotFreeMain(registry string) {
	//kubectl.K("cd ../../docs;bundle exec jekyll build")

	if buildReactAndCopy {
		val, err := kubectl.K8s("pwd", "")
		fmt.Println("buildTheKnotFreeMain in ", val, err)

		// /Users/awootton/Documents/workspace/knotfree-net-homepage/build_to_knotfree_docs.sh

		kubectl.K("ls -lah ../../../knotfree-net-homepage/")

		kubectl.K("rm -rf ../../docs/ ")

		kubectl.K("cd ../../../knotfree-net-homepage/ ; ./build_to_knotfree_docs.sh")
	}

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
	//kubectl.K("cd ../;ls -lah")
	// docker build --file knotoperator/Dockerfile .
	kubectl.K("cd ../../;docker build --file knotoperator/Dockerfile -t " + registry + "/knotoperator .")
	//kubectl.K("cd ../;docker build -t " + registry + "/knotoperator .")
	digest, _ = kubectl.K8s("docker inspect --format='{{.RepoDigests}}' "+registry+"/knotoperator", "")
	fmt.Println("digest of knotoperator 2", digest)
	kubectl.K("docker push " + registry + "/knotoperator")
	digest, _ = kubectl.K8s("docker inspect --format='{{.RepoDigests}}' "+registry+"/knotoperator", "")
	fmt.Println("digest of knotoperator 3", digest)
}

func XXXbuildTheMonitor(registry string) {

	kubectl.K("cd ../../;docker build -t  gcr.io/fair-theater-238820/monitor_pod .")
	kubectl.K("cd ../../;docker push gcr.io/fair-theater-238820/monitor_pod")
	kubectl.K("cd ../../;docker push gcr.io/fair-theater-238820/monitor_pod")

}
