
minikube start

edit .kube/config 

minikube image load knotfreeserver
  	minikube image ls --format table

eval $(minikube -p minikube docker-env)

FIXME: the operatorv1 isn't operating. 