

// needs nacl.js -- http://tweetnacl.cr.yp.to/
// and   mqtt.js -- https://unpkg.com/mqtt@3.0.0/dist/mqtt.js 


class Mqtt5BoxSecureMessage {

    constructor(config_json, callback, mqtt) {
        this.config = config;
        this.callback = callback;
        this.mqttClient = mqttClient;

        mqttClient = mqtt.connect(config_json.MQTT_websocket, {
            clientId: config_json.public_key,
            username: config_json.public_key,
            password: config_json.password,
            protocolVersion: 5,
            clean: true,
            reconnectPeriod: 10000,
        });
 
        mqttClient.on('connect', () => {
            log('connected, subscribing to "test" topic...');

            client.subscribe(config_json.self.human_name, {}, (err) => {
                if (err) {
                    log('failed to subscribe to topic', err);
                    return;
                }
                log('subscribed to "test" topic, nnoottpublishing message...');
            });
            client.subscribe(config_json.broadcast_channel, {}, (err) => {
                if (err) {
                    log('failed to subscribe to log channel', err);
                    return;
                }
                log('subscribed to "test" topic, nnoottpublishing message...');
            });
        });

        mqttClient.on('message', (topic, msg) => {
            log('received message in topic "' + topic + '": "' + msg.toString('utf8') + '"');
        
                // decode, set meta, call callback.
        
        });

        mqttClient.on('close', () => {
            log('close-disconnected');
        })

        mqttClient.on('error', (err) => {
            log('mqtt client error:', err);
            mqttClient.end(true) // force disconnect
        });

    };

    send(data, address , meta) {

    }

    reply(data, meta) {
        address = mets.replyAddress
        this.send(data,address,meta)
    }

}


EventEmitter.prototype.once = function once(type, listener) {
    if (typeof listener !== 'function')
        throw new TypeError('"listener" argument must be a function');
    this.on(type, _onceWrap(this, type, listener));
    return this;
};

