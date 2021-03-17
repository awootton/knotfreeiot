
import asyncio
import os
import signal
import mqtt5boxed
import json

#from mqtt5boxed import Client as BoxedClient

STOP = asyncio.Event()

'''
    We'll have a device which will be a switch with a light and we'll have an admin.
    
    admin:

        passphrase : "any string"
        public_key : "=and 43 base64 chars or $and 64 hex chars , sometimes binary?"
        private_key: "=and 43 base64 chars or $and 64 hex chars , sometimes binary?"

        log_channel: "'log_channel' or any string"
        broadcast_channel: "'broadcast_channel' or any string"

    #    device_list []


        MQTT_server: "a url. eg knotfree.net"
        MQTT_password: "**********"
        WiFi_name : "aka ssid"
        WiFi_password: ""**********""

    device:

        public_key : "=and 43 base64 chars or $and 64 hex chars , sometimes binary?"
        private_key: "=and 43 base64 chars or $and 64 hex chars , sometimes binary?"   

        admin_public_key : "same as admin"
       
        log_channel: "same as admin and all other devices"
        broadcast_channel: "same as admin and all other devices"

        MQTT_server: "a url. eg knotfree.net"
        MQTT_password: "**********"
        WiFi_name : "aka ssid"
        WiFi_password: ""**********""

'''

masterAdminExample = {
    "comment":"This is my first admin config",
    "passphrase":"any string at all",
    "public_key":"=KdsbMweME9Dw29FKooIDAd0qtZIH_ethPBmk3r0UGzk",
    "private_key":"=4eDP7Zy6aRB-TvZdex2QpBsWIiF_aIdkBI-D2rmRhwE",
    "log_channel":"KdsbMweME9Dw29FKooIDAd0qtZIH_ethPBmk3r0UGzk_logs",
    "broadcast_channel":"KdsbMweME9Dw29FKooIDAd0qtZIH_ethPBmk3r0UGzk_broadcast_channel",
}
sss = json.dumps(masterAdminExample, indent=4)
print(sss)




ddddd = DeviceConfig()
sss = ddddd.toJSON()
print(sss)
   
'''
{
    "comment": "This is my first admin config",
    "passphrase": "any string at all",
    "public_key": "=KdsbMweME9Dw29FKooIDAd0qtZIH_ethPBmk3r0UGzk",
    "private_key": "=4eDP7Zy6aRB-TvZdex2QpBsWIiF_aIdkBI-D2rmRhwE",
    "log_channel": "KdsbMweME9Dw29FKooIDAd0qtZIH_ethPBmk3r0UGzk_logs",
    "broadcast_channel": "KdsbMweME9Dw29FKooIDAd0qtZIH_ethPBmk3r0UGzk_broadcast_channel"
}
'''
 

def ask_exit(*args):
    STOP.set()


async def main(broker_host, user, passw):
    print("main")

    #client = BoxedClient()

    await STOP.wait()
    await client.disconnect()


if __name__ == '__main__':
    loop = asyncio.get_event_loop()

    host = 'localhost'
    user = "atw"
    passw = 'eyJhbGciOiJFZDI1NOYork2OeXRJKjfl9j0DQ'
   
    user = "abc"
    passw = '123'

    loop.add_signal_handler(signal.SIGINT, ask_exit)
    loop.add_signal_handler(signal.SIGTERM, ask_exit)

    loop.run_until_complete(main(host, user, passw))

