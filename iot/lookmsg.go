package iot

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/awootton/knotfreeiot/monitor_pod"
	"github.com/awootton/knotfreeiot/packets"
	"golang.org/x/crypto/nacl/box"
)

// a global for the commands
type lookupContext struct {
	CommandMap map[string]monitor_pod.Command
}

// a global for the list of commands we can handle
var lookupContextGlobal lookupContext

func init() {
	lookupContextGlobal.CommandMap = make(map[string]monitor_pod.Command)
	setupCommands(&lookupContextGlobal)
}

// different for every command
type lookupCallContext struct {
	me      *LookupTableStruct
	bucket  *subscribeBucket
	lookMsg *lookupMessage
	pubk    string
}

// It's little convoluted.
// processLookup is the entry point for executing the lookup API.
// When the lookup-table, which precesses all the packets directed at the buckets, gets a
// packets.Lookup it calls in here.
// There is a channel involved so for the duration we are in here we have exclusive access to the bucket.
// We get the 'cmd' option from the packet and look it up in the command map. If it's not found we use the 'help' command which just lists the commands.
// if the command map has it marked as requiring encryption we try to decrypt it with the public key of the owner of the name record.
// If decryption fails we return an error.
// Then we execute the command: comandStruct.Execute(cmd, args, &lcxt) where cmd is the command string, args are the arguments and
// lcxt is the lookupCallContext which has the bucket, the message and the public key of the sender.
// still in the main thread.
// this will execute the func declared in monitor_pod.MakeCommand
// Which MAY call getAndSetWatcher as a go routine in a new thread.
// one must not send a reply in that case and must wait for the getAndSetWatcher
// which MUST send a reply.

// When we come in here we have exclusive access to the bucket. Otherwise not.
// The trick is that we want to return control of the bucket before any database (mongo) access.
// If we do this when we'll have to re-q to get accress again. See callBackCommand

// NEW RULE. No lookmsg come in here without a proper session value. This means that the service-contact has to have a session Value for every packet it sends.
// If you need to call this somehow not from the service-contact then you need to make a session Value and put it in the packet.
// No, Copilot, this is NOT a security measure: Copilot:This is a security measure to prevent replay attacks.
// The sessionValue is a random 28 byte value that is unique for each packet.

