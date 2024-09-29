package iot

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/awootton/knotfreeiot/packets"
	"github.com/awootton/knotfreeiot/tokens"
	"golang.org/x/crypto/nacl/box"
)

func (api ApiHandler) NameService(w http.ResponseWriter, req *http.Request) {
	// This function will forward signed requests to the lookup service api.

	var err error

	command := req.URL.Query().Get("cmd")   // this the command in the clear
	sealed := req.URL.Query().Get("sealed") // this is in b63 and was signed by the sender with owner's private key
	nonceStr := req.URL.Query().Get("nonce")

	theirPubk := req.URL.Query().Get("pubk") // this has to be the owners public key of the name
	aName := req.URL.Query().Get("name")     // this has to be text name of the subscription involved

	if len(aName) == 0 {
		http.Error(w, "no name provided", 500)
		return
	}

	fmt.Println("NameService command", command)
	fmt.Println("NameService theirPubk", theirPubk)

	look := packets.Lookup{}
	look.Address.FromString(aName)
	look.SetOption("cmd", []byte(command))
	look.SetOption("pubk", []byte(theirPubk))
	look.SetOption("nonc", []byte(nonceStr)) // raw nonce

	binSealed, err := base64.RawURLEncoding.DecodeString(sealed)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	look.SetOption("sealed", binSealed)

	// send it
	var reply packets.Interface
	if DEBUG {
		// have a little more time for debugging
		reply, err = api.ce.PacketService.GetPacketReplyLonger(&look, time.Duration(50*time.Second))
	} else {
		reply, err = api.ce.PacketService.GetPacketReply(&look)
	}
	if err != nil {
		http.Error(w, "nameservice:"+err.Error(), 500)
		return
	}
	replyBytes := reply.(*packets.Send).Payload

	fmt.Println("NameService reply", string(replyBytes))
	// this is httpd so no need to encrypt the reply
	// the encrypt of the send was to prove ownership
	n, err := w.Write(replyBytes)
	if err != nil {
		fmt.Println("NameService write error", err)
		http.Error(w, err.Error(), 500) // does this work?
		return
	}
	if n != len(replyBytes) {
		fmt.Println("NameService write error", n, len(replyBytes))
	}
}

// createNameFunc is a callback function for the lookup service for "reserve"
// See lookmsg.go
// It is called when a new name is created or to change/set the owner of a name.
// note that we can change the exp and the jwtid
func createNameFunc(msg string, args []string, callContext interface{}) string {
	// reserve name tok
	me, bucket, lookMsg, pubk := getCallContext(callContext)

	if len(args) < 2 {
		fmt.Println("add name error: too few args")
		sendReply(me, lookMsg, "add name error: too few args")
		return ""
	}
	name := args[0]
	token := args[1]
	payload, err := tokens.ValidateToken(string(token))
	if err != nil {
		fmt.Println("add name error: invalid token", err)
		sendReply(me, lookMsg, "add name error: invalid token")
		return ""
	}
	if payload.Pubk != pubk {
		fmt.Println("add name error: not owner")
		sendReply(me, lookMsg, "add name error: not owner")
		return ""
	}
	watchedTopic, watcherExisted := getWatcher(bucket, &lookMsg.topicHash)

	// now we must release the bucket
	func() {
		// and we are async now
		count, err := GetSubscriptionListCount(payload.Pubk)
		if err != nil {
			fmt.Println("getNames GetSubscriptionList", err)
			// http.Error(w, err.Error(), 500)
			sendReply(me, lookMsg, "error GetSubscriptionList")
			return
		}
		if count+1 > int(payload.Subscriptions) {
			fmt.Println("add name error: too many names")
			sendReply(me, lookMsg, "add name error: you own more names than allowed by your token")
			return
		}
		// now check if exists ? we don't need to do this?
		if watcherExisted {

			// we can't change the owner of a name.
			// What if we want to change the owner of a name?
			// TODO: make another command for this.
			if watchedTopic.Owner != payload.Pubk {
				fmt.Println("add name error: not owner")
				sendReply(me, lookMsg, "add name error: not owner")
				return
			}
			fmt.Println("add name: name already exists", name)
			watchedTopic.Expires = payload.ExpirationTime
			watchedTopic.Jwtid = payload.JWTID
			watchedTopic.Owner = payload.Pubk
			err = SaveSubscription(watchedTopic)
			if err != nil {
				fmt.Println("add name: save subscription err", err)
			}
		} else {
			// try to pull it first
			gotwatchedTopic, ok := GetSubscription(lookMsg.topicHash.ToBase64())
			if ok {
				if gotwatchedTopic.Owner != pubk {
					fmt.Println("add name error: not owner", name)
					sendReply(me, lookMsg, "add name error: not owner")
					return
				}
				fmt.Println("add name: name already exists")
				gotwatchedTopic.Expires = payload.ExpirationTime
				gotwatchedTopic.Jwtid = payload.JWTID
				err = SaveSubscription(gotwatchedTopic)
				if err != nil {
					fmt.Println("add name: save subscription err", err)
				}
				watchedTopic = gotwatchedTopic
			} else {
				// make the watchedItem and mongo insert
				newWatchedTopic := WatchedTopic{
					Name:              lookMsg.topicHash,
					NameStr:           name,
					Expires:           payload.ExpirationTime,
					thetree:           NewWithInt64Comparator(),
					OptionalKeyValues: nil,
					Bill:              nil,
					Jwtid:             payload.JWTID,
					Owner:             payload.Pubk,
				}
				err = SaveSubscription(&newWatchedTopic)
				if err != nil {
					fmt.Println("add name error: save subscription2", err)
				}
				watchedTopic = &newWatchedTopic
			}

		}
		// now reaquire the bucket and replace the watcher.
		mmm := lookBackCommand{
			callContext: callContext,
			callback: func(me *LookupTableStruct, bucket *subscribeBucket, cmd *callBackCommand) {
				// remember it
				setWatcher(bucket, &lookMsg.topicHash, watchedTopic)
				// and send the reply
				sendReply(me, lookMsg, "ok")
			},
		}
		bucket.incoming <- &mmm
	}()
	return ""
}

