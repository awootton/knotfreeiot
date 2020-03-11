package main

import (
	"fmt"
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
	//kubectl.K("cd ..;operator-sdk build gcr.io/fair-theater-238820/knotoperator")
	//kubectl.K("docker push gcr.io/fair-theater-238820/knotoperator")
}

// See deploy.sh

// pre-req:
// kind create cluster --config kind-example-config.yaml
// kubectl config use-context "kind-kind"
// kubectl create ns knotspace
// kubectl config set-context --current --namespace=knotspace
// or else kubectl goes to google

// cd workspace/kube-prometheus
// kubectl create -f manifests/setup
// until kubectl get servicemonitors --all-namespaces ; do date; sleep 1; echo ""; done
// kubectl apply -f manifests/

// it's much faster when we don't build the docker every time.
var needtobuild = true

func main() {

	kubectl.K("pwd") // /Users/awootton/Documents/workspace/knotfreeiot/knotoperator/deploy

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

	kubectl.K("kubectl create ns knotspace")
	kubectl.K("kubectl config set-context --current --namespace=knotspace")

	kubectl.K("kubectl apply -f service_account.yaml")
	kubectl.K("kubectl apply -f role.yaml")
	kubectl.K("kubectl apply -f role_binding.yaml")
	kubectl.K("kubectl apply -f crds/app.knotfree.io_appservices_crd.yaml")
	kubectl.K("kubectl apply -f crds/app.knotfree.io_v1alpha1_appservice_cr.yaml")

	wg.Wait()

	kubectl.K("kubectl apply -f knotfreedeploy.yaml")

	kubectl.K("kubectl apply -f operator.yaml")

	fmt.Println(time.Now())

}