func processLookup(me *LookupTableStruct, bucket *subscribeBucket, lookmsg *lookupMessage) {

	_, ok := lookmsg.p.GetOption(SessionKeyString)
	if !ok {
		fmt.Println("processLookup ERROR lookmsg has no sessionValue. Very naughty. This is a violation of the new rules. This should never happen. ", lookmsg.p.Sig())
	}

	if !me.isGuru {
		// fmt.Println("processLookup PushUp", me.ex.Name)
		err := bucket.looker.PushUp(lookmsg.p, lookmsg.topicHash)
		if err != nil {
			fmt.Println("processLookup PushUp error: ", err)
		}
		return
	}

	// fmt.Println("processLookup TOP:", me.ex.Name, lookmsg.p.Sig(), " v", version)

	// else we are the guru or we have no upstream
	// We will handle it here.
	// find the command
	cmd := ""
	tmp, ok := lookmsg.p.GetOption("cmd")
	if ok {
		cmd = string(tmp)
	}
	cmd = strings.TrimSpace(cmd)
	parts := strings.Split(cmd, " ")
	var comandStruct monitor_pod.Command
	ok2 := false
	var args []string
	if len(parts) > 1 { // try two word command match
		tmp := strings.ToLower(parts[0] + " " + parts[1])
		comandStruct, ok2 = lookupContextGlobal.CommandMap[tmp]
		args = parts[2:]
	}
	if !ok2 { // try one word command
		comandStruct, ok2 = lookupContextGlobal.CommandMap[strings.ToLower(parts[0])]
		args = parts[1:]
	}
	if !ok2 {
		// make this a get option txt for default?
		comandStruct = lookupContextGlobal.CommandMap["help"]
	}
	pubk, ok := lookmsg.p.GetOption("pubk")
	if !ok {
		pubk = []byte("")
	}

	lcxt := lookupCallContext{me, bucket, lookmsg, string(pubk)}

	RewriteSessionKeyToCheckLookup(lookmsg.p) // make sure the session value is copied to the send packet so we can match the reply to the request.

	{
		// Do we need to timeout in here?
		startTime := time.Now()

		if serviceDebugSession1 {
			fmt.Println("processLookup have command:", comandStruct.CommandString)
		}

		// does it require encryption?
		// todo: don't string compare and use a flag and defer the decryption?
		requiresEncryption := !strings.Contains(comandStruct.Description, "🔓")
		encryptedGood := true
		if requiresEncryption {
			encryptedGood = decryptCommand(me, lookmsg.p, cmd)
		}

		reply := ""
		if encryptedGood {
			// let's base64 decode any args that start with '='
			for i, arg := range args {
				if strings.HasPrefix(arg, "=") {
					decoded, err := base64.RawURLEncoding.DecodeString(arg[1:])
					if err != nil {
						sendReply(me, lookmsg, "error: invalid base64 encoding")
						return
					}
					args[i] = string(decoded)
				}
			}
			comandStruct.Execute(cmd, args, &lcxt)
			return
		}
		reply = "error: decryption failed"
		// now send the reply back. This is an example of a reply
		send := packets.Send{}
		send.Address = lookmsg.p.Source
		send.Source = lookmsg.p.Address
		send.CopyOptions(&lookmsg.p.PacketCommon)
		send.Payload = []byte(reply)
		// if requiresEncryption && encryptedGood {
		// 	// encrypt the answer TODO: do we need this?
		// }

		delta := time.Since(startTime)
		fmt.Println("processLookup BOTTOM:", lookmsg.p.Sig(), delta, reply)
		if len(me.ex.channelToAnyAide) >= cap(me.ex.channelToAnyAide) {
			fmt.Println("processLookup ERROR me.ex.channelToAnyAide channel full")
		}
		// can we have just ONE sendReply please?
		send.DeleteOption("cmd")
		send.SetOption("processed_by_lookup", []byte("1"))

		CheckSendPacket(&send)
		me.ex.channelToAnyAide <- &send
	}
}

func sendReply(me *LookupTableStruct, lookmsg *lookupMessage, reply string) {
	send := packets.Send{}
	send.Address = lookmsg.p.Source
	send.Source = lookmsg.p.Address
	send.CopyOptions(&lookmsg.p.PacketCommon)
	send.Payload = []byte(reply)
	if len(me.ex.channelToAnyAide) >= cap(me.ex.channelToAnyAide) {
		fmt.Println("processLookup sendReply ERROR me.ex.channelToAnyAide channel full")
	}

	if serviceDebugSession1 {
		cmd, _ := lookmsg.p.GetOption("cmd")
		fmt.Println("processLookup sending reply to:", string(cmd), "reply:", reply)
	}

	// you know, we don't really need the "cmd" key in the reply option's.
	// Sometimes the service-command is getting an echo back from aide instead of this proper reply.
	// That echo is eating the real reply. 	So, let's remove the "cmd" option from the reply. It's not needed anyway.
	// we should add a definitive "processed by lookup" option to the reply so the service-command can tell the difference between a real reply and an echo from aide.
	send.DeleteOption("cmd")
	send.SetOption("processed_by_lookup", []byte("1"))

	RewriteSessionKeyToCheck(&send)
	CheckSendPacket(&send) // always?

	// now, service-command can throw any replies with a "cmd" option away and not get confused.
	me.ex.channelToAnyAide <- &send
}

func getCallContext(calContest interface{}) (*LookupTableStruct, *subscribeBucket, *lookupMessage, string) {
	ctx := calContest.(*lookupCallContext)
	return ctx.me, ctx.bucket, ctx.lookMsg, ctx.pubk
}

type LookupNameExistsReturnType struct {
	Exists bool
	Online bool
	Owner  string // is it safe to publically return the owner pubk?
}

type ProxyStatusReturnType struct {
	Exists bool
	Online bool
	Static string // path to static files aka FORWARD
	Proxy  string // path to proxy/ Are these the same?
}

