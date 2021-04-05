# knotfree  
#TheI4T #knotfree #iot #mqtt #mqtt5 #NewPeoplesInternet

![](https://github.com/awootton/knotfreeiot/workflows/Go/badge.svg)

## A 'simple MQTT' service is at knotfree.net:1883 To use it you need a 'password' [and you can get one here](http://knotfree.net/). 

Welcome to the pro bono science project: Massively Scalable Distributed Open Source Non Profit Routing Network with JWT to Control Access..

This repository is for the backend code.

## Note to Developers
* knotfree is a prototype of how a common distributed routing network could function. Think IP and DNS only better and cheaper.
* Some API's may change in the future except for MQTT. 
* There is a live demonstration running. A 'simple MQTT' service is at knotfree.net:1883 To use it you need a 'password' [and you can get one here](http://knotfree.net/). 

See the mqtt example [here](https://github.com/awootton/knotfreeiot/blob/master/clients/mqttclient.py).

## Running the code

run main.go

The k8s deploy is in knotoperator/deploy/knotfreedeploy.yaml  which is normally deployed by running knotoperator/deploy/apply_namespace.go with kubectl configured to access a k8s cluster of your choice.