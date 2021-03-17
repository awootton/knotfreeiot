
import asyncio
import json

import logging
import uuid
#from typing import Union, Sequence
from gmqtt import Client as MQTTClient

logger = logging.getLogger(__name__)

class Subscriptionxxx:
    def __init__(self, topic, qos=0, no_local=False, retain_as_published=False, retain_handling_options=0,
                 subscription_identifier=None):
        self.topic = topic
        self.qos = qos
        self.no_local = no_local
        self.retain_as_published = retain_as_published
        self.retain_handling_options = retain_handling_options

        self.mid = None
        self.acknowledged = False

        # this property can be used only in MQTT5.0
        self.subscription_identifier = subscription_identifier


class Client:
    def __init__(self, topic, payload, qos=0, retain=False, **kwargs):
        self.topic = topic
        self.qos = qos
        self.retain = retain
        self.dup = False
        self.properties = kwargs

        self.MQTTClient = None ##


class AccessConfig:
    def __init__(self):
        self.MQTT_server = "localhost:1883"
        self.MQTT_websocket = "ws://localhost:8085/mqtt"
        self.MQTT_password = "none"         # keep this secret.
        self.WiFi_name = "none"
        self.WiFi_password = "none"         # keep this secret.
 
class MasterConfig(AccessConfig):
   
    def __init__(self):

        self.comment = "This is my first admin config" 
        self.passphrase = "any string at all"   # keep this very secret.
        # note that the private_key can be recovered from the passphrase.
        # and the public_key can be recovered from the private_key. 
        self.public_key = "=KdsbMweME9Dw29FKooIDAd0qtZIH_ethPBmk3r0UGzk"                # keep this very secret.
        self.private_key = "=4eDP7Zy6aRB-TvZdex2QpBsWIiF_aIdkBI-D2rmRhwE"

        self.log_channel = "KdsbMweME9Dw29FKooIDAd0qtZIH_ethPBmk3r0UGzk_logs" 
        self.broadcast_channel = "KdsbMweME9Dw29FKooIDAd0qtZIH_ethPBmk3r0UGzk_broadcast_channel" 
       # self.restriction_level = 0;# zero means all permissions aka super admin.


class DeviceConfig(AccessConfig):
   
    def __init__(self):

        self.comment = "The playground light."
        self.human_name = "KdsbMweME9Dw29FKooIDAd0qtZIH_ethPBmk3r0UGzk_playground_light"
        
        self.public_key = "=CLtx33J1MrXgfe1j5ybi6cXCzg16jB8qzLvK4BIHL0s"
        self.private_key = "=me5q5jWp4y19enrUQoU_GpF33Be2qMM3E8s9CQTI43E"               # keep this secret.

        self.super_admin = "=KdsbMweME9Dw29FKooIDAd0qtZIH_ethPBmk3r0UGzk" # public key of master

        self.log_channel = "KdsbMweME9Dw29FKooIDAd0qtZIH_ethPBmk3r0UGzk_logs"
        self.broadcast_channel = "KdsbMweME9Dw29FKooIDAd0qtZIH_ethPBmk3r0UGzk_broadcast_channel"
       # self.restriction_level = 0;# zero means all permissions aka super admin.
    def toJSON(self) : 
        result = "{"
        result += 'c"comment":"' + self.comment + '",\n'
        result += 'c"human_name":"' + self.human_name + '",\n'
        result += 'c"public_key":"' + self.public_key + '",\n'
        result += 'c"private_key":"' + self.private_key + '",\n'
        result += 'c"super_admin":"' + self.super_admin + '",\n'
        result += 'c"log_channel":"' + self.log_channel + '",\n'
        result += 'c"broadcast_channel":"' + self.broadcast_channel + '"}\n'
        return result

