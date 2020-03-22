#!/bin/bash 

# run this locally to start the cluster

#!/bin/sh
set -o errexit

# desired cluster name; default is "kind"
KIND_CLUSTER_NAME="${KIND_CLUSTER_NAME:-kind}"

# create registry container unless it already exists
reg_name='kind-registry'
reg_port='5000'
running="$(docker inspect -f '{{.State.Running}}' "${reg_name}" 2>/dev/null || true)"
if [ "${running}" != 'true' ]; then
  docker run \
    -d --restart=always -p "${reg_port}:5000" --name "${reg_name}" \
    registry:2
fi
reg_ip="$(docker inspect -f '{{.NetworkSettings.IPAddress}}' "${reg_name}")"

# create a cluster with the local registry enabled in containerd
cat <<EOF | kind create cluster --name "${KIND_CLUSTER_NAME}" --config=-
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
containerdConfigPatches: 
- |-
  [plugins."io.containerd.grpc.v1.cri".registry.mirrors."localhost:${reg_port}"]
    endpoint = ["http://${reg_ip}:5000"]
EOF


kubectl create ns knotspace | true
kubectl config set-context --current --namespace=knotspace

#do this all the time:
operator-sdk generate k8s

kubectl apply -f deploy/service_account.yaml
kubectl apply -f deploy/role.yaml
kubectl apply -f deploy/role_binding.yaml
kubectl apply -f deploy/crds/app.knotfree.io_appservices_crd.yaml
# always goes to default: 
#kubectl apply -f  deploy/promethius_op.yaml 
kubectl apply -f deploy/crds/app.knotfree.io_v1alpha1_appservice_cr.yaml
	
# build this:
cd ..
docker build -t gcr.io/fair-theater-238820/knotfreeserver .
docker push gcr.io/fair-theater-238820/knotfreeserver 
cd knotoperator

kubectl apply -f deploy/knotfreedeploy.yaml