func deleteNameFunc(msg string, args []string, callContext interface{}) string {
	// reserve name tok
	me, bucket, lookMsg, pubk := getCallContext(callContext)
	_ = bucket
	if len(args) < 1 {
		fmt.Println("delete name error: too few args")
		sendReply(me, lookMsg, "delete name error: too few args")
		return ""
	}
	name := args[0]
	_ = name

	watchedTopic, watcherExisted := getWatcher(bucket, &lookMsg.topicHash)
	// now we must release the bucket
	func() {
		// and we are async now
		// now check if exists ? we don't need to do this?
		// we NEVER delete without loading first and checking the owner.
		if watcherExisted {
			if watchedTopic.Owner != pubk {
				fmt.Println("delete name error: not owner", name)
				sendReply(me, lookMsg, "delete name error: not owner")
				return
			}
			var hashed HashType
			hashed.HashString(name)
			hashedStr := hashed.ToBase64() // should we check this more often?
			if hashed != lookMsg.topicHash || hashed != watchedTopic.Name {
				sendReply(me, lookMsg, "delete name error: hash mismatch")
				return
			}
			err := DeleteSubscription(hashedStr)
			if err != nil {
				fmt.Println("add name error: save subscription", err)
			}
		} else {
			gotwatchedTopic, ok := GetSubscription(lookMsg.topicHash.ToBase64())
			if !ok {
				// is this really an error?
				fmt.Println("delete name error: not found", name)
				sendReply(me, lookMsg, "delete name error: not found")
				return
			}
			if gotwatchedTopic.Owner != pubk {
				fmt.Println("delete name error: not owner", name)
				sendReply(me, lookMsg, "delete name error: not owner")
				return
			}
			var hashed HashType
			hashed.HashString(name)
			hashedStr := hashed.ToBase64() // should we check this more often?
			if hashed != lookMsg.topicHash || hashed != gotwatchedTopic.Name {
				sendReply(me, lookMsg, "delete name error: hash mismatch")
				return
			}
			err := DeleteSubscription(hashedStr)
			if err != nil {
				fmt.Println("add name error: save subscription", err)
			}
		}
		// now reaquire the bucket and replace the watcher.
		// with nil which is a delete.
		// what if it has subscribers?
		mmm := lookBackCommand{
			callContext: callContext,
			callback: func(me *LookupTableStruct, bucket *subscribeBucket, cmd *callBackCommand) {
				// remember it
				setWatcher(bucket, &lookMsg.topicHash, nil)
				// and send the reply
				sendReply(me, lookMsg, "ok")
			},
		}
		bucket.incoming <- &mmm
	}()
	return ""
}

// func (api ApiHandler) XXXaddName(w http.ResponseWriter, req *http.Request, cmdParts []string) (string, error) {

// 	_ = w
// 	_ = req

