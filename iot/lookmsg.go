// Copyright 2019,2020,2021 Alan Tracey Wootton
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
var lookupContextGlobal lookupContext

func init() {
	lookupContextGlobal.CommandMap = make(map[string]monitor_pod.Command)
	setupCommands(&lookupContextGlobal)
}

// a global for the commands
type lookupContext struct {
	CommandMap map[string]monitor_pod.Command
}

// different for every command
type lookupCallContext struct {
	me      *LookupTableStruct
	bucket  *subscribeBucket
	lookMsg *lookupMessage
}

func processLookup(me *LookupTableStruct, bucket *subscribeBucket, lookmsg *lookupMessage) {

	if !me.isGuru {
		fmt.Println("processLookup PushUp", me.ex.Name)
		err := bucket.looker.PushUp(lookmsg.p, lookmsg.topicHash)
		if err != nil {
			// we should be ashamed
			fmt.Println("processLookup PushUp error: ", err)
		}
		return
	}

	fmt.Println("processLookup TOP:", me.ex.Name, lookmsg.p.Sig())

	// else we are the guru or we have no upstream
	// we will handle it here.
	cmd := "exists"
	tmp, ok := lookmsg.p.GetOption("cmd")
	if ok {
		cmd = string(tmp)
	}
	cmd = strings.TrimSpace(cmd)
	cmd = strings.ToLower(cmd)
	parts := strings.Split(cmd, " ")
	var comandStruct monitor_pod.Command
	ok2 := false
	var args []string
	if len(parts) > 1 { // try two word command match
		tmp := parts[0] + " " + parts[1]
		comandStruct, ok2 = lookupContextGlobal.CommandMap[tmp]
		args = parts[2:]
	}
	if !ok2 { // try one word command
		comandStruct, ok2 = lookupContextGlobal.CommandMap[parts[0]]
		args = parts[1:]
	}
	if !ok2 {
		// make this a get option txt for default?
		comandStruct = lookupContextGlobal.CommandMap["help"]
	}
	lcxt := lookupCallContext{me, bucket, lookmsg}

	go func(comandStruct monitor_pod.Command, lcxt lookupCallContext) { // must be async because of mongo
		// Do we need to timeout in here?
		startTime := time.Now()

		fmt.Println("processLookup have command:", comandStruct.CommandString)

		// does it require encryption?
		requiresEncryption := !strings.Contains(comandStruct.Description, "🔓")
		encryptedGood := true
		if requiresEncryption {
			encryptedGood = decryptCommand(me, lookmsg.p, cmd)
		}

		reply := ""
		if !encryptedGood {
			// if we can't decrypt it. We can't do anything with it.
			reply = "decryption error"
		} else {
			reply = comandStruct.Execute(cmd, args, &lcxt)
		}

		// now send the reply back.
		send := packets.Send{}
		send.Address = lookmsg.p.Source
		send.Source = lookmsg.p.Address
		send.CopyOptions(&lookmsg.p.PacketCommon)
		send.Payload = []byte(reply)
		// if requiresEncryption && encryptedGood {
		// 	// encrypt the answer TODO:
		// }

		delta := time.Since(startTime)
		fmt.Println("processLookup BOTTOM:", lookmsg.p.Sig(), delta, reply)
		if len(me.ex.channelToAnyAide) >= cap(me.ex.channelToAnyAide) {
			fmt.Println("ERROR me.ex.channelToAnyAide channel full")
		}
		me.ex.channelToAnyAide <- &send
	}(comandStruct, lcxt)
}

func getCallContext(calContest interface{}) (*LookupTableStruct, *subscribeBucket, *lookupMessage) {
	ctx := calContest.(*lookupCallContext)
	return ctx.me, ctx.bucket, ctx.lookMsg
}

