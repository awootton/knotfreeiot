#!/usr/bin/env python3

# pip3 install paho-mqtt

import time
import datetime
import paho.mqtt.client as paho

# Brokers tested: see http://moxd.io/2015/10/17/public-mqtt-brokers/ which is from 2015

# iot.eclipse.org -- Times out.
# broker.hivemq.com -- Returned code= 1 unacceptable protocol version
# test.mosquitto.org -- Bad connection Returned code= 5 (needs auth)
# test.mosca.io -- Times out.
# broker.mqttdashboard.com -- OK . needs unique client Bad disconnect Returned code= 1
# knotfree.net -- OK
# https://www.cloudmqtt.com/plans.html  out of stock.


broker = "knotfree.net" # 192.168.86.159" 
broker = "localhost" # 192.168.86.159" 

clientid = "clientId-ws131u1ewt"
password = '["My token expires: 2020-12-31",{"iss":"/9sh","in":32,"out":32,"su":4,"co":2,"url":"knotfree.net"},"eyJhbGciOiJFZDI1NTE5IiwidHlwIjoiSldUIn0.eyJleHAiOjE2MDkzNzI4MDAsImlzcyI6Ii85c2giLCJqdGkiOiJqQ0ZqYVNQRGUrUVVwb3NCc0VGK2Uxa2wiLCJpbiI6MzIsIm91dCI6MzIsInN1Ijo0LCJjbyI6MiwidXJsIjoia25vdGZyZWUubmV0In0.LLTrTcFRpngXlOpgte_F6HaLxkXDf5fz17eRMvR5Ymo5lHDb3zoedRklD_dyr1qMIqZ52cOffVj6EqYu8ah8Dg"]'
#password = ''

# define callbacks

def on_message(client, userdata, message):
    print(str(datetime.datetime.now())+"received message =", str(message.payload.decode("utf-8")))
    #print("topic is " + str(message.topic))
    # what is this? is None print(str(userdata))
    user_properties=message.properties.UserProperty
    print("user properties received= ",user_properties)


def on_connect(client, userdata, flags, rc):
    if rc==0:
        print("connected OK Returned code=",rc)
        topic = "atw/xsgournklogc/house/bulb1/client-001" 
        print("subscribing " + topic)
        client.subscribe(topic)
        #client.subscribe(topic,no_local=True)
    else:
        print("Bad connection Returned code=",rc)

# self._userdata,flags_dict, reason, properties
def on_connectV5(client, userdata, flags, rc, properties):
    if rc==0:
        print("connected OK Returned code=",rc)
        topic = "atw/xsgournklogc/house/bulb1/client-001" 
        print("subscribing " + topic)
        client.subscribe(topic)
        #client.subscribe(topic,no_local=True)
    else:
        print("Bad connection Returned code=",rc)

def on_disconnect(client, userdata, rc):
    if rc==0:
        print("disconnect OK Returned code=",rc)
    else:
        print("Bad disconnect Returned code=",rc)

def on_close(client, userdata, flags): ## , rc):
    #if rc==0:
    print("closed " , userdata)
    #else:
    #    print("Bad close Returned code=",rc)

client = paho.Client(clientid )
# Bind function to callback
client.on_message = on_message
client.on_connect = on_connectV5
client.on_disconnect = on_disconnect
client.on_socket_close = on_close
client.protocol = paho.MQTTv5
client._protocol = paho.MQTTv5 # wtf 

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
    topic = "atw/xsgournklogc/house/bulb1/client-001"
    message = "msg#"+clientid+"_"+str(x)
    #props = paho.Properties( packet_type = paho.PUBLISH )# paho.PacketTypes.PUBLISH)
    #props.user_property= ('sfilename', 'test.txt')  # [('sfilename', 'test.txt'),('dfilename', 'test.txt')])
    
    props =[('sfilename', 'test.txt'),('dfilename', 'test.txt')]
    #, user_property = props
    client.publish(topic, message)
    time.sleep(10)
     
client.disconnect()  # disconnect
client.loop_stop()  # stop loop

