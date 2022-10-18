## What

## Why

## How

#### How to connect using text mode and netcat.

We will create a tcp connection to the text based service at port 7465 using netcat.
```
nc knotfree.net 7465
```
We will send a connect command with an access token.
```
C token "eyJhbGciOiJFZERTQSIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2OTc2MTQ5MjMsImlzcyI6Il85c2giLCJqdGkiOiJJc2dLamd1RmdfVjNzaVVIYWQ3MFNyVlMiLCJpbiI6MTAyNCwib3V0IjoxMDI0LCJzdSI6MjAsImNvIjoyMCwidXJsIjoia25vdGZyZWUubmV0In0.9N6kW6QK4ZUk9129uzJDnU1jSrX6XTcHthsQZiAFL7nwfzRNNEqOWeZgjKlL7ekcHMF-H0VTHKizXZoR1J1_BA"
```
If the token is stale or overloaded and you need your own you can get that from http://knotfree.net/.
Then we will make up a random, and hopefully unique, address for ourselves for this session. We will 'subscribe' to it.
```
S myaddresstopicchannelthing82754459
```
Then we can send the command "get time" to the address "get-unix-time". Note that we send our address as the return address.
```
P get-unix-time myaddresstopicchannelthing82754459 "get time"
```
The reply is a P (for publish) in an array of strings and the destination address and the source address have been changed into a hashed format. It looks like this:
```
[P,=AAFKYuRRQIKzSiS2MzHbg65zVm05snxI,=xOZPbNiNsA_lM_6xJEwM1C7YmVMGlDpA,1666078249]
```
Where **1666078249** is the current unix time in seconds. 
The address ```get-unix-time``` will respond to several commands. Sent it a ```help``` command to see the posibilities.

We can do all of this at the same time by pasting this into a terminal:
```
nc newpeoples.net 7465
C token "eyJhbGciOiJFZERTQSIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2OTc2MTQ5MjMsImlzcyI6Il85c2giLCJqdGkiOiJJc2dLamd1RmdfVjNzaVVIYWQ3MFNyVlMiLCJpbiI6MTAyNCwib3V0IjoxMDI0LCJzdSI6MjAsImNvIjoyMCwidXJsIjoia25vdGZyZWUubmV0In0.9N6kW6QK4ZUk9129uzJDnU1jSrX6XTcHthsQZiAFL7nwfzRNNEqOWeZgjKlL7ekcHMF-H0VTHKizXZoR1J1_BA"
S myaddresstopicchannelthing82754459
P get-unix-time myaddresstopicchannelthing82754459 "help"
```
One may also subscribe to the 'testtopic' and get periodic messages. Note that this is *not* our proper request/response format.
```
nc newpeoples.net 7465
C token "eyJhbGciOiJFZERTQSIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2OTc2MTQ5MjMsImlzcyI6Il85c2giLCJqdGkiOiJJc2dLamd1RmdfVjNzaVVIYWQ3MFNyVlMiLCJpbiI6MTAyNCwib3V0IjoxMDI0LCJzdSI6MjAsImNvIjoyMCwidXJsIjoia25vdGZyZWUubmV0In0.9N6kW6QK4ZUk9129uzJDnU1jSrX6XTcHthsQZiAFL7nwfzRNNEqOWeZgjKlL7ekcHMF-H0VTHKizXZoR1J1_BA"
S testtopic
```
and then wait for 10 seconds. 

## Where

The server code is here at https://github.com/awootton/knotfreeiot
The C++ versios is Arduino clients that are to be found here: https://github.com/awootton/knotfreeiot

## When

2020 to 2022.

## Who 

Copyright 2022 Alan Tracey Wootton. See LICENSE.

 


















# knotfree  
#TheI4T #knotfree #iot #mqtt #mqtt5 #NewPeoplesInternet

![](https://github.com/awootton/knotfreeiot/workflows/Go/badge.svg)

## A 'simple MQTT' service is at knotfree.net:1883 To use it you need a 'password' [and you can get one here](http://knotfree.net/). 

Knotfree is a pub/sub server. It will also accept MQTT5 and MQTT3.1 connections by TCP or websocket.

This repository is for the backend code.

## Note to Developers
* knotfree is a prototype of how a common distributed routing network could function. Think IP and DNS only better and cheaper.
* Some API's may change in the future except for MQTT. 
* There is a live demonstration running. A 'simple MQTT' service is at knotfree.net:1883 To use it you need a 'password' [and you can get one here](http://knotfree.net/). 

See the mqtt example [here](https://github.com/awootton/knotfreeiot/blob/master/clients/mqttclient.py).

## Running the code

run main.go

The k8s deploy is in knotoperator/deploy/knotfreedeploy.yaml  which is normally deployed by running knotoperator/deploy/apply_namespace.go with kubectl configured to access a k8s cluster of your choice.
