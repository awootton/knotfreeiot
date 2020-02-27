
see https://github.com/operator-framework/operator-sdk

needs prom 
https://github.com/coreos/prometheus-operator

https://banzaicloud.com/blog/operator-sdk/

add this lines to .bash_profile 
    export KUBE_EDITOR="atom --wait"
    alias kk='kubectl'

workflow:

kind create cluster --config kind-example-config.yaml
kubectl config use-context "kind-kind" 
kk create ns knotspace
kubectl config set-context --current --namespace=knotspace

# and then don't build the operator yet: 
# operator-sdk build gcr.io/fair-theater-238820/app-operatorc
# docker push gcr.io/fair-theater-238820/app-operatorc

#do this all the time:
operator-sdk generate k8s

kubectl apply -f deploy/service_account.yaml
kubectl apply -f deploy/role.yaml
kubectl apply -f deploy/role_binding.yaml
kubectl apply -f deploy/crds/app.knotfree.io_appservices_crd.yaml
# always goes to default: 
kubectl apply -f  deploy/promethius_op.yaml 
kubectl apply -f deploy/crds/app.knotfree.io_v1alpha1_appservice_cr.yaml
	
# build this:
cd ..
docker build -t gcr.io/fair-theater-238820/knotfreeserver .
docker push gcr.io/fair-theater-238820/knotfreeserver 
cd knotoperator
kubectl apply -f deploy/knotfreedeploy.yaml
	
    
find .Watch( and read that code. 

then  start the debugger with cmd/manager/main.go

operator-sdk generate k8s and go again


kind delete cluster

