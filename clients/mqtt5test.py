#!/usr/bin/python3

import asyncio
import os
import signal
import time
import datetime
import threading
from gmqtt import Client as MQTTClient

import _thread

# pip3 install gmqtt
# pip3 install uvloop

# gmqtt also compatibility with uvloop  
import uvloop
asyncio.set_event_loop_policy(uvloop.EventLoopPolicy())
STOP = asyncio.Event()

def on_connect(client, flags, rc, properties):
    print('Connected')
    client.subscribe('TEST/TIMEabcd', qos=0)
    time.sleep(1)
    props = [("key1","val1"),("key2","val2")]
    client.subscribe('TEST/TIMEefghijk', qos=1,user_property=props)


def on_message(client, topic, payload, qos, properties):
    returnAddress = properties['response_topic']
    userProps = properties['user_property']
    print('RECV MSG:', payload, " at ", topic, " from ", str(returnAddress), " with ", userProps)

def on_disconnect(client, packet, exc=None):
    print('Disconnected')

def on_subscribe(client, mid, qos, properties):
    i = len(client.subscriptions) - 1
    if i >= 0 :
        print('SUBSCRIBED to ' + client.subscriptions[i].topic)
    else:
        print('SUBSCRIBED to ???' )
    
def ask_exit(*args):
    STOP.set()

async def publish_loop( client ):
    count = 1
    while(True):
        #break
        msg = "message at " + str(datetime.datetime.now()) + " c=" + str(count)
        topic = 'TEST/TIMEabcd'
        props = [("key1","val1"),("key2","val2")]
        client.publish(topic, msg , qos=1 ,response_topic='TEST/TIMEefghijk',user_property=props)
        count += 1
        print("looping...")
        await asyncio.sleep(10)

async def main(broker_host, user, passw):
    client = MQTTClient("client-id-uwjnbnegfgtwfqk")

    client.on_connect = on_connect
    client.on_message = on_message
    client.on_disconnect = on_disconnect
    client.on_subscribe = on_subscribe

    client.set_auth_credentials(user, passw)
    await client.connect(broker_host)

    asyncio.create_task(publish_loop(client))

    await STOP.wait()
    await client.disconnect()
  

if __name__ == '__main__':
    loop = asyncio.get_event_loop()

    host = 'mqtt.flespi.io'
    user = os.environ.get('FLESPI_TOKEN')
    user = 'GD9AyM3vUy9Hi5BzNk4hdkil6j9jGgccJ9B6LurwQr1YkroXBg4EqcOljNsQRsQr'
    passw = ""
 
    host = 'localhost'
    user = "atw"
    passw = 'eyJhbGciOiJFZDI1NTE5IiwidHlwIjoiSldUIn0.eyJleHAiOjE2MDkzNzI4MDAsImlzcyI6Ii85c2giLCJqdGkiOiIwZ2dBNFJIRjV0czdxOWNxTC9NK2czemMiLCJpbiI6MzIsIm91dCI6MzIsInN1Ijo0LCJjbyI6MiwidXJsIjoia25vdGZyZWUubmV0In0.43JcJsNNeCvElP8ZF9IT6G5vkhgPN85wsE8o2_6h7SKvTSsEWR0ldP5H9bP-NPymrCOYork2OeXRJKjfl9j0DQ'
   
    user = "abc"
    passw = '123'

    loop.add_signal_handler(signal.SIGINT, ask_exit)
    loop.add_signal_handler(signal.SIGTERM, ask_exit)

    loop.run_until_complete(main(host, user, passw))