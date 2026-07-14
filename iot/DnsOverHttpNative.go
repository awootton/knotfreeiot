package iot

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"errors"

	"github.com/awootton/knotfreeiot/packets"

	"github.com/hashicorp/golang-lru/v2/expirable"
)

var dnsResponseCacheN *expirable.LRU[string, DnsResponse]

func init() {
	// 32K keys that go nowhere and expire after 5 minutes. why 5 min?
	dnsResponseCacheN = expirable.NewLRU[string, DnsResponse](32*1024, nil, time.Second*300)
	// I don't want to make mongodb atlas mad at me.
	// I just want to survive an alpha stage
}

// LookupDnsOverHttpKnotfree will look up the given domain strings and record type using knotfree.
// in knotfree a domain name is the same thing as a subscription name.
// So we will send a command to knotfree to get the option for that subscription name and record type.
// The response will be the answer to the dns query. We will cache the responses for 5 minutes to avoid hitting the database too much.
// We have a perpetually open Contact to knotfree that we can send commands to. We use the lookup api in lookmsg.go to do this.
// The command we send is "get option A subkey" where subkey is the first part of the domain name if it has more than 2 parts, otherwise it's empty.
// For example, for "testmain-0n1u1e15p.xyz" the subkey is "testmain-0n1u1e15p" and for "example.com" the subkey is "".
// We will send a command like "get option A subkey" where subkey is the first part of the domain name if it has more than 2 parts, otherwise it's empty. For example, for "testmain-0n1u1e15p.xyz" the subkey is "testmain-0n1u1e15p" and for "example.com" the subkey is "".
// Note that the "get option" command does not require the request to be signed with the owners key.

func LookupDnsOverHttpKnotfree(ce *ClusterExecutive, domainstrings []string, recordType int) ([]DnsResponse, error) {
	var wg sync.WaitGroup
	responses := make([]DnsResponse, len(domainstrings))

	log.Println("LookupDnsOverHttpKnotfree starting, looking up ", len(domainstrings), " domains with record type ", recordType)
	defer log.Println("LookupDnsOverHttpKnotfree found ", len(responses), " domains with record type ", recordType)

	for i, domain := range domainstrings {

		// now that we have an entire array of service-contacts at our disposal, we should probably do these in parallel.
		// But there are some bugs that need to be worked out first and I DON'T have TIME today.

		wg.Add(1)
		go func(i int, domain string) {

			defer wg.Done()

			typeStr := "A"
			if recordType == 16 {
				typeStr = "TXT"
			} else if recordType != 1 {
				typeStr = fmt.Sprintf("%d", recordType)
			}

			cachekey := domain + "____" + typeStr

			// log.Println("LookupDnsOverHttpKnotfree Cache check for ", cachekey)

			if cachedResponse, found := dnsResponseCacheN.Get(cachekey); found {
				// log.Println("LookupDnsOverHttpKnotfree Cache hit for ", cachekey, ": ", cachedResponse)
				responses[i] = cachedResponse
				return
			}
			// log.Println("LookupDnsOverHttpKnotfree Resolving ", cachekey) // eg testmain-0n1u1e15p.xyz_1

			response, err := LookupDnsOverHttpKnotfreeOnce(ce, domain, recordType)
			if err != nil {
				log.Println("Error looking up DNS over HTTP for domain", domain, ":", err)
				// response status will be 2
			}

			// log.Println("LookupDnsOverHttpKnotfree Response for ", cachekey, ": ", response.Status)

			// only add the 0 and the 3's to the cache. The 2's are probably temporary errors and the 4's are unsupported record types that we don't want to cache.
			if response.Status == 0 || response.Status == 3 {
				dnsResponseCacheN.Add(cachekey, response)
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

// if this worked properly we should do them all at the same time. but there's bugs.
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
	// log.Println("Sending command to knotfree: ", command, " for domain ", domainstring, " with subscription name ", subscriptionName)

	cmd := packets.Lookup{}
	cmd.Address.FromString(subscriptionName)
	cmd.SetOption("cmd", []byte(command))
	// send it, now long. No really. When it's working it's fast.
	replyPacket, err := ce.GetPacketService().GetPacketReplyLonger(&cmd, 5*time.Second) // 2*time.Second)

	if err != nil {
		log.Println("LookupDnsOverHttpKnotfree to get from service contact", err)
		resp.Status = 2 // SERVFAIL
		resp.Comment = "Server failure"
		return resp, err
	}

	sendPacket, ok := replyPacket.(*packets.Send)
	CheckSendPacket(sendPacket)
	stringReturned := len(sendPacket.Payload) > 0
	if !ok || !stringReturned {
		// weird
		log.Println("LookupDnsOverHttpKnotfree failed to get a valid send packet from service contact", err, replyPacket.Sig())
		resp.Status = 2 // SERVFAIL
		resp.Comment = "Server failure"
		return resp, errors.New("Invalid response from server")
	}
	if stringReturned {
		// either 216.128.128.195
		// or topic not found errid=bvBbhJawYXIMWsxJOWHt
		// or maybe empty string?
		//log.Println("LookupDnsOverHttpKnotfree returned message ", string(sendPacket.Payload))
		// it's just a string. It's not json.
		resp.Answer = []DnsAnswer{
			{
				Name: domainstring,
				Type: recordType,
				Data: string(sendPacket.Payload),
				TTL:  300, // we do NOT know what this actually is. So, we lie.
			},
		}
		resp.Status = 0 // NOERROR
		// an awkward api
		errorIdOfNotFound := "bvBbhJawYXIMWsxJOWHt"
		if strings.Contains(string(sendPacket.Payload), errorIdOfNotFound) {
			resp.Status = 3 // NXDOMAIN
			resp.Comment = string(sendPacket.Payload)
		}
		return resp, nil

	} else {
		log.Println("LookupDnsOverHttpKnotfree returned empty response", replyPacket.Sig())
		resp.Status = 2 // SERVFAIL
		resp.Comment = "Empty response from server"
		return resp, errors.New("Empty response from server")
	}
}
