#!/usr/bin/env python3

# pip3 install paho-mqtt

import time
import paho.mqtt.client as paho

# doesn't work: broker = "broker.hivemq.com"
broker = "iot.eclipse.org"
broker = "test.mosquitto.org"

broker = "localhost"

clientid = "client-001"

# define callback

def on_message(client, userdata, message):
    time.sleep(1)
    print("received message =", str(message.payload.decode("utf-8")))

def on_connect(client, userdata, flags, rc):
    if rc==0:
        print("connected OK Returned code=",rc)
        topic = "atw/xsgournklogc/house/bulb1/"+clientid
        print("subscribing " + topic)
        client.subscribe(topic)
    else:
        print("Bad connection Returned code=",rc)

def on_disconnect(client, userdata, flags, rc):
    if rc==0:
        print("disconnect OK Returned code=",rc)
    else:
        print("Bad disconnect Returned code=",rc)

# create client object client1.on_publish = on_publish #assign function to callback client1.connect(broker,port) #establish connection client1.publish("house/bulb1","on")
client = paho.Client(clientid )
# Bind function to callback
client.on_message = on_message
client.on_connect = on_connect
client.on_disconnect = on_disconnect
#####
print("connecting to broker ", broker)
client.connect(broker)  # connect

client.loop_start()  
time.sleep(12)
print("publishing ")
for x in range(9999):
    print(x)
    client.publish("atw/xsgournklogc/house/bulb1/client-001", "on"+clientid+"_"+str(x))
    time.sleep(10)
     
client.disconnect()  # disconnect
client.loop_stop()  # stop loop
