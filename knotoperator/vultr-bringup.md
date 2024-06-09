
Start cluster at vultr

setup ~/.kube/config

run apply_namespace.go

check: 
kubectl get po
get:
aide-97455668c-stzzb
because there's no operator to monitor and add pods. 

in .../workspace/knotoperatorv1 do the ```make docker``` and ```make deploy``` commands.

note that the operator won't run unless knotfree.net resolves to the new address # was 212.2.241.55 

kubectl get po 
returns: 
aide-97455668c-stzzb                                 1/1     Running   0          61m
knotoperatorv1-controller-manager-74bf5cf595-8l85n   2/2     Running   0          2m11s

now we have a load balancer  

Manage Load Balancer (a90d2293e3552480bbd7a681120cbc0d)
Dallas |216.128.128.195 |2001:19f0:6402:2:ffff:ffff:ffff:ffff |46d3d919-afb5-4df0-b748-a2c93be66224

add 216.128.128.195 to knotfreeiot startPublicServer. Run apply_namespace.go again to redeploy

// knotoperatorv1 not working. Needs http I think

See: https://docs.vultr.com/how-to-install-a-wildcard-let-s-encrypt-ssl-certificate-on-vultr-kubernetes-engine

    setup the vultr DNS. change the cname * wildcard to an A wildcard
    change the ns at squarespace.
    make 01-dns.yaml
    save it in the root dir (not deploy/)
    add the VULTR_API_KEY 

    kubectl apply -f 01-dns.yaml

    Install the nginx-ingress controller.

    # kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v1.5.1/deploy/static/provider/cloud/deploy.yaml
    Verify the installation.

    kubectl get services/ingress-nginx-controller -n ingress-nginx
        note that we are waiting for an external IP -->  is 155.138.240.174 

    Install the cert-manager plugin.

    kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.10.1/cert-manager.yaml
    Verify the installation.

    kubectl get pods --namespace cert-manager

    Clone the GitHub repository. In knotoperatorv1

    git clone https://github.com/vultr/cert-manager-webhook-vultr.git
    Create a secret resource. key is vultr api key. see .kube

    ### no kubectl create secret generic "vultr-credentials" --from-literal=apiKey=EXX...OML3A --namespace=cert-manager

    kk apply -f 0-vutlr-secret.yaml

    ## to test key: export VULTR_API_KEY=EXXQ...ML3A


    Install the webhook. 

    cd cert-manager-webhook-vultr
    helm install --namespace cert-manager cert-manager-webhook-vultr ./deploy/cert-manager-webhook-vultr

    make 2-webhook.yaml
    kubectl apply -f 2-webhook.yaml
    Verify the installation.

    kubectl get clusterissuer

    create 3-certificate.yaml

    kubectl apply -f 3-certificate.yaml

    make 4-nginx.yaml

    kubectl apply -f 4-nginx.yaml

    make 5-ingress.yaml

    kubectl apply -f 5-ingress.yaml

    check kubectl describe challenges --all-namespaces
    for invalid api token

    TODO: fix all ipv4 in api token  permissions

    #### rebuild the operator

    mkdir knotoperatorv2
    cd knotoperatorv2
    operator-sdk init --domain knotfree.net --repo github.com/awootton/knotoperatorv2

    brew unlink go@1.22.2
    brew link @go@1.21

    operator-sdk create api --group cache --version v2alpha2 --kind Knotfree --resource --controller

    see notes in operatorv2
  

defer:
    push to git. 