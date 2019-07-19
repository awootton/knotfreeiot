#!/bin/bash -ex

for (( i=0; i<4; i++ ))
do

    export N=client$i

    POD=""
    while [ "$POD" == "" ]
    do
        POD=$(kubectl get pods -o name | grep -m1 knotfree${N} | cut -d'/' -f 2) 
    done

    echo "stopping main.go $N"
    kubectl exec ${POD} -- bash -c "pkill main$N" | true

    echo "finished"

done