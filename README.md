# knotfree  
#TheI4T #knotfree #iot #mqtt

![](https://github.com/awootton/knotfreeiot/workflows/Go/badge.svg)

## A 'simple MQTT' service is at knotfree.net:1883 To use it you need a 'password' [and you can get one here](http://knotfree.net/). 

Welcome to the pro bono science project Massively Scalable Distributed Open Source Non Profit IoT Backend with JWT to Control Access and Monetize..

This repository is for the code. [The IoT blog is in another repository](https://thei4t.github.io/).

## Note to Developers
* knotfree is a prototype of how a common iot backend could function.
* Some API's may change in the future except for MQTT. 
* There is a live demonstration running. A 'simple MQTT' service is at knotfree.net:1883 To use it you need a 'password' [and you can get one here](http://knotfree.net/). 

See the mqtt example [here](https://github.com/awootton/knotfreeiot/blob/master/clients/mqttclient.py).

## Running the code

run main.go

The k8s deploy is in knotoperator/deploy/knotfreedeploy.yaml  which is normally deployed with knotoperator/deploy/apply_namespace.go