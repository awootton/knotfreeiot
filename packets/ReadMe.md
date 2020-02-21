### Herein lies the definition of the objects that will be passed around.

Your basic Connect, Subscribe, Publish kid of objects (or types since this is Go).

They will have fields and virtual methods and will have to marshal and unmarshal. There are many fine library's available for marshalling, or serializing, these objects and in the meantime we're serializing everything as a unique character followed by an array of byte arrays. 

The players are:

* Connect is the first packet received. Will contain a credential

* Disconnect is the last packet received.

* Subscribe contains a string ([]byte really) with a channel name, or topic, or address, or source address or domain name. 

* Unsubscribe reverses what Subscribe did. 

* Lookup will return options set during the Subscribe (like an IPv6 address) and also whether any thing is subscribed to this channel.

* Send sends a message or payload (a byte array) to another channel, or topic, or destination address. 

All the objects have optional byte arrays that can be sent along.

The serialization can be read in the code (packets.go)

There is a particular String() format that I like for debugging this. See packets_test.go

eg: A Send looks like this `[P,dest,,source,,some_data]` which would be better json if the quote marks weren't missing.

There will be more description of these structs later. 

#TODO: finish the code coverage past 93.7%
#TODO: formalize/finalize the marshaling. 

<!-- Global site tag (gtag.js) - Google Analytics -->
<script async src="https://www.googletagmanager.com/gtag/js?id=UA-156005349-2"></script>
<script>
  window.dataLayer = window.dataLayer || [];
  function gtag(){dataLayer.push(arguments);}
  gtag('js', new Date());

  gtag('config', 'UA-156005349-2');
</script>
