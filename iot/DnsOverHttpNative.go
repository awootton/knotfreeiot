package iot

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"errors"

	"github.com/awootton/knotfreeiot/packets"
)

func LookupDnsOverHttpKnotfree(ce *ClusterExecutive, domainstrings []string, recordType int) ([]DnsResponse, error) {
	var wg sync.WaitGroup
	responses := make([]DnsResponse, len(domainstrings))

	for i, domain := range domainstrings {
		wg.Add(1)
		go func(i int, domain string) {
			defer wg.Done()
			response, err := LookupDnsOverHttpKnotfreeOnce(ce, domain, recordType)
			if err != nil {
				fmt.Println("Error looking up DNS over HTTP for domain", domain, ":", err)
			}
			responses[i] = response
		}(i, domain)
	}
	wg.Wait()

	return responses, nil
}

// TODO: we should cache these for 5 minutes.
// At least cache the non existing names, those don't change much and they hit the DB every time.
// The downside is a user has to wait to see his new domain exists, but that's probably acceptable since it's a new domain and it might take some time to propagate anyway.

func LookupDnsOverHttpKnotfreeOnce(ce *ClusterExecutive, domainstring string, recordType int) (DnsResponse, error) {

	resp := DnsResponse{}
	resp.Question = make([]DnsQuestion, 0)
	resp.Question = append(resp.Question, DnsQuestion{
		Name: domainstring,
		Type: recordType,
	})
	resp.CD = true  // Checking Disabled. We don't care about dnssec, we just want the answer. This is also to prevent some weird edge cases where the dns server might have dnssec issues and return an error instead of the answer.
	resp.RD = true  // Recursion Desired. We want the dns server to do the recursion for us, we don't want to do it ourselves.
	resp.RA = false // Recursion Available. We want to know if the dns server supports recursion, if not we might want to try another server or return an error.
	resp.AD = false // Authenticated Data. We don't care about dnssec, we just want the answer. This is also to prevent some weird edge cases where the dns server might have dnssec issues and return an error instead of the answer.
	resp.Status = 0 // NOERROR

	// we have a channel already that runs commands.
	sc := ce.PacketService

	subscriptionName := domainstring

	parts := strings.Split(subscriptionName, ".")
	subkey := ""
	if len(parts) > 2 {
		subkey = parts[0]
		parts = parts[len(parts)-2:]
	}
	subscriptionName = strings.Join(parts, "_")

	typeStr := "A"
	if recordType == 16 {
		typeStr = "TXT"
	} else if recordType != 1 {
		resp.Status = 4 // NOTIMP
		resp.Comment = "Unsupported record type"
		return resp, errors.New("Unsupported record type")
	}

	command := "get option " + typeStr + " " + subkey // eg get option A
	// this has overloaded and crashed the server !
	// fmt.Println("Sending command to knotfree: ", command, " for domain ", domainstring, " with subscription name ", subscriptionName)

	cmd := packets.Lookup{}
	cmd.Address.FromString(subscriptionName)
	cmd.SetOption("cmd", []byte(command))
	// send it
	replyPacket, err := sc.GetPacketReplyLonger(&cmd, 2*time.Second)
	if err != nil {
		fmt.Println("knotfree failed to get from service contact", err)
		resp.Status = 2 // SERVFAIL
		resp.Comment = "Server failure"
		return resp, err
	}
	// fmt.Println("knotfree returned from service contact", replyPacket.Sig())
	sendPacket, ok := replyPacket.(*packets.Send)
	stringReturned := len(sendPacket.Payload) > 0
	if !ok || !stringReturned {
		fmt.Println("knotfree failed to get a valid send packet from service contact", err, replyPacket.Sig())
		resp.Status = 2 // SERVFAIL
		resp.Comment = "Server failure"
		return resp, errors.New("Invalid response from server")
	}
	if stringReturned {
		// fmt.Println("knotfree returned message ", string(sendPacket.Payload))
		// it's just a string. It's not json.
		resp.Answer = []DnsAnswer{
			{
				Name: domainstring,
				Type: recordType,
				Data: string(sendPacket.Payload),
				TTL:  300, // we do NOT know what this really is.
			},
		}
		resp.Status = 0 // NOERROR
		// TODO: Use a better way in case someone actually puts the string "error:" in their dns txt record. Maybe use a prefix or something. But for now this is fine.
		if strings.HasPrefix(string(sendPacket.Payload), "error: not found") || strings.Contains(string(sendPacket.Payload), "error:") {
			resp.Status = 3 // NXDOMAIN
			resp.Comment = string(sendPacket.Payload)
			// this is NOT an error. It's a not found.
		}
		return resp, nil

	} else {
		fmt.Println("knotfree returned empty response", replyPacket.Sig())
		resp.Status = 2 // SERVFAIL
		resp.Comment = "Empty response from server"
		return resp, errors.New("Empty response from server")
	}
}