// set up for some commands
func setupCommands(c *lookupContext) {

	monitor_pod.MakeCommand("get option",
		"get key val. eg A 12.34.56.78 🔓", 0,
		func(msg string, args []string, callContext interface{}) string {

			if len(args) < 1 {
				return "error: not enough arguments"
			}
			key := args[0]
			fmt.Println("get option", key)
			me, bucket, lookMsg := getCallContext(callContext)
			_ = me
			watchedTopic, ok := getWatcher(bucket, &lookMsg.topicHash)
			if !ok {
				// checkMongo
				str := lookMsg.topicHash.ToBase64()
				watchedTopic, ok = GetSubscription(str)
				if !ok {
					// don't make a new one
					return "error: topic not found"
				}
				// remember it here
				setWatcher(bucket, &lookMsg.topicHash, watchedTopic)
			}
			val, ok := watchedTopic.GetOption(key)
			if !ok {
				if key == "a" { // a total hack where the default of A is knotfree.io
					val = []byte("216.128.128.195")
				} else {
					return "error: key not found"
				}
			}
			return string(val)
		}, c.CommandMap)

	monitor_pod.MakeCommand("get txt", // same as get option TXT
		"get key val. eg A 12.34.56.78 🔓", 0,
		func(msg string, args []string, callContext interface{}) string {

			key := "TXT"
			fmt.Println("get txt")
			me, bucket, lookMsg := getCallContext(callContext)
			_ = me
			watchedTopic, ok := getWatcher(bucket, &lookMsg.topicHash)
			if !ok {
				// checkMongo
				str := lookMsg.topicHash.ToBase64()
				watchedTopic, ok = GetSubscription(str)
				if !ok {
					// don't make a new one
					return "error: topic not found"
				}
				// remember it here
				setWatcher(bucket, &lookMsg.topicHash, watchedTopic)
			}
			val, ok := watchedTopic.GetOption(key) // TODO:get option map
			if !ok {
				return "error: txt key not found"
			}
			return string(val)
		}, c.CommandMap)

	monitor_pod.MakeCommand("set option",
		"add key val. eg A 12.34.56.78 ", 0,
		func(msg string, args []string, callContext interface{}) string {

			if len(args) < 2 {
				return "error: not enough arguments"
			}
			key := args[0]
			val := args[1]
			fmt.Println("set option", key, val)
			changed := true
			me, bucket, lookMsg := getCallContext(callContext)
			_ = me
			watchedTopic, ok := getWatcher(bucket, &lookMsg.topicHash)
			if !ok {
				// checkMongo
				str := lookMsg.topicHash.ToBase64()
				watchedTopic, ok = GetSubscription(str)
				if !ok {
					// don't make a new one
					return "error: topic not found"
				}
				setWatcher(bucket, &lookMsg.topicHash, watchedTopic)
			}
			key = strings.ToUpper(key)
			watchedTopic.SetOption(key, val)
			// save to mongo !
			if changed {
				SaveSubscription(watchedTopic)
			}
			return "ok"
		}, c.CommandMap)

	monitor_pod.MakeCommand("reserve",
		"assign a public key to a name", 0,
		func(msg string, args []string, callContext interface{}) string {
			changed := false
			me, bucket, lookMsg := getCallContext(callContext)
			pubk, ok := lookMsg.p.GetOption("pubk")
			if !ok {
				return "error pubk not found"
			}
			watchedTopic, ok := getWatcher(bucket, &lookMsg.topicHash)
			if !ok {
				// checkMongo
				str := lookMsg.topicHash.ToBase64()
				watchedTopic, ok = GetSubscription(str)
				if !ok {
					changed = true
					watchedTopic = &WatchedTopic{}
					watchedTopic.Name = lookMsg.topicHash
					watchedTopic.thetree = NewWithInt64Comparator()
					// watchedTopic.Expires = 20*60 + me.getTime() this should match a token
					nameStr, ok := lookMsg.p.GetOption("name")
					if ok {
						watchedTopic.NameStr = string(nameStr)
					}

					t, _ := lookMsg.p.GetOption("jwtid") // don't they ALL have this?, except billing topics
					if len(t) != 0 {                     // it's always 64 bytes binary
						watchedTopic.Jwtid = string(t)
					}
					setWatcher(bucket, &lookMsg.topicHash, watchedTopic)
					TopicsAdded.Inc()

					now := me.getTime()
					watchedTopic.nextBillingTime = now + 30 // 30 seconds to start with
					watchedTopic.lastBillingTime = now
				}
				setWatcher(bucket, &lookMsg.topicHash, watchedTopic)
			}

			watchedTopic.Owners = append(watchedTopic.Owners, string(pubk))
			// save to mongo !
			if changed {
				SaveSubscription(watchedTopic)
			}

			return "ok"
		}, c.CommandMap)

	monitor_pod.MakeCommand("exists",
		"returns true if the name exists 🔓", 0,
		func(msg string, args []string, callContext interface{}) string {
			me, bucket, lookMsg := getCallContext(callContext)
			watchedTopic, ok := getWatcher(bucket, &lookMsg.topicHash)
			_ = watchedTopic
			if ok {
				return "true"
			}
			// check mongo
			str := lookMsg.topicHash.ToBase64()
			watchedTopic, ok = GetSubscription(str)
			if ok {
				setWatcher(bucket, &lookMsg.topicHash, watchedTopic)
				return "true"
			}
			_ = me
			return "false"
		}, c.CommandMap)

	monitor_pod.MakeCommand("get time",
		"seconds since 1970🔓", 0,
		func(msg string, args []string, callContext interface{}) string {
			sec := time.Now().UnixMilli() / 1000
			secStr := strconv.FormatInt(sec, 10)
			return secStr
		}, c.CommandMap)
	monitor_pod.MakeCommand("get random",
		"returns a random integer", 0,
		func(msg string, args []string, callContext interface{}) string {
			tmp := rand.Uint32()
			secStr := strconv.FormatInt(int64(tmp), 10)
			return secStr
		}, c.CommandMap)
	monitor_pod.MakeCommand("get pubk",
		"device public key 🔓", 0,
		func(msg string, args []string, callContext interface{}) string {
			//  this is the public key of the cluster
			me, bucket, lookMsg := getCallContext(callContext)
			str := base64.RawURLEncoding.EncodeToString(me.ex.ce.PublicKeyTemp[:])
			_ = bucket
			_ = lookMsg
			return str
		}, c.CommandMap)
	monitor_pod.MakeCommand("version",
		"info about this thing", 0,
		func(msg string, args []string, callContext interface{}) string {
			return "v0.1.6"
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
			return s
		}, c.CommandMap)
}

func decryptCommand(me *LookupTableStruct, p *packets.Lookup, command string) bool {
	ourPrivKey := me.ex.ce.PrivateKeyTemp
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

	out := make([]byte, 0, len(sealed)) // it's actyually smaller
	result, ok := box.Open(out, sealed, &nonce2, &pubk2, ourPrivKey)
	if !ok {
		return false
	}

	// split the payload into command and time
	payload := string(result)
	parts := strings.Split(payload, " ")
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
	cmdtmp := strings.Join(parts[0:len(parts)-1], " ")
	// check the command
	return cmdtmp == command
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
