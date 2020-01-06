# knotfree  
#TheI4T #knotfree #iot #mqtt

![](https://github.com/awootton/knotfreeiot/workflows/Go/badge.svg)

The purpose of this project is to test auto-scaling of a large system. 

To get started with that there is a pub/sub iot server in package iot. MQTT is supported and other formats are proposed.

## The Essay
    draft notes:
## We're going to need a bigger internet for the things. 

The Internet that we all know and love is inadequate for IOT (Internet Of Things) in that 'things' cannot communicate with one another without help and this results in fragmentation into intranets and lack of interoperability between suppliers. One can, and will, predict that this situation won't continue and that soon there will be a new 'internet' for the things (The Internet For Things or I4T ). It's time to discuss the difficulties and requirements for The I4T.

### Social considerations

Since there is essentially one internet it would be, if run by a business, a monopoly. We can agree that people won't stand for that. The I4T will need to be run by something like ICANN. So, there's no profit to be made inventing it. No startup would survive by creating The I4T. 

Large companies are understandably reluctant to rely on any fundamental technology that is not created and controlled by them yet, for reasons outlined above, they can create new intranets or iot standards and they will never be adopted. 

Obviously it's going to need to be open source. The accounting will need to be transparent. 

Some form of logging will be necessary but identifying information should not be collected in any way. 

### Technical considerations.

Re-inventing everything is too much. It will be enough to have a new version of the very lowest layer of the protocol. That means a new version of IP. IP is unreliable and insecure and yet the entire Internet runs atop of it. Functionality similar to DNS will also be needed. 

It will need to scale out to billions of devices. 

It will need to have a hierarchical architecture and local routing should be handled locally then regionally, then nationally then globally. 

### Concrete implementation ideas. 

Since The Internet was created computing power has increased more than 2 orders of magnitude. It would be sensible to allow that we might have an implementation that uses more computing and bandwidth. I4T addresses should be unlimited. 

Premature optimizations should be avoided while we concentrate on the core functionality. Eg. Local gateways will eventually reduce the connection cost and Ipv6 routing will reduce the bandwidth costs.

It can run on top of the existing Internet.

### Cost

It's not free but it's very very inexpensive. I estimate that a thermostat or smart switch can utilize The I4T for $0.001 per month or $0.012 per year. Moreover the bulk of that cost is for maintaining a connection. Whenever a hub or gateway can be used the cost is only for routing and name resolution and is 100 times smaller. Since a non-profit is funneling all fees paid to pay for computing one could imagine that The I4T will be less expensive than any other conceivable option. 

### Conclusions

While my meager contributions here may or may not prove useful it's clear that eventually The I4T will arrive and will have these desirable properties.

### About me

I've written a version here. Your comments and contributions would be greatly appreciated and noted.


