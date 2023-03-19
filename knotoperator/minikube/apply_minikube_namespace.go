package main

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/awootton/knotfreeiot/kubectl"
	"github.com/awootton/knotfreeiot/tokens"
)

// ## build and deploy knotfree using kubectl to current namespace.

func main() {

	kubectl.Quiet = false

	kubectl.K("pwd") // /Users/awootton/Documents/workspace/knotfreeiot/knotoperator/minikube

	registry := ""

	kubectl.K("kubectl create ns knotspace")
	kubectl.K("kubectl config set-context --current --namespace=knotspace")

	if false { // make a new giant token, deploy the monitor_pod

		TARGET_CLUSTER := "localhost" // for monitor pod
		tokens.LoadPrivateKeys("~/atw/privateKeys4.txt")
		TOKEN := tokens.GetImpromptuGiantToken() // 256k connections is GiantX32

		// wtf? hangs kubectl.K("cd ../../monitor_pod;go mod tidy")
		kubectl.K("cd ../../;docker build -f DockerfileMonitor -t  gcr.io/fair-theater-238820/monitor_pod .")
		kubectl.K("cd ../../;docker push gcr.io/fair-theater-238820/monitor_pod")

		data, _ := os.ReadFile("../../monitor_pod/deploy.yaml")
		sdata := strings.ReplaceAll(string(data), "__TARGET_CLUSTER__", TARGET_CLUSTER)
		sdata = strings.ReplaceAll(sdata, "__TOKEN__", TOKEN)
		sdata = strings.ReplaceAll(sdata, "imagePullPolicy: Always", "imagePullPolicy: Never")

		err := os.WriteFile("dummy.yaml", []byte(sdata), 0644)
		if err != nil {
			fmt.Println("fail flail 888")
		}
		kubectl.K("kubectl apply -f dummy.yaml")
	}

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()

		buildTheKnotFreeMain(registry)

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

	hh, _ := os.UserHomeDir()
	dir := hh + "/atw"
	kubectl.K("kubectl create secret generic privatekeys4 --from-file=" + dir)

	//kubectl.K("kubectl apply -f knotfreedeploy.yaml")
	data, _ := os.ReadFile("../deploy/knotfreedeploy.yaml")
	sdata := strings.ReplaceAll(string(data), "gcr.io/fair-theater-238820/", "")
	sdata = strings.ReplaceAll(sdata, "imagePullPolicy: Always", "imagePullPolicy: Never")
	sdata = strings.ReplaceAll(sdata, `["/knotfreeiot/manager"]`, `["/knotfreeiot/manager","-nano"]`)

	fmt.Println(sdata)
	kubectl.K8s("kubectl apply -f -", sdata)

	if !strings.Contains(previousPodNames, "No resources found") {
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

	kubectl.K("kubectl config set-context --current --namespace=knotspace")

	fmt.Println("apply_namespace finished apply_namespace finished apply_namespace finished apply_namespace finished ")
	fmt.Println(time.Now())
	fmt.Println("apply_namespace finished apply_namespace finished apply_namespace finished apply_namespace finished ")
}

func buildTheKnotFreeMain(registry string) {

	if false {
		val, err := kubectl.K8s("pwd", "")
		fmt.Println("buildTheKnotFreeMain in ", val, err)

		// /Users/awootton/Documents/workspace/knotfree-net-homepage/build_to_knotfree_docs.sh

		kubectl.K("ls -lah ../../../knotfree-net-homepage/")

		kubectl.K("rm -rf ../../docs/ ")

		kubectl.K("cd ../../../knotfree-net-homepage/ ; ./build_to_knotfree_docs.sh")
	}

	// kubectl.K("cd ../..;docker build -t " + registry + "knotfreeserver .")

	kubectl.K("cd ../..;minikube image build -t " + "knotfreeserver .")

	// kubectl.K("minikube image load " + registry + "knotfreeserver")

	//kubectl.K("docker push " + registry + "knotfreeserver")

}

// func buildTheOperator(registry string) {
// 	digest, _ := kubectl.K8s("docker inspect --format='{{.RepoDigests}}' "+registry+"/knotoperator", "")
// 	fmt.Println("digest of knotoperator 1", digest)
// 	//kubectl.K("cd ../;ls -lah")
// 	// docker build --file knotoperator/Dockerfile .
// 	kubectl.K("cd ../../;docker build --file knotoperator/Dockerfile -t " + registry + "/knotoperator .")
// 	//kubectl.K("cd ../;docker build -t " + registry + "/knotoperator .")
// 	digest, _ = kubectl.K8s("docker inspect --format='{{.RepoDigests}}' "+registry+"/knotoperator", "")
// 	fmt.Println("digest of knotoperator 2", digest)
// 	kubectl.K("docker push " + registry + "/knotoperator")
// 	digest, _ = kubectl.K8s("docker inspect --format='{{.RepoDigests}}' "+registry+"/knotoperator", "")
// 	fmt.Println("digest of knotoperator 3", digest)
// }
