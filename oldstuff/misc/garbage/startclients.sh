#!/bin/bash -ex

# The currect directory should be src/knotfreeiot/deploy

#kubectl create namespace clients | true
#kubectl config set-context --current --namespace=clients

#export N=xx

export CPU=20m
export MEM=64Mi

export CPU=100m
export MEM=700Mi


for (( i=0; i<4; i++ ))
do
   export N=client$i

    ./template.sh server.yaml | kubectl apply -f -

    POD=""
    while [ "$POD" == "" ]
    do
        POD=$(kubectl get pods -o name | grep -m1 knotfree${N} | cut -d'/' -f 2) 
    done



    #kubectl exec ${POD} -- bash -c "go get -u github.com/eclipse/paho.mqtt.golang"

    #  kubectl exec -it ${POD} -- bash 

    kubectl exec ${POD} -- bash -c "pkill main" | true

    echo "Copy source...$N"
    kubectl cp ../../knotfree ${POD}:/go/src/

 #   echo "start $N"
 #   kubectl exec ${POD} -- bash -c "cd src/knotfree && go run main.go client "

done

for (( i=0; i<4; i++ ))
do
    export N=client$i
    POD=""
    while [ "$POD" == "" ]
    do
        POD=$(kubectl get pods -o name | grep -m1 knotfree${N} | cut -d'/' -f 2) 
    done

    echo "start $N"
    kubectl exec ${POD} -- bash -c "cd src/knotfree && go run main.go client " &

done

# for (( i=0; i<4; i++ ))
# do

#     export N=client$i

#     echo "stopping main.go $N"
#     #kubectl exec ${POD} -- bash -c "pkill main$N" | true

#     echo "finished"

# done
