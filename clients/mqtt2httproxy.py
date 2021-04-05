#!/usr/bin/env python3

# pip3 install paho-mqtt

import os
import io
import time
import datetime
import paho.mqtt.client as paho

from http.server import BaseHTTPRequestHandler

# from http_parser.http import HttpStream
# from http_parser.reader import SocketReader

import requests


class HTTPRequest(BaseHTTPRequestHandler):
    def __init__(self, request_text):
        self.rfile = io.BytesIO(request_text)
        self.raw_requestline = self.rfile.readline()
        self.error_code = self.error_message = None
        self.parse_request()

    def send_error(self, code, message):
        self.error_code = code
        self.error_message = message

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

def parseHttp( httppacket ):
    lines = []
    content = ""
    result = 200

    prev = 0
    i = 0
    theLen = len(httppacket)
    for i in range(theLen):
        c = httppacket[i]
        if c == 10: # line feed
            theline = httppacket[prev:i]
            if theline == "":
                break
            lines.append(theline)
            prev = i + 1

    content = httppacket[i+1:]

    return lines,content,result

# define callbacks

def on_message(client, userdata, message):

    payload = str(message.payload.decode("utf-8"))

    print(str(datetime.datetime.now())+"received message =", payload)
    print("topic is " + str(message.topic))
    print("reply addr is " + str(message.properties.ResponseTopic))
    # what is this? is None print(str(userdata))
    user_properties=message.properties.UserProperty
    #print("user properties received= ",user_properties)

    # this is horrible: I'll do it myself
    # request = HTTPRequest(message.payload) #  parse it

    # print( "request err code ", request.error_code    )
    # print( "request method ", request.command  ) # GET
    # print( "request path ", request.path  ) # /
    # print( "request protocol_version ", request.protocol_version  ) # HTTP/1.0
    # print( "request raw_requestline ", request.raw_requestline  ) # b'GET / HTTP/1.1\n'
    # print( "request headers ", request.headers    )
    # print( "request content ", request.get_payload()    ) # what is this?? 

    lines,content,resultCode = parseHttp(message.payload)
    if len(lines) <= 1:
        return 

    parts = str(lines[0]).split(" ")
    print( "the parts are 3 ", parts) # ["b'GET", '/', "HTTP/1.1'"]

    method = str(parts[0])[2:] # skip the b'

    # api-endpoint
    URL = "http://localhost:4000" + parts[1]
    
    # location given here
    location = "my location here"
    
    # defining a params dict for the parameters to be sent to the API
    PARAMS = {'address':location}
    PARAMS = {}
    
    # sending get request and saving the response as response object
    #r = requests.get(url = URL, params = PARAMS, headers=request.headers )

    r = None

    if method == "GET":
        r = requests.get( url = URL, headers=request.headers )
    elif method == "POST":
        r = requests.post( url = URL, headers=request.headers , data = content )
    elif method == "PUT":
        r = requests.put( url = URL, headers=request.headers , data = content )
    else:
        print(" do we have to do option and delete")

    if r.status_code != 200 :
        return # make no publish/reply
    # extracting data in json format
    httpData = r.content
    print( "http request returned ", data )

    topic = str(message.properties.ResponseTopic)

    # we need to make it back to a string

    client.publish(topic, httpData)


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