type lookBackCommand struct {
	callContext interface{}

	// cmd is 'this' aka 'self'.
	// callback handles the case after we have a name
	callback func(me *LookupTableStruct, bucket *subscribeBucket, cmd *callBackCommand)
}

func (cb *lookBackCommand) Run(me *LookupTableStruct, bucket *subscribeBucket) {
	cb.callback(me, bucket, nil)
}

// we must NOT both send a reply and also re-q something in bucket.incoming
func getAndSetWatcher(callContext interface{}, finish func(callContext interface{}, watchedTopic *WatchedTopic), makeName func(callContext interface{}, name string)) {
	// get the watcher
	// set the watcher, as necessary
	// call finish
	me, bucket, lookMsg, pubk := getCallContext(callContext)
	_ = me
	_ = pubk
	watchedTopic, ok := getWatcher(bucket, &lookMsg.topicHash)
	if ok { // we have it
		// we can be done now. no need to q to bucket.incoming
		finish(callContext, watchedTopic)
		return
	}
	// ok, we don't have it and that means we'll be going to mongo. But, mongo might already know,
	// right away that it diesn;t exist.
	str := lookMsg.topicHash.ToBase64()

	notgonna := GetNoneSubscription(str)
	if notgonna {
		sendReply(me, lookMsg, "status: topic not found errid=bvBbhJawYXIMWsxJOWHt")
		return
	}
	// we're going to end up in the q to bucket.incoming. Blech.
	// we'll call this and heaven help who's waiting.
	go func() {
		// checkMongo
		gotwatchedTopic, ok := GetSubscription(str)
		if !ok {
			if makeName != nil {
				// make a new one
				makeName(callContext, str)
				// keep going
				gotwatchedTopic, ok = GetSubscription(str)
				if !ok {
					sendReply(me, lookMsg, "error: topic failed to make")
					return
				}
			} else {
				// don't make a new one
				sendReply(me, lookMsg, "status: topic not found errid=bvBbhJawYXIMWsxJOWHt")
				return
			}
		}
		watchedTopic = gotwatchedTopic
		// now acquire the bucket again and finish.
		mmm := lookBackCommand{
			callContext: callContext,
			callback: func(me *LookupTableStruct, bucket *subscribeBucket, cmd *callBackCommand) {
				// remember it
				setWatcher(bucket, &lookMsg.topicHash, gotwatchedTopic)

				finish(callContext, watchedTopic)
			},
		}
		// fmt.Println("sending to incoming q whose length is now ", len(bucket.incoming))
		bucket.incoming <- &mmm
	}()
}

