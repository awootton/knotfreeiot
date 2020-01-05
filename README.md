# knotfree, # TheI4T

![](https://github.com/awootton/knotfreeiot/workflows/Go/badge.svg)

The purpose of this project is to test auto-scaling of a large system. 

To get started with that there is a pub/sub iot server in package iot. MQTT is supported and other formats are proposed.

##The Essay

##We're going to need a bigger internet. 

The Internet that we all know and love is inadequate for IOT (Internet Of Things) in that 'things' cannot communicate with one another without help and this results in fragmentation into intranets and lack of interoperability between suppliers. One can, and will, predict that this situation won't continue and that soon there will be a new 'internet' for the things (The Internet For Things or I4T ). It's time to discuss the difficulties and requirements for The I4T.

### Social considerations

Since there is essentially one internet it would be, if run by a business, a monopoly. We can probably agree that prople won't stand for that. The I4T will need to be run by something like ICANN so there's no profit to be made inventing it. No startup would survive by creating The I4T. 

Large companies are understandably reluctant to rely on any fundamental technology that is not created and controled by them yet, for reasons outlined above, they can create new intranets or iot standards and they will never be adopted. 

Obviously it's going to need to be open source.

### Technical considerations.

Re-inventing everything is too much. It will be enough to have a new version of the lowest layer of the protocol. That means a new version of IP. IP is unreliable and insecure and yet the entire Internet run atop of it. Functionality similiar to DNS will also be needed. 

It will need to scale out to billions of devices. 

It will need to have a heirarchical archicture and local routing should be handled locally then regionally, then nationally then globally. 

### Concrete implementation ideas. 

Since The Internet was created computing power has increased more than 2 orders of magnitude. It would be sensible to allow that we might have an implekmentation that uses more computing and bandwidth. I4T addresses should be unlimited. 

### Cost

It's not free but it's very very inexpensive. I estimate that a thertmostat or smart switch can utilze The I4T for $0.001 per month or $0.012 per year. Moreover the bulk of that cost is for maintaining a connection. Whenever a hub or gateway can be used the cost for routing falls by 100. Since a non-profit is funneling all feees paid to pay for computing one could imagine that The I4T will be less expensive than any other concievable option. 

### About me

I'm building it here as a volunteer. Your contributions would be greatly appreciated and noted.


