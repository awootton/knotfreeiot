
![knotfree knot](/KnotFreeKnot256cropped.png)
 # knotfree.net

See example projects: https://wootton.substack.com/

## What

This is the source code to a publish/subscribe (pubsub) service running at [knotfree.net](https://knotfree.net).

Services include:
- Mqtt 3.1 and Mqtt 5.0 on port 1883
- Mqtt 3.1 and Mqtt 5.0 in port 80 via websockets.
- A text based service on port 7465[^1]
- A binary pubsub service of the knotfreeiot format on port 8384[^2].
- A http forwarding service where http GET messages to http://anychannel.knotfree.net will be published to 'anychannel'
- Service is very cheap. Cheaper than giving out your email address. JWT tokens are used for access and you can get one at [knotfree.net](https://knotfree.net). 
- There are 'attrtibutes' on the Topics aka Subscriptions supporting DNS queries. See [the DNS server project here](https://github.com/awootton/coredns).

[^1]: See example below.

[^2]: See example in ```monitor_pod/main.go``` and definition in ```packets/packets.go```

There is a frontend running at knotfree.net and the source code for that is at https://github.com/awootton/knotfree-net-homepage
There is C++ for Arduino which is the embedded code for microcontrollers that completes an IOT stack. This is at https://github.com/awootton/mqtt5nano

Doncumentation is in the knotfreeiot wiki here: https://github.com/awootton/knotfreeiot/wiki

## Why

The iot architecture that is being presented involves every message sent having a *return address* to accommodate a *reply*. Mqtt5 now has this feature. To complete the iot architecture demos, and to provide the hobbyist with an service that is undemanding required that one be written. 

## How

See the wiki for [howto's](https://github.com/awootton/knotfreeiot/wiki/How-to-use-the-publish-subscribe)

## Where

The server code is here at https://github.com/awootton/knotfreeiot
The C++ version is Arduino clients that are to be found here: https://github.com/awootton/mqtt5nano

## When

2019 to 2024.

## Who 

Copyright 2024 Alan Tracey Wootton. See LICENSE.

#knotfree #iot #mqtt #mqtt5 #go #mqtt5nano

![](https://github.com/awootton/knotfreeiot/workflows/Go/badge.svg)
