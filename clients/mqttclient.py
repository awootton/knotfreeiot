#!/usr/bin/env python3

# pip3 install paho-mqtt

import time
import datetime
import paho.mqtt.client as paho

# These used to work: broker = "broker.hivemq.com"
# broker = "test.mosquitto.org" #[Errno 61] Connection refused
# broker = "iot.eclipse.org" times out

broker = "knotfree.net"

clientid = "client-001"
password = '["My token expires: 2020-12-31",{"iss":"/9sh","in":32,"out":32,"su":4,"co":2,"url":"knotfree.net"},"eyJhbGciOiJFZDI1NTE5IiwidHlwIjoiSldUIn0.eyJleHAiOjE2MDkzNzI4MDAsImlzcyI6Ii85c2giLCJqdGkiOiJqQ0ZqYVNQRGUrUVVwb3NCc0VGK2Uxa2wiLCJpbiI6MzIsIm91dCI6MzIsInN1Ijo0LCJjbyI6MiwidXJsIjoia25vdGZyZWUubmV0In0.LLTrTcFRpngXlOpgte_F6HaLxkXDf5fz17eRMvR5Ymo5lHDb3zoedRklD_dyr1qMIqZ52cOffVj6EqYu8ah8Dg"]'

# define callbacks

def on_message(client, userdata, message):
    print(str(datetime.datetime.now())+"received message =", str(message.payload.decode("utf-8")))

def on_connect(client, userdata, flags, rc):
    if rc==0:
        print("connected OK Returned code=",rc)
        topic = "atw/xsgournklogc/house/bulb1/"+clientid
        print("subscribing " + topic)
        client.subscribe(topic)
    else:
        print("Bad connection Returned code=",rc)

def on_disconnect(client, userdata, rc):
    if rc==0:
        print("disconnect OK Returned code=",rc)
    else:
        print("Bad disconnect Returned code=",rc)

def on_close(client, userdata, flags, rc):
    if rc==0:
        print("close OK Returned code=",rc)
    else:
        print("Bad close Returned code=",rc)


client = paho.Client(clientid )
# Bind function to callback
client.on_message = on_message
client.on_connect = on_connect
client.on_disconnect = on_disconnect
client.on_socket_close = on_close


theToken = password
client.username_pw_set("usernamexxatw", password=theToken)
#####
print("connecting to broker ", broker)
client.connect(broker)  # connect

client.loop_start()  
time.sleep(12)
print("publishing ")
for x in range(9999):
    print(x)
    client.publish("atw/xsgournklogc/house/bulb1/client-001", "msg#"+clientid+"_"+str(x))
    time.sleep(10)
     
client.disconnect()  # disconnect
client.loop_stop()  # stop loop
