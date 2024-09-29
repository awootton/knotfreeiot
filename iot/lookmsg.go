// Copyright 2019,2020,2021-2024 Alan Tracey Wootton
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

// processLookup  executes the lookup api.
// When we come in here we have exclusive access to the bucket. Otherwise not.
// The trick is that we want to return control of the bucket before any database (mongo) access.
// If we do this when we'll have to re-q to get accress again. See callBackCommand
func processLookup(me *LookupTableStruct, bucket *subscribeBucket, lookmsg *lookupMessage) {

	if !me.isGuru {
		fmt.Println("processLookup PushUp", me.ex.Name)
		err := bucket.looker.PushUp(lookmsg.p, lookmsg.topicHash)
		if err != nil {
			fmt.Println("processLookup PushUp error: ", err)
		}
		return
	}

	fmt.Println("processLookup TOP:", me.ex.Name, lookmsg.p.Sig())

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

	{
		// Do we need to timeout in here?
		startTime := time.Now()

		fmt.Println("processLookup have command:", comandStruct.CommandString)

		// does it require encryption?
		// todo: don't string compare and use a flag and defer the decryption?
		requiresEncryption := !strings.Contains(comandStruct.Description, "🔓")
		encryptedGood := true
		if requiresEncryption {
			encryptedGood = decryptCommand(me, lookmsg.p, cmd)
		}

		reply := ""
		if encryptedGood {
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
			fmt.Println("ERROR me.ex.channelToAnyAide channel full")
		}
		me.ex.channelToAnyAide <- &send
	} // (comandStruct, lcxt)
}

func sendReply(me *LookupTableStruct, lookmsg *lookupMessage, reply string) {
	send := packets.Send{}
	send.Address = lookmsg.p.Source
	send.Source = lookmsg.p.Address
	send.CopyOptions(&lookmsg.p.PacketCommon)
	send.Payload = []byte(reply)
	if len(me.ex.channelToAnyAide) >= cap(me.ex.channelToAnyAide) {
		fmt.Println("ERROR me.ex.channelToAnyAide channel full")
	}
	me.ex.channelToAnyAide <- &send
}

func getCallContext(calContest interface{}) (*LookupTableStruct, *subscribeBucket, *lookupMessage, string) {
	ctx := calContest.(*lookupCallContext)
	return ctx.me, ctx.bucket, ctx.lookMsg, ctx.pubk
}