// set up for some commands
func setupCommands(c *lookupContext) {

	// we want to know, in one call
	// if there is a subscriber than we can tunnel http to.
	// If not can we proxy to a known address? via A record?
	// do we forward to a host or just an ip?

	monitor_pod.MakeCommand("details",
		"A serialization of the name record", 0,
		func(msg string, args []string, callContext interface{}) string {

			getAndSetWatcher(callContext, func(callContext interface{}, watchedTopic *WatchedTopic) {
				me, bucket, lookMsg, pubk := getCallContext(callContext)
				_ = bucket
				if pubk != watchedTopic.Owner {
					sendReply(me, lookMsg, "error: not owner")
					return
				}
				json, err := json.Marshal(watchedTopic)
				if err != nil {
					sendReply(me, lookMsg, "json error: "+err.Error())
					return
				}
				// fmt.Println("details returns ", string(json))
				sendReply(me, lookMsg, string(json))
			}, nil)
			return ""
		}, c.CommandMap)

	// how does this work. When there's no watcher it will have to re-q and thencall mongo GetSubscription to get the subscription.
	// Then it will have to re-q the bucket to set the watcher. Then it will have to call the callback function.
	// This is dumb if the the mongo cache already know this thing doesn't exist and that happens 87.5% of of the time.
	monitor_pod.MakeCommand("get option",
		"get key val. eg A 12.34.56.78 🔓", 0,
		func(msg string, args []string, callContext interface{}) string {

			if len(args) < 1 {
				return "error: not enough arguments"
			}
			key := args[0]
			subKey := "@"
			if len(args) > 1 {
				subKey = args[1]
			}

			// fmt.Println("get option TOP", key, subKey)
			getAndSetWatcher(callContext, func(callContext interface{}, watchedTopic *WatchedTopic) {
				me, bucket, lookMsg, _ := getCallContext(callContext)
				_ = bucket
				key = strings.ToUpper(key)
				val, ok := watchedTopic.GetOption(key)
				if !ok {
					val = []byte("")
				}
				optionMap := StringToMap(string(val))
				subValue, ok := optionMap[subKey]
				if !ok {
					if key == "A" && subKey == "@" { // a total hack where the default of A,@ is knotfree.io
						subValue = "216.128.128.195"
					} else {
						sendReply(me, lookMsg, "error: not found errid=nGToaKwTIhaIxBmgxjvY "+key+" "+subKey)
						return
					}
				}
				// fmt.Println("get option returning", key, subValue)
				sendReply(me, lookMsg, subValue)
			}, nil)
			return ""

		}, c.CommandMap)

	monitor_pod.MakeCommand("get txt", // same as get option TXT
		"get key val. eg A 12.34.56.78 🔓", 0,
		func(msg string, args []string, callContext interface{}) string {

			key := "TXT"
			// fmt.Println("get txt")
			subKey := "@"
			if len(args) > 0 {
				subKey = args[0]
			}

			getAndSetWatcher(callContext, func(callContext interface{}, watchedTopic *WatchedTopic) {
				me, bucket, lookMsg, _ := getCallContext(callContext)
				_ = bucket

				val, ok := watchedTopic.GetOption(strings.ToUpper(key))
				if !ok {
					val = []byte("")
				}
				optionMap := StringToMap(string(val))
				subValue, ok := optionMap[subKey]
				if !ok {
					sendReply(me, lookMsg, "status: not found"+key+" "+subKey)
					return
				}
				sendReply(me, lookMsg, subValue)
			}, nil)
			return ""

		}, c.CommandMap)

	monitor_pod.MakeCommand("set option",
		"add key subkey value. eg A @ 12.34.56.78 ", 0,
		func(msg string, args []string, callContext interface{}) string {
			me, bucket, lookMsg, pubk := getCallContext(callContext)
			_ = bucket
			_ = pubk

			// subkey is REQUIRED?
			if len(args) < 3 {
				//return "error: not enough arguments"
				// sendReply(me, lookMsg, "ok")
				if len(args) == 2 {
					args = append(args[0:1], "@", args[1])
				} else {
					sendReply(me, lookMsg, "error: not enough arguments")
				}
			}
			key := strings.ToUpper(args[0])
			subKey := args[1]
			newOptionVal := args[2]
			// this was done to ALL the args.
			// if strings.HasPrefix(newOptionVal, "=") {
			// 	// it's base 64 encoded. decode it.
			// 	decoded, err := base64.RawURLEncoding.DecodeString(newOptionVal[1:])
			// 	if err != nil {
			// 		sendReply(me, lookMsg, "error: invalid base64 encoding")
			// 	}
			// 	newOptionVal = string(decoded)
			// }

			fmt.Println("processLookup set option", key, newOptionVal, subKey)

			getAndSetWatcher(callContext, func(callContext interface{}, watchedTopic *WatchedTopic) {
				me, bucket, lookMsg, pubk := getCallContext(callContext)
				_ = bucket
				if pubk != watchedTopic.Owner {
					sendReply(me, lookMsg, "error: not owner")
					return
				}
				val, ok := watchedTopic.GetOption(strings.ToUpper(key))
				if !ok {
					val = []byte("")
				}
				optionMap := StringToMap(string(val))
				optionMap[subKey] = newOptionVal
				newMapAsStr := MapToString(optionMap)
				watchedTopic.SetOption(key, newMapAsStr)
				// save to mongo !
				SaveSubscription(watchedTopic)

				sendReply(me, lookMsg, "ok")
			}, nil)
			return ""

		}, c.CommandMap)

	monitor_pod.MakeCommand("bulk option",
		"add key kv pairs", 0,
		func(msg string, args []string, callContext interface{}) string {
			me, bucket, lookMsg, pubk := getCallContext(callContext)
			_ = bucket
			_ = pubk
			// subkey is REQUIRED?
			if len(args) < 2 {
				sendReply(me, lookMsg, "error: not enough arguments")
				return ""
			}
			key := strings.ToUpper(args[0])
			bulkVals := args[1:]
			fmt.Println("processLookup bulk option", key, bulkVals)

			getAndSetWatcher(callContext, func(callContext interface{}, watchedTopic *WatchedTopic) {
				me, bucket, lookMsg, pubk := getCallContext(callContext)
				_ = bucket
				if pubk != watchedTopic.Owner {
					sendReply(me, lookMsg, "error: not owner")
					return
				}
				val, ok := watchedTopic.GetOption(strings.ToUpper(key))
				if !ok {
					val = []byte("")
				}
				optionMap := StringToMap(string(val))
				parts := bulkVals
				for i := 0; i < len(parts)-1; i += 2 {
					subKey := parts[i]
					newOptionVal := parts[i+1]
					optionMap[subKey] = newOptionVal
				}

				newMapAsStr := MapToString(optionMap)
				watchedTopic.SetOption(key, newMapAsStr)
				// save to mongo !
				SaveSubscription(watchedTopic)

				sendReply(me, lookMsg, "ok")
			}, nil)
			return ""

		}, c.CommandMap)

	// todo: needs test
	monitor_pod.MakeCommand("replace options",
		"Replace all the options. Arg is json map in base64.", 0,
		func(msg string, args []string, callContext interface{}) string {
			me, bucket, lookMsg, pubk := getCallContext(callContext)
			_ = bucket
			_ = pubk

			if len(args) < 1 {
				sendReply(me, lookMsg, "error: not enough arguments")
				return ""
			}
			newOptionsString64 := args[0]
			newOptionsString, err := base64.RawURLEncoding.DecodeString(newOptionsString64)
			if err != nil {
				sendReply(me, lookMsg, "replace options error: "+err.Error())
				return ""
			}
			fmt.Println("processLookup replace options", newOptionsString)

			getAndSetWatcher(callContext, func(callContext interface{}, watchedTopic *WatchedTopic) {
				me, bucket, lookMsg, pubk := getCallContext(callContext)
				_ = bucket
				if pubk != watchedTopic.Owner {
					sendReply(me, lookMsg, "error: not owner")
					return
				}
				aMap := make(map[string]string)
				err := json.Unmarshal([]byte(newOptionsString), &aMap)
				if err != nil {
					sendReply(me, lookMsg, "replace options error: "+err.Error())
					return
				}
				watchedTopic.ReplaceOptions(aMap)

				// save to mongo !
				SaveSubscription(watchedTopic)

				sendReply(me, lookMsg, "ok")
			}, nil)
			return ""

		}, c.CommandMap)

	monitor_pod.MakeCommand("proxy-status",
		"returns ProxyStatusReturnType 🔓", 0,
		func(msg string, args []string, callContext interface{}) string {

			status := ProxyStatusReturnType{false, false, "", ""}

			// fmt.Println("proxy-status TOP")

			getAndSetWatcher(callContext, func(callContext interface{}, watchedTopic *WatchedTopic) {
				me, _, lookMsg, _ := getCallContext(callContext)

				// fmt.Println("proxy-status has watcher")

				if watchedTopic == nil {
					bytes, _ := json.Marshal(status)
					sendReply(me, lookMsg, string(bytes))
					return
				}
				status.Exists = true
				val, ok := watchedTopic.GetOption("FORWARD")
				if ok {
					optionMap := StringToMap(string(val))
					status.Static = optionMap["@"]
				}
				val, ok = watchedTopic.GetOption("PROXY")
				if ok {
					optionMap := StringToMap(string(val))
					status.Proxy = optionMap["@"]
				}
				status.Online = !watchedTopic.thetree.Empty()

				bytes, _ := json.Marshal(status)
				sendReply(me, lookMsg, string(bytes))
			}, nil)

			return ""

		}, c.CommandMap)
	monitor_pod.MakeCommand("exists",
		"returns true if the name exists 🔓", 0,
		func(msg string, args []string, callContext interface{}) string {

			exists := LookupNameExistsReturnType{false, false, ""}
			me, bucket, lookMsg, _ := getCallContext(callContext)

			fmt.Println("top of exists")

			watchedTopic, ok := getWatcher(bucket, &lookMsg.topicHash)
			if ok { // we have it. It was loaded already
				// we can be done now
				exists.Exists = true
				//it's loaded but no subscribers?
				exists.Online = !watchedTopic.thetree.Empty()
				exists.Owner = watchedTopic.Owner
				s, _ := json.Marshal(exists)
				sendReply(me, lookMsg, string(s))
				return ""
			}
			// it wasn't loaded. We have try to load it.
			str := lookMsg.topicHash.ToBase64()
			// we have to do this in a go routine because we have to release the bucket
			// we will lose exclusive access to the bucket now.
			go func() {
				// checkMongo
				fmt.Println("exists check mongo")
				gotwatchedTopic, ok := GetSubscription(str)
				fmt.Println("exists got mongo", ok)
				if !ok {
					exists.Exists = false
					exists.Online = false
					exists.Owner = ""
					s, _ := json.Marshal(exists)
					sendReply(me, lookMsg, string(s))
					return
				}
				watchedTopic = gotwatchedTopic
				// now acquire the bucket again and do the setWatcher, since we have it now.
				mmm := lookBackCommand{
					callContext: callContext,
					callback: func(me *LookupTableStruct, bucket *subscribeBucket, cmd *callBackCommand) {
						// remember it
						setWatcher(bucket, &lookMsg.topicHash, gotwatchedTopic)
						exists.Exists = true
						exists.Online = false
						exists.Owner = watchedTopic.Owner
						s, _ := json.Marshal(exists)
						sendReply(me, lookMsg, string(s))
					},
				}
				fmt.Println("sending to incoming q (2) whose length is now ", len(bucket.incoming))

				bucket.incoming <- &mmm
			}()
			return ""

		}, c.CommandMap)

	monitor_pod.MakeCommand("get time",
		"seconds since 1970🔓", 0,
		func(msg string, args []string, callContext interface{}) string {
			sec := time.Now().UnixMilli() / 1000
			secStr := strconv.FormatInt(sec, 10)
			me, _, lookMsg, _ := getCallContext(callContext)

			sendReply(me, lookMsg, secStr)
			return ""
		}, c.CommandMap)
	monitor_pod.MakeCommand("get random",
		"returns a random integer", 0,
		func(msg string, args []string, callContext interface{}) string {
			tmp := rand.Uint32()
			secStr := strconv.FormatInt(int64(tmp), 10)
			me, _, lookMsg, _ := getCallContext(callContext)
			sendReply(me, lookMsg, secStr)
			return ""
		}, c.CommandMap)
	monitor_pod.MakeCommand("get pubk",
		"device public key 🔓", 0,
		func(msg string, args []string, callContext interface{}) string {
			//  this is the public key of the cluster
			me, _, lookMsg, _ := getCallContext(callContext)
			str := base64.RawURLEncoding.EncodeToString(me.ex.ce.PublicKeyTemp[:])
			sendReply(me, lookMsg, str)
			return ""
		}, c.CommandMap)
	monitor_pod.MakeCommand("version",
		"info about this thing", 0,
		func(msg string, args []string, callContext interface{}) string {
			me, _, lookMsg, _ := getCallContext(callContext)
			sendReply(me, lookMsg, "v0.1.7")
			return ""
		}, c.CommandMap)
	monitor_pod.MakeCommand("help",
		"lists all commands. 🔓 means no encryption required", 0,
		func(msg string, args []string, callContext interface{}) string {
			s := ""
			if msg != "help" {
				s = "// Unknown command: " + msg + " \n"
			}
			keys := make([]string, 0, len(c.CommandMap)) //  maps.Keys(c.CommandMap)
			for k := range c.CommandMap {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				command := c.CommandMap[k]
				argCount := ""
				if command.ArgCount > 0 {
					argCount = " +" + strconv.FormatInt(int64(command.ArgCount), 10)
				}
				s += "[" + k + "]" + argCount + " " + command.Description + "\n"
			}

			me, _, lookMsg, _ := getCallContext(callContext)
			sendReply(me, lookMsg, s)
			return ""
		}, c.CommandMap)

	monitor_pod.MakeCommand("reserve",
		"assign a public key to a name, create", 0,
		createNameFunc, c.CommandMap)

	monitor_pod.MakeCommand("delete",
		"delete a name", 0,
		deleteNameFunc, c.CommandMap)

}