// 	// cmdParts[1] is the name to add
// 	// cmdParts[2] is a token
// 	if len(cmdParts) < 3 {
// 		fmt.Println("add name error: no name provided")
// 		return "add name error: no name provided", errors.New("no name provided")
// 	}
// 	name := cmdParts[1]
// 	token := cmdParts[2]
// 	// validate token
// 	payload, err := tokens.ValidateToken(string(token))
// 	if err != nil {
// 		fmt.Println("add name error: invalid token", err)
// 		return "add name error: invalid token", err
// 	}
// 	// now count the names for this owner
// 	list, err := GetSubscriptionList(payload.Pubk) // TODO: just get the count
// 	if err != nil {
// 		fmt.Println("getNames GetSubscriptionList", err)
// 		// http.Error(w, err.Error(), 500)
// 		return "error GetSubscriptionList", err
// 	}
// 	if len(list)+1 > int(payload.Subscriptions) {
// 		fmt.Println("add name error: too many names")
// 		return "add name error: too many names", errors.New("too many names")
// 	}

// 	// now check if exists
// 	look := packets.Lookup{}
// 	look.Address.FromString(name)
// 	look.SetOption("cmd", []byte("exists"))
// 	val, err := api.ce.PacketService.GetPacketReply(&look)
// 	if err != nil {
// 		// http.Error(w, err.Error(), 500)
// 		return "add name error: ", err
// 	}
// 	if val == nil {
// 		// http.Error(w, "no reply", 500)
// 		return "add name error: no reply", errors.New("no reply")
// 	}
// 	got := string(val.(*packets.Send).Payload)
// 	exists := LookupNameExistsReturnType{false, false}
// 	err = json.Unmarshal([]byte(got), &exists)
// 	if err != nil {
// 		fmt.Println("add name error: json unmarshal", err)
// 		return "add name error: json unmarshal", err
// 	}
// 	if exists.Exists {
// 		fmt.Println("add name error: name already exists")
// 		return "add name error: name already exists", errors.New("name already exists")
// 	}

// 	// make the watchedItem and mongo insert
// 	var h HashType
// 	h.HashString(name)

// 	watchedTopic := WatchedTopic{
// 		Name:              h,
// 		NameStr:           name,
// 		Expires:           payload.ExpirationTime,
// 		thetree:           NewWithInt64Comparator(),
// 		OptionalKeyValues: nil,
// 		Bill:              nil,
// 		Jwtid:             payload.JWTID,
// 		Owner:             payload.Pubk,
// 	}
// 	err = SaveSubscription(&watchedTopic)
// 	// did it work?
// 	if err != nil {
// 		fmt.Println("add name error: save subscription", err)
// 		return "add name error: save subscription", err
// 	}

// 	reply := "ok"
// 	return reply, nil
// }

// func (api ApiHandler) XXXdeleteName(w http.ResponseWriter, req *http.Request, cmdParts []string, theirPubk string) (string, error) {

// 	_ = w
// 	_ = req

// 	// cmdParts[1] is the name to delete

// 	if len(cmdParts) < 2 {
// 		fmt.Println("delete name error: no name provided")
// 		return "delete name error: no name provided", errors.New("no name provided")
// 	}
// 	name := cmdParts[1]

// 	// now check if it's online and delete
// 	nonceStr := []byte(tokens.GetRandomB36String())
// 	nonce := new([24]byte)
// 	copy(nonce[:], nonceStr[:])

// 	timeStr := strconv.FormatInt(time.Now().Unix(), 10)

// 	command := "delete"
// 	cmd := packets.Lookup{}
// 	cmd.Address.FromString(name)
// 	cmd.SetOption("cmd", []byte(command))
// 	cmd.SetOption("pubk", []byte(theirPubk))
// 	cmd.SetOption("nonc", nonce[:]) // raw nonce

// 	// we need to sign this
// 	payload := command + "#" + timeStr

// 	var privk [32]byte
// 	var devicePublicKey [32]byte // FIXME: dummy code

// 	buffer := make([]byte, 0, (len(payload) + box.Overhead))
// 	sealed := box.Seal(buffer, []byte(payload), nonce, &devicePublicKey, &privk)
// 	cmd.SetOption("sealed", sealed)

