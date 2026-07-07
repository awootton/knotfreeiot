package iot

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/awootton/knotfreeiot/packets"
	"github.com/awootton/knotfreeiot/tokens"
)

func serveBillingCommand(p *packets.Send, billingAccumulator *BillingAccumulator, seconds uint32) packets.Send {

	command := string(p.Payload)

	isHttp := false
	if strings.HasPrefix(command, `GET /`) {
		isHttp = true
		lines := strings.Split(command, "\n")
		if len(lines) < 1 {
			command = "bad http request"
		} else {
			getline := lines[0]
			getparts := strings.Split(getline, " ")
			if len(getparts) != 3 {
				command = "expected 3 parts to http request "
			} else {
				// now we passed the headers
				command = getparts[1]
				command = strings.Split(command, "?")[0]
				command = strings.TrimPrefix(command, "/")
				command = strings.ReplaceAll(command, "/", " ")
			}
		}
	}

	// fmt.Println(" billing channel has command", command)
	// TODO: make this like the monnitor_pod and lookmsg
	reply := ""
	switch command {
	case "get max":

		bytes, err := json.Marshal(billingAccumulator.Max)
		if err != nil {
			reply = "error fail " + err.Error()
		} else {
			reply = string(bytes)
		}

	case "get stats":
		current := &tokens.KnotFreeContactStats{}
		billingAccumulator.GetStats(seconds, current)
		bytes, err := json.Marshal(current)
		if err != nil {
			reply = "error fail " + err.Error()
		} else {
			reply = string(bytes)
		}
	case "about":
		reply = "billing_v.0.1.2"
	case "get pubk":
		reply = "-none"
	case "get admin hint":
		reply = "-none"
	default: // if command == "help"

		reply += "[get stats] current usage numbers🔓\n"
		reply += "[get max] maximum allowed numbers🔓\n"
		reply += "[get pubk] device public key🔓\n"
		reply += "[get admin hint] the first chars of the admin public keys🔓\n"
		reply += "[about] info on this device\n"
		reply += "[help] lists all commands🔓\n"

	}

	if isHttp {
		tmp := "HTTP/1.1 200 OK\r\n"
		tmp += "Content-Length: "
		tmp += strconv.FormatInt(int64(len(reply)), 10)
		tmp += "\r\n"
		tmp += "Content-Type: text/plain\r\n"
		tmp += "Access-Control-Allow-Origin: *\r\n"
		tmp += "access-control-expose-headers: nonc\r\n"
		tmp += "Connection: Closed\r\n"
		//tmp += "nonc: " + string(nonc) + "\r\n" // this might be redundant
		tmp += "\r\n"
		tmp += reply
		reply = tmp
	}
	pub := packets.Send{}
	pub.Address = p.Source
	pub.Payload = []byte(reply)
	pub.Source = p.Address
	pub.CopyOptions(&p.PacketCommon)

	CheckSendPacket(&pub)

	return pub

}