// FIXME: return error
func decryptCommand(me *LookupTableStruct, p *packets.Lookup, command string) bool {
	// ourPrivKey := me.ex.ce.PrivateKeyTemp
	sealed, ok := p.GetOption("sealed")
	if !ok {
		return false
	}
	nonce, ok := p.GetOption("nonc")
	if !ok {
		return false
	}
	pubk, ok := p.GetOption("pubk")
	if !ok {
		return false
	}
	pubkBytes, err := base64.RawURLEncoding.DecodeString(string(pubk))
	if err != nil {
		return false
	}
	var nonce2 [24]byte
	copy(nonce2[:], nonce)
	var pubk2 [32]byte
	copy(pubk2[:], pubkBytes)

	out := make([]byte, 0, len(sealed)) // it's actually smaller
	result, ok := box.Open(out, sealed, &nonce2, &pubk2, me.ex.ce.PrivateKeyTemp)
	if !ok {
		return false
	}
	// split the payload into command and time
	payload := string(result)
	parts := strings.Split(payload, "#")
	if len(parts) < 2 {
		return false
	}
	// check the time
	t, err := strconv.ParseInt(parts[len(parts)-1], 10, 64)
	if err != nil {
		return false
	}
	diff := time.Now().Unix() - t
	if diff <= 0 {
		diff = -diff
	}
	if diff > 10 { // 10 seconds
		return false
	}
	cmdtmp := parts[0]
	command = strings.TrimSpace(command)
	cmdtmp = strings.TrimSpace(cmdtmp)
	// check the command.
	if command != cmdtmp {
		fmt.Println("command mismatch", cmdtmp, command)
		return false
	}
	return true
}

