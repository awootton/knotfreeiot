#!/bin/bash 

N=9999999999
for (( i=0; i<${N}; i++ ))
do
  kubectl get po
  kubectl top pod
  dt=$(date '+%d/%m/%Y %H:%M:%S');
  echo "$dt -------------$i "
  sleep 30
done