type LookupNameExistsReturnType struct {
	Exists bool
	Online bool
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

func getAndSetWatcher(callContext interface{},
	finish func(callContext interface{}, watchedTopic *WatchedTopic),
	makeName func(callContext interface{}, name string)) {
	// get the watcher
	// set the watcher, as necessary
	// call finish
	me, bucket, lookMsg, pubk := getCallContext(callContext)
	_ = me
	_ = pubk
	watchedTopic, ok := getWatcher(bucket, &lookMsg.topicHash)
	if ok { // we have it
		// we can be done now
		finish(callContext, watchedTopic)
		return
	}
	str := lookMsg.topicHash.ToBase64()
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
				sendReply(me, lookMsg, "error: topic not found")
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

			fmt.Println("get option", key)
			getAndSetWatcher(callContext, func(callContext interface{}, watchedTopic *WatchedTopic) {
				me, bucket, lookMsg, pubk := getCallContext(callContext)
				_ = bucket
				if pubk != watchedTopic.Owner {
					sendReply(me, lookMsg, "error: not owner")
					return
				}
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
						sendReply(me, lookMsg, "error: not found"+key+" "+subKey)
						return
					}
				}
				sendReply(me, lookMsg, subValue)
			}, nil)
			return ""

		}, c.CommandMap)

	monitor_pod.MakeCommand("get txt", // same as get option TXT
		"get key val. eg A 12.34.56.78 🔓", 0,
		func(msg string, args []string, callContext interface{}) string {

			key := "TXT"
			fmt.Println("get txt")
			subKey := "@"
			if len(args) > 0 {
				subKey = args[0]
			}

			getAndSetWatcher(callContext, func(callContext interface{}, watchedTopic *WatchedTopic) {
				me, bucket, lookMsg, pubk := getCallContext(callContext)
				_ = bucket
				_ = pubk // get txt is not protected

				val, ok := watchedTopic.GetOption(strings.ToUpper(key))
				if !ok {
					val = []byte("")
				}
				optionMap := StringToMap(string(val))
				subValue, ok := optionMap[subKey]
				if !ok {
					sendReply(me, lookMsg, "error: get txt not found"+subKey)
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
			fmt.Println("set option", key, newOptionVal, subKey)

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
			fmt.Println("bulk option", key, bulkVals)

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
			fmt.Println("replace options", newOptionsString)

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

	monitor_pod.MakeCommand("exists",
		"returns true if the name exists 🔓", 0,
		func(msg string, args []string, callContext interface{}) string {

			exists := LookupNameExistsReturnType{false, false}
			me, bucket, lookMsg, _ := getCallContext(callContext)

			fmt.Println("top of exists")

			watchedTopic, ok := getWatcher(bucket, &lookMsg.topicHash)
			if ok { // we have it. It was loaded already
				// we can be done now
				exists.Exists = true
				//it's loaded but no subscribers?
				exists.Online = !watchedTopic.thetree.Empty()
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
						s, _ := json.Marshal(exists)
						sendReply(me, lookMsg, string(s))
					},
				}
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
	// check the command.
	if command != cmdtmp {
		fmt.Println("command mismatch", cmdtmp, command)
		return false
	}
	return true
}

// watcheditem, ok := getWatcher(bucket, &lookmsg.topicHash)
// // count := uint32(0) // people watching
// _ = watcheditem
// _ = ok

// send := packets.Send{} // this will be the reply
// send.Address = lookmsg.p.Source
// send.Source = lookmsg.p.Address
// send.CopyOptions(&lookmsg.p.PacketCommon)

// if ok {
// 	send.SetOption("isLookup", []byte("true"))

// 	//
// 	IsPermanent := len(watcheditem.Owners) > 0
// 	send.SetOption("perm", []byte(strconv.FormatBool(IsPermanent)))
// 	send.SetOption("exp", []byte(strconv.FormatUint(uint64(watcheditem.Expires), 10)))
// 	_, ok := watcheditem.IsBilling()
// 	if ok {
// 		send.SetOption("bill", []byte(strconv.FormatBool(ok)))
// 	}
// 	if watcheditem.OptionalKeyValues != nil {
// 		it := watcheditem.OptionalKeyValues.Iterator()
// 		for it.Next() {
// 			key := it.Key().(string)
// 			val := it.Value().([]byte)
// 			send.SetOption(key, val)
// 		}
// 	}

// } else {
// 	send.SetOption("isLookup", []byte("false"))
// }
// // if !ok {
// // 	// nobody watching
// // 	lookReplyObject.Null = true
// // } else {
// // 	count = uint32(watcheditem.getSize())
// // 	// todo: add more info
// // 	lookReplyObject.Null = false
// // 	lookReplyObject.Count = count
// // }
// // // set count, in decimal
// // str := strconv.FormatUint(uint64(count), 10)
// // lookmsg.p.SetOption("count", []byte(str))
// // level := int64(0)
// // levelBytes, ok := lookmsg.p.GetOption("level")
// // if ok {
// // 	level, _ = strconv.ParseInt(string(levelBytes), 10, 32)
// // }
// // level += 1
// // lookmsg.p.SetOption("level", []byte(strconv.FormatUint(uint64(level), 10)))

// // now, reply to the retrun address. With what type of message?
// // Has to be a send unless we want to add another type
// send.CopyOptions(&lookmsg.p.PacketCommon)
// // we have level
// // we have the count at this level
// nodeName := me.ex.Name

// // lookReplyObject.Level = uint32(level)
// // //lookReplyObject.Count = int(count)
// // lookReplyObject.Node = nodeName
// // repl, err := json.Marshal(lookReplyObject)
// // _ = err
// // send.Payload = repl

// val, ok := lookmsg.p.GetOption("debg")
// if ok {
// 	send.SetOption("debg", val)
// }

// if len(me.ex.channelToAnyAide) >= cap(me.ex.channelToAnyAide) {
// 	fmt.Println("me.ex.channelToAnyAide channel full")
// }
// me.ex.channelToAnyAide <- &send

// SpecialPrint(&lookmsg.p.PacketCommon, func() {
// 	json, _ := send.ToJSON()
// 	fmt.Println("Lookup channelToAnyAide because ", string(json), " in ", me.ex.Name, "on")
// })

// _ = nodeName
//}

// type xxxLookReply struct {
// 	Level       uint32
// 	Count       uint32
// 	Null        bool
// 	Node        string // node name
// 	IsPermanent bool

// 	// What else?
// }

// TODO: chop out the dead wood in subscribe etc.
// there is not one of these. Lookup replies to the return address
//func processLookupDown(me *LookupTableStruct, bucket *subscribeBucket, lookmsg *lookupMessageDown) {

//	fmt.Println("FIXME processLookupDown FIXME processLookupDown FIXME processLookupDown FIXME processLookupDown FIXME processLookupDown ")

// FIXME: needs test. This is not right. there is no processLookupDown
// the reply doesn't go down - it becomes a publish to a
// return address.

// watcheditem, ok := getWatcher(bucket, &lookmsg.h)
// count := uint32(0) // people watching
// if ok == false {
// 	// nobody watching
// } else {
// 	count = uint32(watcheditem.getSize())
// 	// todo: add more info
// }
// // set count, in decimal
// str := strconv.FormatUint(uint64(count), 10)
// lookmsg.p.SetOption("count", []byte(str))
// lookmsg.ss.WriteDownstream(lookmsg.p)

//}

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

// monitor_pod.MakeCommand("reser ve",
// 		"assign a public key to a name, create", 0,
// 		func(msg string, args []string, callContext interface{}) string {
// 			changed := false
// 			me, bucket, lookMsg := getCallContext(callContext)
// 			pubk, ok := lookMsg.p.GetOption("pubk")
// 			if !ok {
// 				return "error pubk not found"
// 			}
// 			watchedTopic, ok := getWatcher(bucket, &lookMsg.topicHash)
// 			if !ok {
// 				// checkMongo
// 				str := lookMsg.topicHash.ToBase64()
// 				watchedTopic, ok = GetSubscription(str)
// 				if !ok {
// 					changed = true
// 					watchedTopic = &WatchedTopic{}
// 					watchedTopic.Name = lookMsg.topicHash
// 					watchedTopic.thetree = NewWithInt64Comparator()
// 					// watchedTopic.Expires = 20*60 + me.getTime() this should match a token
// 					nameStr, ok := lookMsg.p.GetOption("name")
// 					if ok {
// 						watchedTopic.NameStr = string(nameStr)
// 					}

// 					t, _ := lookMsg.p.GetOption("jwtid") // don't they ALL have this?, except billing topics
// 					if len(t) != 0 {                     // it's always 64 bytes binary
// 						watchedTopic.Jwtid = string(t)
// 					}
// 					setWatcher(bucket, &lookMsg.topicHash, watchedTopic)
// 					TopicsAdded.Inc()

// 					now := me.getTime()
// 					watchedTopic.nextBillingTime = now + 30 // 30 seconds to start with
// 					watchedTopic.lastBillingTime = now
// 				}
// 				setWatcher(bucket, &lookMsg.topicHash, watchedTopic)
// 			}

// 			watchedTopic.Owner = string(pubk)
// 			// save to mongo !
// 			if changed {
// 				SaveSubscription(watchedTopic)
// 			}

// 			return "ok"
// 		}, c.CommandMap)