//	export function StringToMap(str: string): Map<string, string> {
//	    const map = new Map<string, string>();
//	    let tmp = str.trim()
//	    const entries = tmp.split(' ');
//	    if (entries.length === 0)
//	        return map;
//	    if (entries.length === 1) {
//	        map.set('@', entries[0].trim());
//	        return map;
//	    }
//	    for (let i = 0; i < entries.length; i++) {
//	        let key = entries[i];
//	        let val = entries[i + 1];
//	        i += 1;
//	        map.set(key.trim(), val.trim());
//	    }
//	    return map;
//	}
func StringToMap(str string) map[string]string {
	m := make(map[string]string)
	tmp := strings.TrimSpace(str)
	if len(tmp) == 0 {
		return m
	}
	entries := strings.Split(tmp, " ")
	if len(entries) == 1 {
		m["@"] = entries[0]
		return m
	}
	for i := 0; i < len(entries); i++ {
		key := entries[i]
		val := entries[i+1]
		i++
		m[key] = val
	}
	return m
}

//	export function MapToString(map: Map<string, string>): string {
//	    let str = '';
//	    for (let [key, value] of map) {
//	        str += key + ' ' + value + ' ';
//	    }
//	    return str.trim();
//	}
func MapToString(m map[string]string) string {
	str := strings.Builder{}
	for key, value := range m {
		str.WriteString(key)
		str.WriteString(" ")
		str.WriteString(value)
		str.WriteString(" ")
	}
	return str.String()[0 : str.Len()-1]
}

// Copyright 2019,2020,2021,2026 Alan Tracey Wootton
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.
