#!/usr/bin/env python3

# pip3 install paho-mqtt

import os
import io
import time
import datetime
import hashlib
import base64

import paho.mqtt.client as paho

import select
import socket
import threading

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


broker = "knotfree.net"  # 192.168.86.159"
#broker = "localhost" # 192.168.86.159"

clientid = "clientId-ws131u1ewt"
password = '[My_token_expires:_2021-12-31,{exp:1641023999,iss:_9sh,jti:amXYKIuS4uykvPem9Fml371o,in:32,out:32,su:4,co:2,url:knotfree.net},eyJhbGciOiJFZDI1NTE5IiwidHlwIjoiSldUIn0.eyJleHAiOjE2NDEwMjM5OTksImlzcyI6Il85c2giLCJqdGkiOiJhbVhZS0l1UzR1eWt2UGVtOUZtbDM3MW8iLCJpbiI6MzIsIm91dCI6MzIsInN1Ijo0LCJjbyI6MiwidXJsIjoia25vdGZyZWUubmV0In0.7ElPyX1Vju8Q5uDOEfgAblwvE2gxT78Jl68JPlqLRcFeMJ7if39Ppl2_Jr_JTky371bIXAn6S-pghtWSqTBwAQ]'
password = 'giant token  eyJhbGciOiJFZDI1NTE5IiwidHlwIjoiSldUIn0.eyJleHAiOjE2NDkzNTQ1NTAsImlzcyI6Il85c2giLCJqdGkiOiJZRkVpWmxlUlNsendXbk9mb0NUMGZheVgiLCJpbiI6MTAwMDAwMCwib3V0IjoxMDAwMDAwLCJzdSI6MjAwMDAwLCJjbyI6MjAwMDAwLCJ1cmwiOiJrbm90ZnJlZS5uZXQifQ.fRquac1VWm7pgAkCad98jbMyM31tiGgAKwG2uny93CiV4wNEW0CYXE_yONJyuFgfEAomZ4DunoSDiGKqJWQrCA'
targetPort = 4000
targetPort = 3000

TheMapping = {
    # topics and ports
    "alan222" : "3000",
    "dummy": "4000",
    "awootton2": "8899",
}

def fixMapping():
    addme = {}
    for key in TheMapping:
        val = TheMapping.get(key)
        if key[0] == '=' :
            continue
        key2bin = hashlib.sha256(key.encode('utf-8')).digest()#.hexdigest()
        key2 = base64.b64encode(key2bin)
        key2 = "=" + key2.decode()[0:32]
        key2 = key2.replace("/","_")
        key2 = key2.replace("+","-")
        addme[key2] = val
    TheMapping.update(addme)

fixMapping()

