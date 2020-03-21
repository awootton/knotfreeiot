package main

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/awootton/knotfreeiot/kubectl"
)

func buildTheKnotFreeMain() {
	kubectl.K("cd ../../docs;bundle exec jekyll build")
	kubectl.K("cd ../..;docker build -t gcr.io/fair-theater-238820/knotfreeserver .")
	kubectl.K("docker push gcr.io/fair-theater-238820/knotfreeserver")
}

func buildTheOperator() {
	kubectl.K("cd ..;operator-sdk build gcr.io/fair-theater-238820/knotoperator")
	kubectl.K("docker push gcr.io/fair-theater-238820/knotoperator")
}

// build and deploy knotfree using kubectl.
// See deploy.sh which is how I used to do it. todo: make better

// pre-req:
// kind create cluster --config kind-example-config.yaml
// kubectl config use-context "kind-kind"
// kubectl create ns knotspace
// kubectl config set-context --current --namespace=knotspace
// or else kubectl goes to google

// TODO: have config and args
// it's much faster when we don't build the docker every time.
var needtobuild = true

var alsoDoLibra = false // might be deprecating libra due to excessive disk usage.

var alsoStartMonitoring = false // might be deprecating libra due to excessive disk usage.

func main() {

	kubectl.K("pwd") // /Users/awootton/Documents/workspace/knotfreeiot/knotoperator/deploy
	kubectl.K("kubectl get no")

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		if needtobuild {
			buildTheKnotFreeMain()
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if needtobuild {
			buildTheOperator()
		}
	}()

	// do this last
	// kubectl.K("cd ../my-kube-prometheus;kubectl create -f manifests/setup")
	// kubectl.K(`until kubectl get servicemonitors --all-namespaces ; do date; sleep 1; echo ""; done`)
	// kubectl.K("cd ../my-kube-prometheus;kubectl apply -f manifests/")

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
	path := hh + "/atw/privateKeys4.txt"
	kubectl.K("kubectl create secret generic privatekeys4 --from-file=" + path)

	kubectl.K("kubectl apply -f knotfreedeploy.yaml")

	kubectl.K("kubectl apply -f operator.yaml")

	if alsoStartMonitoring {

		kubectl.K("cd ../my-kube-prometheus;kubectl create -f manifests/setup")
		kubectl.K(`until kubectl get servicemonitors --all-namespaces ; do date; sleep 1; echo ""; done`)
		kubectl.K("cd ../my-kube-prometheus;kubectl apply -f manifests/")
	}
	if needtobuild {
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
