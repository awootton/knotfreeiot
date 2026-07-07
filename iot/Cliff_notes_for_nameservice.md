Cliff notes for the commands in lookmsg.go

The easiest one is: 
```
curl "https://knotfree.net/api1/nameService?name=dummmyName&cmd=help"
```

Running a ```get option A @``` command:

Calling it from TestDialTCP_1000_get_option_A

processLookup is where we start. 

    it finds "get option" in the lookupContextGlobal.CommandMap

    does the decrypt.
    fixes and base64 in the args.

    calls comandStruct.Execute
        which is the func in MakeCommand("get option"
    which calls getAndSetWatcher and passes to it 
     a call back.

    If the getAndSetWatcher finds the topic right away then it calls the callback right away. 

    otherwise it calls an non func that does 
    the GetSubscription and 
    sends ```mmm := lookBackCommand``` to the bucket incoming q which does a ```setWatcher``` and then finish()

    The 'finish' is the getAndSetWatcher we see in the command declaration.







