


civo k8s 

Server Version: version.Info{Major:"1", Minor:"22", GitVersion:"v1.22.11+k3s1", GitCommit:"bb0cdd929a960aade81e51911f98fdee44ebce4e", GitTreeState:"clean", BuildDate:"2022-06-27T22:14:54Z", GoVersion:"go1.16.10", Compiler:"gc", Platform:"linux/amd64"}

makig new v1 version:


// new operator version
operator-sdk init --domain knotfree.net --repo github.com/awootton/knotfreeiot/knotoperatorv1

operator-sdk create api --group cache --version v1alpha1 --kind Knotoperator --resource --controller

editing... see https://sdk.operatorframework.io/docs/building-operators/golang/tutorial/

make manifests ??

