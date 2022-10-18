#!/usr/bin/env python3

# pip3 install paho-mqtt
# is (1.5.1)

import time
import datetime
import paho.mqtt.client as paho
from paho.mqtt.properties import Properties
from paho.mqtt.packettypes import PacketTypes

# Brokers tested: see http://moxd.io/2015/10/17/public-mqtt-brokers/ which is from 2015

# iot.eclipse.org -- Times out.
# broker.hivemq.com -- Returned code= 1 unacceptable protocol version
# test.mosquitto.org -- Bad connection Returned code= 5 (needs auth)
# test.mosca.io -- Times out.
# broker.mqttdashboard.com -- OK . needs unique client Bad disconnect Returned code= 1
# knotfree.net -- OK
# https://www.cloudmqtt.com/plans.html  out of stock.

broker = "knotlocal.com" # "knotfree.net" # 192.168.86.159" knotlocal.com is localhost in my hosts
# broker = "knotfree2.com" #  aka localhost in my /etc/hosts file

clientid = "clientId-ws131u1ewt"
password = '[Free_token_expires:_2021-12-31,{exp:1641023999,iss:_9sh,jti:HpifIJkhgnTOGc3EDmOJaV0A,in:32,out:32,su:4,co:2,url:knotfree.net},eyJhbGciOiJFZDI1NTE5IiwidHlwIjoiSldUIn0.eyJleHAiOjE2NDEwMjM5OTksImlzcyI6Il85c2giLCJqdGkiOiJIcGlmSUpraGduVE9HYzNFRG1PSmFWMEEiLCJpbiI6MzIsIm91dCI6MzIsInN1Ijo0LCJjbyI6MiwidXJsIjoia25vdGZyZWUubmV0In0.YSo2Ur7lbkwTPZfQymyvy4N1mWQaUn_cziwK36kTKlASgqOReHQ4FAocVvgq7ogbPWB1hD4hNoJtCg2WWq-BCg]'

password = 'eyJhbGciOiJFZDI1NTE5IiwidHlwIjoiSldUIn0.eyJleHAiOjE2OTMzMTIzNTQsImlzcyI6Il85c2giLCJqdGkiOiJ4TkR4SV9sbkZ1NV9aakNITFU4MXlvTTkiLCJpbiI6MTAyNCwib3V0IjoxMDI0LCJzdSI6MjAsImNvIjoyMCwidXJsIjoia25vdGZyZWUubmV0In0.Utcx5e2ve-U8nVrjmYipB45X5QsGDwlufczfq34i8tEA-QO659ox3IsXHu9qLeIukiAFc3cKGtRjeJqCww40Aw'

lastTime = time.time() * 1000

# define callbacks

# 2022-09-06 02:28:54.868949on_message received message = GET /get/wifi/password HTTP/1.1
# Upgrade-Insecure-Requests: 1
# User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/105.0.0.0 Safari/537.36
# Accept: text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9
# Accept-Encoding: gzip, deflate
# Accept-Language: en-US,en;q=0.9
# Connection: keep-alive
# Cache-Control: max-age=0


# topic is =xHKLHv0yN_784LG-smK64uX6y2M8NlFd
# return addr=  =6mqUoJPl2dvk32NDZPqlUNBX9IWKKH1s
# latency=  1258

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
        client.subscribe(topic)
        topic = "PyClientReturnAddr" 
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
time.sleep(2)
print("publishing ")
for count in range(9999):
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
    client.publish(topic, message, qos=0, retain=False, properties=properties)
    time.sleep(10)
     
client.disconnect()  # disconnect
client.loop_stop()  # stop loop

