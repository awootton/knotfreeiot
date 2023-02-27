#!/usr/bin/env python3

# pip3 install paho-mqtt
# is (1.5.1)

import time
import datetime
import paho.mqtt.client as paho
from paho.mqtt.properties import Properties
from paho.mqtt.packettypes import PacketTypes

# broker = "192.168.86.31"
broker = "knotfree.io"

# broker = "broker.mqttdashboard.com"

clientid = "clientId-ws131u1ewt"

# aka token
password = 'eyJhbGciOiJFZERTQSIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2NzkxNzA1ODcsImlzcyI6Il85c2giLCJqdGkiOiJhdDZhZTN4emVxZmhmNm51YWlqOGZ6Z3QiLCJpbiI6MTUyLCJvdXQiOjE1Miwic3UiOjEwMCwiY28iOjQsInVybCI6Imtub3RmcmVlLmNvbTo4MDg1In0.ncQsiwTWoBrB2_l_p2VfSX7gQ1WsAty9r5qJF1EFvBw7LMxpDFuqiuj8ElvMP8oNePa_GAJ0ZXgqLdtQ6Qz6AQ'

lastTime = time.time() * 1000

def on_message(client, userdata, message):
    global lastTime
    rawMessageLength = len(message.payload)
    theMessageStr = str(message.payload.decode("utf-8"))
    print(str(datetime.datetime.now())+"on_message received message =", theMessageStr)
    print("topic is " + str(message.topic))
    responseTopic = ""
    if hasattr(message,'properties')  :
        if hasattr(message.properties,'UserProperty') :
            user_properties=message.properties.UserProperty
            print("user properties received= ",user_properties)
        if hasattr(message.properties,'ResponseTopic') :
            responseTopic = message.properties.ResponseTopic
            print("ResponseTopic= ", message.properties.ResponseTopic)
        if hasattr(message.properties,'CorrelationData'):
            cccc = message.properties.CorrelationData
            print("corr data= ",cccc)
    now = time.time() * 1000
    delta = now - lastTime
    print("latency= ",int(delta))
    if theMessageStr.startswith("GET "):
        if "HTTP/1.1\n" in theMessageStr :
            print ( "rawMessageLength ", rawMessageLength)
            topic = responseTopic
            cleaned = theMessageStr[4:theMessageStr.find("HTTP/1.1\n")]
            cleaned = cleaned.replace("/"," ")
            message = "GET command received: " + cleaned
            properties = Properties(PacketTypes.PUBLISH)
            properties.ResponseTopic = "xxxx"
            #client.publish(topic, message, qos=0, retain=False, properties=properties)


def on_connect(client, userdata, flags, rc):
    if rc==0:
        print("connected OK Returned code=",rc)
        topic = "alice_vociferous_mcgrath" 
        print("subscribing " + topic)
        client.subscribe(topic)
        #client.subscribe(topic,no_local=True)
    else:
        print("Bad connection Returned code=",rc)

# self._userdata,flags_dict, reason, properties
def on_connectV5(client, userdata, flags, rc, properties):
    if rc==0:
        print("v5 connected OK Returned code=",rc)
        topic = "testtopic" # "alice_vociferous_mcgrath" # "atw/xsgournklogc/house/bulb1/client-001" 
        print("subscribing " + topic)
        # properties.UserProperty=[("xxdebg","xx12345678","key9","val9")]
        client.subscribe(topic)# ,properties=properties)

        ##topic = "PyClientReturnAddr" 
        ##print("subscribing " + topic)
        ##client.subscribe(topic)
        ##client.subscribe(topic,no_local=True)
    else:
        print("Bad connection Returned code=",rc)

def on_subscribe(client, userdata, mid, granted_qos, properties):
    print("v5 on_subscribe")
    print(userdata)
    print(properties)

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
client.on_subscribe = on_subscribe 

theToken = password
client.username_pw_set("usernamexxatw", password=theToken)
#####
print("connecting to broker ", broker)
client.connect(broker)  # connect

client.loop_start()  
time.sleep(2)
print("publishing ")
for count in range(9999999):
    print(count)
    topic = "alice_vociferous_mcgrath" 
    topic = "testtopic" 
    message = "query__"+str(count)

    properties = Properties(PacketTypes.PUBLISH)
    properties.UserProperty=[("xxdebg","xx12345678","key9","val9")]
    properties.ResponseTopic = "PyClientReturnAddr"
    cdatabytes = bytearray(str(count), 'utf-8')
    properties.CorrelationData = cdatabytes #bytes([count]) 
    lastTime = time.time() * 1000
    #### client.publish(topic, message, qos=0, retain=False, properties=properties)
    time.sleep(10)
     
client.disconnect()  # disconnect
client.loop_stop()  # stop loop