def socket2send(client, userdata, message):

    responseTopic = str(message.properties.ResponseTopic)
    #print(message.topic)
    port = targetPort  
    k = str(message.topic)
    portStr = TheMapping.get(k)
    if portStr :
        port = int(portStr)

    if 1==1:
        ppp = str(message.payload.decode("utf-8"))
        firstLineEnd = ppp.find("\n")
        firstLine = ppp[0:firstLineEnd]
        print("top sending payload to socket ", firstLine)

    host = "localhost"
    # The same port as used by the web server
    socket.setdefaulttimeout(10.0)
    conn = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    conn.connect((host, port))

    payloadSlice = message.payload
    #conn.sendall(message.payload)
    packetCount = 0

    # loop here until we ... forever
    startTime = time.time()
    running = True
    while running:
        now = time.time()
        # if now - startTime > 35:
        #     # timed  out after 35 sec. is this used? 
        #     print("timed  out after SOME sec")
        #     break
        conn.settimeout(10.0)
        collectedData = bytearray(b'')
        while True:
            now = time.time()
            if now - startTime > 30:
                # timed  out after 30 sec. is this used? 
                print("timed  out after SOME sec",firstLine)
                running = False
                break
            
            writeSelectList = [conn,]
            if len(payloadSlice) == 0:
                writeSelectList = []
            try:
                ready_to_read, ready_to_write, in_error = select.select([conn,], writeSelectList, [conn,], 5)
            except select.error as e:
                conn.shutdown(2)    # 0 = done receiving, 1 = done sending, 2 = both
                conn.close()
                # connection error event here, maybe reconnect
                print('connection error', firstLine, e)
                running = False
                break
            except select.timeout as e:
                conn.shutdown(2)    # 0 = done receiving, 1 = done sending, 2 = both
                conn.close()
                # connection error event here, maybe reconnect
                print('connection timeout', firstLine, e)
                running = False
                break
            if len(in_error) != 0 or conn.fileno()==-1 :
                print('connection exceptional condition', firstLine, in_error)
                running = False
                break
            if len(ready_to_read) > 0:
                data = conn.recv(62 * 1024)
                # do stuff with received data
                if len(data) == 0:
                    continue
                startTime = now
                #collectedData.extend(data)
                collectedData = data
                print('Received len ', len(collectedData), "with", firstLine)
                break
                #if len(collectedData) > (17 * 1024):
                #    break
                #else keep collecting
            if len(ready_to_write) > 0:
                # connection established, send some stuff
                if len(payloadSlice) > 0:
                    amountsent = conn.send(payloadSlice)
                    payloadSlice  = payloadSlice[amountsent:]

            # try:
            #     data = s.recv(16 * 1024)  # try to receive
            # except socket.error as se:
            #     # usually timed out 
            #     print("got socket.error ", firstLine, se)
            #     break
            # except OSError as err:  # we don't get this one so much
            #     print("Didn't receive data!", firstLine, err)
            #     break
            # if len(data) == 0:  # why does this happen??
            #     #is the socket open still?  
            #     print("empty data! ", firstLine)
            #     #break # lets say this is the end. fall through and close
            #     # maybe someone sent an empty string?
            #     continue
            #print('Received len ', len(data))

        if running == False:
            break
        
        end = 32
        if end > len(collectedData):
            end = len(collectedData)
        
        properties=Properties(PacketTypes.PUBLISH)
        count="1"
        properties.UserProperty=[("indx",str(packetCount))]
        
        print("sending publish for",firstLine, " len(collectedData)=" , len(collectedData))
        client.publish(responseTopic, collectedData,   qos=0, retain=False, properties=properties)
        packetCount = packetCount + 1

    ##  This doesn't work. It never comes here. 
    print("sending final packet for", firstLine , " of ", packetCount)
    properties=Properties(PacketTypes.PUBLISH)
    properties.UserProperty=[ ("of",str(packetCount))]    
    client.publish(responseTopic, "no-data",   qos=0, retain=False, properties=properties)
    conn.close()


def on_message(client, userdata, message):

    #payload = str(message.payload.decode("utf-8"))
    #print(str(datetime.datetime.now())+"received message =", payload)

    thread = threading.Thread(target=socket2send, args=(client, userdata, message,))
    thread.start()


def on_connect(client, userdata, flags, rc):
    if rc == 0:
        print("connected OK Returned code=", rc)
        topic = "atw/xsgournklogc/house/bulb1/client-001"
        print("subscribing " + topic)
        client.subscribe(topic)
        # client.subscribe(topic,no_local=True)
    else:
        print("Bad connection Returned code=", rc)

# self._userdata,flags_dict, reason, properties


def on_connectV5(client, userdata, flags, rc, properties):
    if rc == 0:
        print("connected OK Returned code=", rc)
        done = {}
        for key in  TheMapping:
            topic = key
            print("subscribing " + topic)
            if done.get(topic) == None:
                client.subscribe(topic)
            done[topic] = "1"

        # client.subscribe(topic,no_local=True)
    else:
        print("Bad connection Returned code=", rc)


def on_disconnect(client, userdata, rc):
    if rc == 0:
        print("disconnect OK Returned code=", rc)
    else:
        print("Bad disconnect Returned code=", rc)


def on_close(client, userdata, flags):  # , rc):
    # if rc==0:
    print("closed ", userdata)
    # else:
    #    print("Bad close Returned code=",rc)


client = paho.Client(clientid)
# Bind function to callback
client.on_message = on_message
client.on_connect = on_connectV5
client.on_disconnect = on_disconnect
client.on_socket_close = on_close
client.protocol = paho.MQTTv5
client._protocol = paho.MQTTv5  # wtf

theToken = password
client.username_pw_set("usernamexxatw", password=theToken)
#####
print("connecting to broker ", broker)
client.connect(broker)  # connect

client.loop_start()
time.sleep(12)
print("publishing loop start ")
for x in range(9999):
    print(x)
    #topic =
    message = "msg#"+clientid+"_"+str(x)
    # props = paho.Properties( packet_type = paho.PUBLISH )# paho.PacketTypes.PUBLISH)
    # props.user_property= ('sfilename', 'test.txt')  # [('sfilename', 'test.txt'),('dfilename', 'test.txt')])

    props = [('sfilename', 'test.txt'), ('dfilename', 'test.txt')]
    # , user_property = props

    # not here
    # client.publish(topic, message)
    client.publish("notopic", "keep alive" ) # , user_property=props)
    time.sleep(60)

client.disconnect()  # disconnect
client.loop_stop()  # stop loop
