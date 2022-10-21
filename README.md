
![knotfree knot](/KnotFreeKnot256cropped.png)
 # knotfree.net

## What

This is the source code to a publish/subscribe (pubsub) service running at [knotfree.net](https://knotfree.net).

Services include:
- Mqtt 3.1 and Mqtt 5.0 on port 1883
- Mqtt 3.1 and Mqtt 5.0 in port 80 via websockets.
- A text based service on port 7465[^1]
- A binary pubsub service of the knotfreeiot format on port 8384[^2].
- A http forwarding service where http GET messages to http://anychannel.knotfree.net will be published to 'anychannel'
- Service is very cheap. Cheaper than giving out your email address. JWT tokens are used for access and you can get one at [knotfree.net](https://knotfree.net). 

[^1]: See example below.

[^2]: See example in ```monitor_pod/main.go``` and definition in ```packets/packets.go```

## Why

The iot architecture that is being presented involves every message sent having a *return address* to accommodate a *reply*. Mqtt5 now has this feature. To complete the iot architecture demos, and to provide the hobbyist with an service that is undemanding required that one be written. 

## How

#### How to access the unix time server via the web.

Simply visit [http://get-unix-time.knotfree.net/get/time](http://get-unix-time.knotfree.net/get/time) and the return will be the unix time in seconds. 

```get-unix-time``` is an address that is being watched by a simple piece of code. There are more examples written for Arduino over at mqtt5nano. Note that the command ```get time``` is written as ```/get/time``` for the web version. Visit [http://get-unix-time.knotfree.net/help](http://get-unix-time.knotfree.net/help) to see the other commands served by ```get-unix-time```.


#### How to connect using text mode and netcat.

We will create a tcp connection to the text based service at port 7465 using netcat. No drivers required.

In a terminal window enter:
```
nc knotfree.net 7465
```
Then, we will send a connect command with an access token.
```
C token "eyJhbGciOiJFZERTQSIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2OTc2MTQ5MjMsImlzcyI6Il85c2giLCJqdGkiOiJJc2dLamd1RmdfVjNzaVVIYWQ3MFNyVlMiLCJpbiI6MTAyNCwib3V0IjoxMDI0LCJzdSI6MjAsImNvIjoyMCwidXJsIjoia25vdGZyZWUubmV0In0.9N6kW6QK4ZUk9129uzJDnU1jSrX6XTcHthsQZiAFL7nwfzRNNEqOWeZgjKlL7ekcHMF-H0VTHKizXZoR1J1_BA"
```
If the token here is stale or overloaded and you need your own you can get that from http://knotfree.net/.
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
The address ```get-unix-time``` will respond to several commands. Sent it a ```help``` command to see the possibilities.

We can do all of this at the same time by pasting this into a terminal:
```
nc knotfree.net 7465
C token "eyJhbGciOiJFZERTQSIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2OTc2MTQ5MjMsImlzcyI6Il85c2giLCJqdGkiOiJJc2dLamd1RmdfVjNzaVVIYWQ3MFNyVlMiLCJpbiI6MTAyNCwib3V0IjoxMDI0LCJzdSI6MjAsImNvIjoyMCwidXJsIjoia25vdGZyZWUubmV0In0.9N6kW6QK4ZUk9129uzJDnU1jSrX6XTcHthsQZiAFL7nwfzRNNEqOWeZgjKlL7ekcHMF-H0VTHKizXZoR1J1_BA"
S myaddresstopicchannelthing82754459
P get-unix-time myaddresstopicchannelthing82754459 "help"
```
One may also subscribe to the 'testtopic' and get periodic messages. Note that this is *not* our proper request/response format.
```
nc knotfree.net 7465
C token "eyJhbGciOiJFZERTQSIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2OTc2MTQ5MjMsImlzcyI6Il85c2giLCJqdGkiOiJJc2dLamd1RmdfVjNzaVVIYWQ3MFNyVlMiLCJpbiI6MTAyNCwib3V0IjoxMDI0LCJzdSI6MjAsImNvIjoyMCwidXJsIjoia25vdGZyZWUubmV0In0.9N6kW6QK4ZUk9129uzJDnU1jSrX6XTcHthsQZiAFL7nwfzRNNEqOWeZgjKlL7ekcHMF-H0VTHKizXZoR1J1_BA"
S testtopic
```
and then wait for tens of seconds. 

## Where

The server code is here at https://github.com/awootton/knotfreeiot
The C++ version is Arduino clients that are to be found here: https://github.com/awootton/mqtt5nano

## When

2019 to 2022.

## Who 

Copyright 2022 Alan Tracey Wootton. See LICENSE.

#knotfree #iot #mqtt #mqtt5 #go #mqtt5nano

![](https://github.com/awootton/knotfreeiot/workflows/Go/badge.svg)
