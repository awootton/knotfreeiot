#!/bin/bash 

# run this locally to start the cluster

kubectl create ns knotspace | true
kubectl config set-context --current --namespace=knotspace

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


