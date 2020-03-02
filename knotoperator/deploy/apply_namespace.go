package main

import (
	"sync"

	"github.com/awootton/knotfreeiot/kubectl"
)

func buildTheKnotFreeMain() {
	kubectl.K("cd ../..;docker build -t gcr.io/fair-theater-238820/knotfreeserver .")
	kubectl.K("docker push gcr.io/fair-theater-238820/knotfreeserver")
}

func buildTheOperator() {
	kubectl.K("cd ..;docker build -t gcr.io/fair-theater-238820/app-operatorc ./build/Dockerfile")
	kubectl.K("docker push gcr.io/fair-theater-238820/app-operatorc")
}

// See deploy.sh

// pre-req:
// kind create cluster --config kind-example-config.yaml
// kubectl config use-context "kind-kind"
// kk create ns knotspace
// kubectl config set-context --current --namespace=knotspace

func main() {

	kubectl.K("pwd") // /Users/awootton/Documents/workspace/knotfreeiot/knotoperator/deploy

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		buildTheKnotFreeMain()
	}()

	kubectl.K("kubectl create ns knotspace")
	kubectl.K("kubectl config set-context --current --namespace=knotspace")

	kubectl.K("cd ..;operator-sdk generate k8s")

	kubectl.K("kubectl apply -f service_account.yaml")
	kubectl.K("kubectl apply -f role.yaml")
	kubectl.K("kubectl apply -f role_binding.yaml")
	kubectl.K("kubectl apply -f crds/app.knotfree.io_appservices_crd.yaml")
	//kubectl.K("kubectl apply -f promethius_op.yaml")
	kubectl.K("kubectl apply -f crds/app.knotfree.io_v1alpha1_appservice_cr.yaml")

	wg.Wait()

	kubectl.K("kubectl apply -f knotfreedeploy.yaml")

	// now build and push the deploy operator.yaml

	// cd ~/Documents/workspace/kube-prometheus/
	// # Create the namespace and CRDs, and then wait for them to be availble before creating the remaining resources
	// kubectl create -f manifests/setup
	// until kubectl get servicemonitors --all-namespaces ; do date; sleep 1; echo ""; done
	// kubectl create -f manifests/

}
