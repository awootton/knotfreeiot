package iot

import (
	"encoding/json"

	"github.com/awootton/knotfreeiot/packets"
	"github.com/awootton/knotfreeiot/tokens"
)

func serveBillingCommand(p *packets.Send, billingAccumulator *BillingAccumulator, seconds uint32) packets.Send {

	command := string(p.Payload)
	// fmt.Println(" billing channel has command", command)
	reply := ""
	if command == "get max" {

		bytes, err := json.Marshal(billingAccumulator.max)
		if err != nil {
			reply = "error fail " + err.Error()
		} else {
			reply = string(bytes)
		}

	} else if command == "get stats" {
		current := &tokens.KnotFreeContactStats{}
		billingAccumulator.GetStats(seconds, current)
		bytes, err := json.Marshal(current)
		if err != nil {
			reply = "error fail " + err.Error()
		} else {
			reply = string(bytes)
		}
	} else if command == "about" {
		reply = "billing_v.0.1.2"
	} else if command == "get pubk" {
		reply = "-none"
	} else if command == "get admin hint" {
		reply = "-none"
	} else { // if command == "help"

		reply += "[get stats] current usage numbers\n"
		reply += "[get max] maximum allowed numbers\n"
		reply += "[get pubk] device public key\n"
		reply += "[get admin hint] the first chars of the admin public keys\n"
		reply += "[about] info on this device\n"
		reply += "[help] lists all commands\n"

	}
	pub := packets.Send{}
	pub.Address = p.Source
	pub.Payload = []byte(reply)
	pub.Source = p.Address
	pub.CopyOptions(&p.PacketCommon)

	return pub

}
