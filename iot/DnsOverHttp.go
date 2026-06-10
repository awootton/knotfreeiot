package iot

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
)

type DnsAnswer struct {
	Name string `json:"name"`
	Type int    `json:"type"` // 1 for A, 16 for TXT, etc.
	TTL  int    `json:"TTL,omitempty"`
	Data string `json:"data,omitempty"`
}
type DnsQuestion struct {
	Name string `json:"name"`
	Type int    `json:"type"`
}
type DnsResponse struct {
	Status    int           `json:"Status"` // 0 for no error, 3 for name error, etc.
	TC        bool          `json:"TC"`
	RD        bool          `json:"RD"`
	RA        bool          `json:"RA"`
	AD        bool          `json:"AD"`
	CD        bool          `json:"CD"`
	Question  []DnsQuestion `json:"Question"`
	Answer    []DnsAnswer   `json:"Answer,omitempty"`
	Authority []DnsAnswer   `json:"Authority,omitempty"` // when do I do this?

	Comment string `json:"Comment,omitempty"`
}

var dnsResponseCache *expirable.LRU[string, DnsResponse]

func init() {
	// 32K keys that go nowhere and expire after 5 minutes. why 5 min?
	dnsResponseCache = expirable.NewLRU[string, DnsResponse](4 * 1024, nil, time.Second*300)
	// I don't want to make Cloudflair mad at me.
	// I just want to survive an alpha stage 
}

// Status codes for DoH responses (based on DNS response codes):
// 0 (NOERROR): The query was successful, and the response contains the requested IP address or records.
// 1 (FORMERR): The DoH resolver could not interpret the format of the DNS query.
// 2 (SERVFAIL): The DNS server failed to answer the request (often a timeout or issue with the upstream server).
// 3 (NXDOMAIN): Non-Existent Domain. The domain name you queried does not exist.
// 4 (NOTIMP): Not Implemented. The DoH server does not support the requested DNS operation.
// 5 (REFUSED): The DNS server refused to process the request (e.g., due to policy

// TODO: have a cache of non existing names. Those don't change much and they hit the DB every time.
// the downside is a user has to wait to see his new domain exists.

// TODO: we should cache these for 5 minutes.
// At least cache the non existing names, those don't change much and they hit the DB every time.
// The downside is a user has to wait to see his new domain exists, but that's probably acceptable since it's a new domain and it might take some time to propagate anyway.
func LookupDnsOverHttp(domains []string, recordType int, dnsServer string) ([]DnsResponse, error) {

	// dial once for the batch.
	resolver := &net.Resolver{
		PreferGo: true, // Forces the native Go resolver instead of cgo. cgo is probably off.
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{
				Timeout: 3 * time.Second,
			}
			// Route all UDP/TCP queries to the specified DNS server
			return d.DialContext(ctx, network, dnsServer+":53")
		},
	}

	answers := make([]DnsResponse, len(domains))
	var wg sync.WaitGroup

	for i, domain := range domains {
		wg.Add(1)
		go func(i int, domain string) {
			defer wg.Done()

			recordTypeStr := "A"
			if recordType == 16 {
				recordTypeStr = "TXT"
			} else if recordType != 1 {
				recordTypeStr = fmt.Sprintf("%d", recordType)
			}

			cachekey :=  domain + "____" + recordTypeStr
			if cachedResponse, found := dnsResponseCache.Get(cachekey); found {
				fmt.Println("Cache hit for ", cachekey, ": ", cachedResponse)
				answers[i] = cachedResponse
				return
			}
			fmt.Println("LookupDnsOverHttp Resolving ", cachekey) // eg testmain-0n1u1e15p.xyz_1

			response, err := lookupOne(domain, recordType, resolver) // this is just a placeholder for now, we will implement it later. It will query the DoH server and return the response.
			if err != nil {
				// only set the status if it's not already set. This is because lookupOne might return a response with a non-zero status, e.g., for unsupported record type, and we don't want to overwrite that with a generic SERVFAIL.
				if response.Status == 0 {
					// NOT the same as domain doesn't exist. This is an actual error, e.g., timeout, network error, etc.
					response.Status = 2 // SERVFAIL
					if strings.Contains(err.Error(), "no such host") {
						response.Status = 3 // NXDOMAIN
					}
				}
				response.Comment = err.Error()
			}
			dnsResponseCache.Add(cachekey, response)
			answers[i] = response
		}(i, domain)
	}
	wg.Wait()
	return answers, nil
}

// lookupOne will call someones dns over http server. Fail fast.
// Be carefull that errors are NOT the same as domain doesn't exits.
func lookupOne(domain string, recordType int, resolver *net.Resolver) (DnsResponse, error) {
	result := DnsResponse{}
	result.Question = make([]DnsQuestion, 0)
	result.Question = append(result.Question, DnsQuestion{
		Name: domain,
		Type: recordType,
	})
	result.CD = true  // Checking Disabled. We don't care about dnssec, we just want the answer. This is also to prevent some weird edge cases where the dns server might have dnssec issues and return an error instead of the answer.
	result.RD = true  // Recursion Desired. We want the dns server to do the recursion for us, we don't want to do it ourselves.
	result.RA = true  // Recursion Available. We want to know if the dns server supports recursion, if not we might want to try another server or return an error.
	result.AD = false // Authenticated Data. We don't care about dnssec, we just want the answer. This is also to prevent some weird edge cases where the dns server might have dnssec issues and return an error instead of the answer.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var err error
	var ips []string
	switch recordType {
	case 1: // A record
		ips, err = resolver.LookupHost(ctx, domain)
	case 16: // TXT record
		ips, err = resolver.LookupTXT(ctx, domain)
	default:
		result.Status = 4 // NOTIMP
		result.Comment = "Unsupported record type"
		return result, errors.New("Unsupported record type")
	}
	if err != nil {
		result.Status = 2 // SERVFAIL
		if strings.Contains(err.Error(), "no such host") {
			result.Status = 3 // NXDOMAIN
		}
		result.Comment = err.Error()
		return result, err
	}
	// println("Resolved " + domain + " to: " + strings.Join(ips, ", "))
	for _, ip := range ips {
		result.Answer = append(result.Answer, DnsAnswer{
			Name: domain,
			Type: recordType,
			TTL:  300, // we don't know the actual TTL, so we just set it to a default value. This is also to prevent some weird edge cases where the dns server might have issues and return an error instead of the answer.
			Data: ip,
		})
	}
	return result, nil
}
