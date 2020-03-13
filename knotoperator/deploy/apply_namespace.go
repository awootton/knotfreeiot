package main

import (
	"fmt"
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

// it's much faster when we don't build the docker every time.
var needtobuild = true

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

	// do thi slast
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

	kubectl.K("kubectl apply -f knotfreedeploy.yaml")

	kubectl.K("kubectl apply -f operator.yaml")

	// do libra now in the other project.

	kubectl.K("cd ../my-kube-prometheus;kubectl create -f manifests/setup")
	kubectl.K(`until kubectl get servicemonitors --all-namespaces ; do date; sleep 1; echo ""; done`)
	kubectl.K("cd ../my-kube-prometheus;kubectl apply -f manifests/")

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

	fmt.Println(time.Now())

}