// 	val, err := api.ce.PacketService.GetPacketReply(&cmd)
// 	if err != nil {
// 		// http.Error(w, err.Error(), 500)
// 		return "delete name error: ", err
// 	}
// 	if val == nil {
// 		// http.Error(w, "no reply", 500)
// 		return "delete name error: no reply", errors.New("no reply")
// 	}
// 	got := string(val.(*packets.Send).Payload)
// 	exists := LookupNameExistsReturnType{false, false}
// 	err = json.Unmarshal([]byte(got), &exists)
// 	if err != nil {
// 		fmt.Println("add name error: json unmarshal", err)
// 		return "add name error: json unmarshal", err
// 	}
// 	if exists.Online {
// 		fmt.Println("add name error: name already exists")
// 		return "add name error: name already exists", errors.New("name already exists")
// 	}

// 	reply := "ok"
// 	return reply, nil
// }

func (api ApiHandler) NameServiceOLD(w http.ResponseWriter, req *http.Request) {
	// This function will forward signed requests to the lookup service api.

	cmd := req.URL.Query().Get("cmd")
	nonceStr := req.URL.Query().Get("nonce")

	theirPubk := req.URL.Query().Get("pubk") // this has to be the owners public key of the name
	//	aName := req.URL.Query().Get("name")     // this has to be text name of the subscription involved

	fmt.Println("NameService cmd", cmd)
	fmt.Println("NameService theirPubk", theirPubk)

	// we need to unbox this
	bincmd, err := base64.RawURLEncoding.DecodeString(cmd)
	if err != nil {
		fmt.Println("NameService decode cmd", err)
		http.Error(w, err.Error(), 500)
		return
	}
	nonce := new([24]byte)
	copy(nonce[:], nonceStr[:])
	openbuffer := make([]byte, 0, (len(cmd))) // - box.Overhead))
	tmp, err := base64.RawURLEncoding.DecodeString(theirPubk)
	if err != nil {
		fmt.Println("NameService decode pubk", err)
		http.Error(w, err.Error(), 500)
		return
	}
	pubk := new([32]byte)
	copy(pubk[:], tmp[:])
	opened, ok := box.Open(openbuffer, bincmd, nonce, pubk, api.ce.PrivateKeyTemp)
	if !ok {
		fmt.Println("NameService box open failed", nonceStr, theirPubk, api.ce.PrivateKeyTemp)
		http.Error(w, "box open failed", 500)
		return
	}
	parts := strings.Split(string(opened), "#")
	if len(parts) != 2 {
		fmt.Println("NameService parts len != 2")
		http.Error(w, "parts len != 2", 500)
		return
	}
	// if parts[0] != theirPubk {
	// 	fmt.Println("pubk not match")
	// 	http.Error(w, "pubk not match", 500)
	// 	return
	// }
	timeStr := parts[1]
	seconds, err := strconv.ParseInt(timeStr, 10, 64)
	if err != nil {
		fmt.Println("time not int")
		http.Error(w, "time not int", 500)
		return
	}
	delta := time.Now().Unix() - seconds
	if delta < 0 {
		delta = -delta
	}
	if delta > 10 {
		fmt.Println("time not match")
		http.Error(w, "time not match", 500)
		return
	}
	cmd = parts[0] // the command is the first part

	// eg delete name
	// add name
	// modify name

	cmdparts := strings.Split(cmd, " ")
	fmt.Println("getNames cmdparts", cmdparts)
	if len(cmdparts) < 1 {
		fmt.Println("getNames cmdparts len < 1")
		http.Error(w, "cmdparts len < 1", 500)
		return
	}
	reply := "error unknown command" + cmd // fixme

	// if cmdparts[0] == "delete" {
	// 	reply, err = api.deleteName(w, req, cmdparts, theirPubk)
	// } else if cmdparts[0] == "add" {
	// 	// add name
	// 	reply, err = api.addName(w, req, cmdparts)
	// } else if cmdparts[0] == "change" {
	// 	// change name
	// 	reply, err = "change name not implemented", errors.New("not implemented")
	// }
	// if err != nil {
	// 	http.Error(w, reply, 500)
	// 	return
	// }

	// this is httpd so no need to encrypt the reply
	// the encrypt of the send was to prove ownership
	w.Write([]byte(reply))

	// now we must encrypt the answer
	// add the time?

	// payload := string(reply)
	// buffer := make([]byte, 0, (len(payload) + box.Overhead))
	// privk := api.ce.PrivateKeyTemp
	// devicePublicKey := pubk
	// sealed := box.Seal(buffer, []byte(payload), nonce, devicePublicKey, privk)

	// sealedb64 := base64.RawURLEncoding.EncodeToString(sealed) // agile rules say no binary
	// w.Write([]byte(sealedb64))
}
