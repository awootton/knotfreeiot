#!/usr/bin/env python3

# pip3 install paho-mqtt

import os
import io
import time
import datetime
import paho.mqtt.client as paho

import socket


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
password = '[My_token_expires:_2021-12-31,{exp:1641023999,iss:_9sh,jti:amXYKIuS4uykvPem9Fml371o,in:32,out:32,su:4,co:2,url:knotfree.net},eyJhbGciOiJFZDI1NTE5IiwidHlwIjoiSldUIn0.eyJleHAiOjE2NDEwMjM5OTksImlzcyI6Il85c2giLCJqdGkiOiJhbVhZS0l1UzR1eWt2UGVtOUZtbDM3MW8iLCJpbiI6MzIsIm91dCI6MzIsInN1Ijo0LCJjbyI6MiwidXJsIjoia25vdGZyZWUubmV0In0.7ElPyX1Vju8Q5uDOEfgAblwvE2gxT78Jl68JPlqLRcFeMJ7if39Ppl2_Jr_JTky371bIXAn6S-pghtWSqTBwAQ]'
#password = ''

targetPort = 4000
#targetPort = 3000

# define callbacks

def on_message(client, userdata, message):

    payload = str(message.payload.decode("utf-8"))

    print(str(datetime.datetime.now())+"received message =", payload)
    print("topic is " + str(message.topic))
    print("reply addr is " + str(message.properties.ResponseTopic))
    # what is this? is None print(str(userdata))
    user_properties=message.properties.UserProperty
    #print("user properties received= ",user_properties)

    responseTopic = str(message.properties.ResponseTopic)

    host = "localhost"
    port = targetPort                   # The same port as used by the server
    s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    s.connect((host, port))
    #print("sending host,port, msg",host,port,message.payload)
    s.sendall(message.payload)

    # loop here until we get it all
    contentLen = -1
    headerSize = 0
    neededSize = 99999999

    dest = bytearray()
    done = False
    startTime = time.time()
    headerendBytes = bytearray(b'\r\n\r\n')
    contentLenBytes = bytearray(b'Content-Length:')

    while  len(dest) < neededSize :
        now = time.time()
        if now - startTime > 10:
            # timed  out after 10 sec
            break
        data = s.recv(1024)
        #print(" recieved len = " , len(data))
        dest.extend(data)
        #print("dest len is now ", len(dest))
        if headerSize == 0 or contentLen == -1:
            # try to parse the header enought to know packet len
            headerEnd = dest.find(headerendBytes,0) + len(headerendBytes)
            if headerEnd > 0:
                headerSize = headerEnd
                # we must find the con
                pos = dest.find(contentLenBytes,0)
                if pos >= 0:
                    pos2 = dest.find(b'\r\n',pos)
                    clenstr = (dest[pos + len(contentLenBytes):pos2]).decode()
                    clenstr = clenstr.strip()
                    print ( "clen str is ", clenstr)
                    clen = int(clenstr)
                    contentLen = clen
                else :
                    pos = dest.find(b'Transfer-Encoding:',0)
                    if pos > 0:
                        # i'm not sure this actually works
                        pos2 = dest.find(b'chunked',pos)
                        if pos2 > 0:
                            needMore = True
                            pos1 = headerSize
                            pos2 = 0
                            while needMore:
                                #print("current data ", dest[pos1-4:])
                                pos2 = dest.find(b'\r\n',pos1)
                                if pos2 < 0:
                                    break
                                lenbytes = dest[pos1:pos2]
                                lenstr = lenbytes.decode()
                                if len(lenstr) == 0:
                                    #we're done
                                    contentLen = (pos2+2) - headerSize
                                    break
                                chunkLen = int(lenstr, 16)
                                newPos = pos2 + chunkLen + 2
                                if newPos >= len(dest) :
                                    break
                                pos1 = newPos
                        else:
                            contentLen = 0
                    else:    
                        contentLen = 0
                neededSize = headerSize + contentLen



    s.close()
    print('Received len ', len(dest))

    client.publish(responseTopic, dest)


    # topic = str(message.properties.ResponseTopic)
    # parts = str(message.payload.decode("utf-8")).split(" ")
    # path = parts[1]
    # path = path[1:] # strip off "/"

    # if os.path.exists(path) :
    #     f = open(path, "rb")
    # else:
    #     f = open("atwIndex1.html", "r")

    # body = f.read()
    # client.publish(topic, body)

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
        topic = "dummy" 
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
    topic = "dummy"
    message = "msg#"+clientid+"_"+str(x)
    #props = paho.Properties( packet_type = paho.PUBLISH )# paho.PacketTypes.PUBLISH)
    #props.user_property= ('sfilename', 'test.txt')  # [('sfilename', 'test.txt'),('dfilename', 'test.txt')])
    
    props =[('sfilename', 'test.txt'),('dfilename', 'test.txt')]
    #, user_property = props
    
 
    # not here 
    # client.publish(topic, message)
    client.publish("notopic", "keep alive")
    time.sleep(60)
     
client.disconnect()  # disconnect
client.loop_stop()  # stop loop

